package process

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superdurable/dex/sdk-go/dex"
)

var (
	ErrUnknownFlow       = errors.New("unknown flow")
	ErrDuplicateApproval = errors.New("approval is not pending")
)

type View struct {
	FlowID        string
	Title         string
	State         State
	ReminderCount int64
	Result        string
}

type Service struct {
	client *dex.Client
	flow   BasicProcessFlow
}

func NewService(client *dex.Client) *Service {
	return &Service{client: client, flow: BasicProcess}
}

func (service *Service) Start(ctx context.Context, title string) (View, error) {
	flowID := "process-" + uuid.NewString()
	input := Input{Title: title}
	if _, err := service.client.StartFlow(ctx, service.flow, flowID, input, dex.StartFlowOptions{IDReusePolicy: dex.IDReuseDisallow}); err != nil {
		return View{}, err
	}
	return View{FlowID: flowID, Title: title, State: StateStarted}, nil
}

func (service *Service) Get(ctx context.Context, flowID string) (View, error) {
	var snapshot Snapshot
	err := service.client.InvokeRPC(ctx, flowID, service.flow.DescribeProcess, nil, &snapshot)
	if err == nil {
		return snapshotView(flowID, snapshot), nil
	}
	var inactive *dex.FlowNotActiveError
	if !errors.As(err, &inactive) {
		var missing *dex.FlowNotFoundError
		if errors.As(err, &missing) {
			return View{}, ErrUnknownFlow
		}
		return View{}, err
	}
	result, err := service.client.WaitForFlow(ctx, flowID, dex.WaitForFlowOptions{NeedsResults: true})
	if err != nil {
		var missing *dex.FlowNotFoundError
		if errors.As(err, &missing) {
			return View{}, ErrUnknownFlow
		}
		return View{}, err
	}
	if result.Status != dex.FlowCompleted {
		return View{}, fmt.Errorf("flow ended with status %s", result.Status)
	}
	var output Result
	if err := result.DecodeSingleOutput(&output); err != nil {
		return View{}, err
	}
	return View{FlowID: flowID, Title: output.Title, State: output.State, ReminderCount: output.ReminderCount, Result: output.Message}, nil
}

func (service *Service) Approve(ctx context.Context, flowID string) (View, error) {
	var output dex.None
	err := service.client.InvokeRPC(ctx, flowID, service.flow.ApproveProcess, nil, &output)
	if err == nil {
		return service.Get(ctx, flowID)
	}
	var inactive *dex.FlowNotActiveError
	if errors.As(err, &inactive) {
		view, getErr := service.Get(ctx, flowID)
		if errors.Is(getErr, ErrUnknownFlow) {
			return View{}, ErrUnknownFlow
		}
		if getErr == nil && view.State == StateCompleted {
			return View{}, ErrDuplicateApproval
		}
		return View{}, err
	}
	var worker *dex.WorkerInvocationError
	if errors.As(err, &worker) && worker.Detail == duplicateApprovalDetail {
		return View{}, ErrDuplicateApproval
	}
	return View{}, err
}

func snapshotView(flowID string, snapshot Snapshot) View {
	view := View{FlowID: flowID, Title: snapshot.Title, State: snapshot.State, ReminderCount: snapshot.ReminderCount}
	if snapshot.State == StateCompleted {
		view.Result = completionMessage
	}
	return view
}
