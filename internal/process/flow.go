package process

import (
	"fmt"
	"strings"
	"time"

	"github.com/superdurable/dex/sdk-go/dex"
)

const (
	ReminderInterval         = 15 * time.Minute
	ReminderTimerConditionID = "approval-reminder"
	completionMessage        = "approved automation completed"
	duplicateApprovalDetail  = "approval is not pending"
)

type State string

const (
	StateStarted            State = "started"
	StateValidated          State = "validated"
	StateWaitingForApproval State = "waiting_for_approval"
	StateReminderEmitted    State = "reminder_emitted"
	StateApproved           State = "approved"
	StateExecuting          State = "executing"
	StateCompleted          State = "completed"
)

type Input struct {
	Title string `json:"title"`
}

type Result struct {
	Title         string `json:"title"`
	State         State  `json:"state"`
	ReminderCount int64  `json:"reminderCount"`
	Message       string `json:"message"`
}

type Snapshot struct {
	Title         string `json:"title"`
	State         State  `json:"state"`
	ReminderCount int64  `json:"reminderCount"`
}

var (
	ProcessTitle         = dex.DefineAttribute[string]("process-title")
	ProcessState         = dex.DefineAttribute[State]("process-state")
	ProcessReminderCount = dex.DefineAttribute[int64]("process-reminder-count")
	ProcessApprovals     = dex.DefineChannel[bool]("process-approvals")
)

type StartProcess struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (StartProcess) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	if err := ProcessTitle.Set(ctx, strings.TrimSpace(input.Title)); err != nil {
		return nil, err
	}
	if err := ProcessReminderCount.Set(ctx, 0); err != nil {
		return nil, err
	}
	if err := ProcessState.Set(ctx, StateStarted); err != nil {
		return nil, err
	}
	return dex.GoTo(ValidateRequest{}, input), nil
}

type ValidateRequest struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (ValidateRequest) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || len(title) > 120 {
		return nil, fmt.Errorf("title must contain between 1 and 120 bytes")
	}
	if err := ProcessState.Set(ctx, StateValidated); err != nil {
		return nil, err
	}
	return dex.GoTo(WaitForApproval{}, input), nil
}

type WaitForApproval struct{ dex.StepDefaults }

func (WaitForApproval) WaitFor(ctx dex.Context, _ Input) (*dex.Wait, error) {
	if err := ProcessState.Set(ctx, StateWaitingForApproval); err != nil {
		return nil, err
	}
	return dex.AnyOf(
		ProcessApprovals.ForOne(),
		dex.Timer(ReminderInterval, dex.WithConditionID(ReminderTimerConditionID)),
	), nil
}

func (WaitForApproval) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	approvals, err := ProcessApprovals.GetConditionResults(ctx)
	if err != nil {
		return nil, err
	}
	if len(approvals) == 1 && approvals[0] {
		if err := ProcessState.Set(ctx, StateApproved); err != nil {
			return nil, err
		}
		return dex.GoTo(ExecuteApprovedAutomation{}, input), nil
	}
	if ctx.HasTimerFired() {
		return dex.GoTo(EmitReminder{}, input), nil
	}
	return nil, fmt.Errorf("approval wait completed without approval or reminder")
}

type EmitReminder struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (EmitReminder) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	count, err := ProcessReminderCount.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := ProcessReminderCount.Set(ctx, count+1); err != nil {
		return nil, err
	}
	if err := ProcessState.Set(ctx, StateReminderEmitted); err != nil {
		return nil, err
	}
	return dex.GoTo(WaitForApproval{}, input), nil
}

type ExecuteApprovedAutomation struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (ExecuteApprovedAutomation) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	if err := ProcessState.Set(ctx, StateExecuting); err != nil {
		return nil, err
	}
	return dex.GoTo(CompleteProcess{}, input), nil
}

type CompleteProcess struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (CompleteProcess) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	count, err := ProcessReminderCount.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := ProcessState.Set(ctx, StateCompleted); err != nil {
		return nil, err
	}
	return dex.GracefulComplete(Result{
		Title: strings.TrimSpace(input.Title), State: StateCompleted,
		ReminderCount: count, Message: completionMessage,
	}), nil
}

type BasicProcessFlow struct{ dex.FlowDefaults }

func (BasicProcessFlow) GetSteps() []dex.StepDef {
	return []dex.StepDef{
		dex.DefineStartStep(StartProcess{}),
		dex.DefineStep(ValidateRequest{}),
		dex.DefineStep(WaitForApproval{}),
		dex.DefineStep(EmitReminder{}),
		dex.DefineStep(ExecuteApprovedAutomation{}),
		dex.DefineStep(CompleteProcess{}),
	}
}

func (flow BasicProcessFlow) GetRPCs() []dex.RPCDef {
	return []dex.RPCDef{
		dex.DefineRPC(flow.DescribeProcess, &dex.RPCOptions{}),
		dex.DefineRPC(flow.ApproveProcess, &dex.RPCOptions{
			IsTransactional: true,
			LockAttributes: []dex.AttributeLock{
				dex.LockAttribute(ProcessState),
				dex.LockAttribute(ProcessReminderCount),
			},
		}),
	}
}

func (BasicProcessFlow) GetPersistenceSchema() dex.PersistenceSchema {
	return dex.PersistenceSchema{
		Attributes: []dex.AttributeDef{ProcessTitle, ProcessState, ProcessReminderCount},
		Channels:   []dex.ChannelDef{ProcessApprovals},
	}
}

func (BasicProcessFlow) DescribeProcess(ctx dex.Context, _ dex.None) (*dex.RPCResult[Snapshot], error) {
	title, err := ProcessTitle.Get(ctx)
	if err != nil {
		return nil, err
	}
	state, err := ProcessState.Get(ctx)
	if err != nil {
		return nil, err
	}
	count, err := ProcessReminderCount.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &dex.RPCResult[Snapshot]{Output: Snapshot{Title: title, State: state, ReminderCount: count}}, nil
}

func (BasicProcessFlow) ApproveProcess(ctx dex.Context, approved bool) (*dex.RPCResult[Snapshot], error) {
	state, err := ProcessState.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !approved || (state != StateWaitingForApproval && state != StateReminderEmitted) {
		return nil, fmt.Errorf("%s", duplicateApprovalDetail)
	}
	title, err := ProcessTitle.Get(ctx)
	if err != nil {
		return nil, err
	}
	count, err := ProcessReminderCount.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := ProcessState.Set(ctx, StateApproved); err != nil {
		return nil, err
	}
	if err := ProcessApprovals.Publish(ctx, true); err != nil {
		return nil, err
	}
	return &dex.RPCResult[Snapshot]{Output: Snapshot{Title: title, State: StateApproved, ReminderCount: count}}, nil
}

var BasicProcess = BasicProcessFlow{}

var _ dex.Flow = BasicProcess
var _ dex.Step[Input] = StartProcess{}
var _ dex.Step[Input] = ValidateRequest{}
var _ dex.Step[Input] = WaitForApproval{}
var _ dex.Step[Input] = EmitReminder{}
var _ dex.Step[Input] = ExecuteApprovedAutomation{}
var _ dex.Step[Input] = CompleteProcess{}
var _ dex.RPC[dex.None, Snapshot] = BasicProcess.DescribeProcess
var _ dex.RPC[bool, Snapshot] = BasicProcess.ApproveProcess
