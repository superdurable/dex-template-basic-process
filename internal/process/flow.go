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
	// dex:indexed-attribute attribute-key:process-title index-key:process-title index-type:fulltext value-type:string description:"Process title"
	ProcessTitle = dex.DefineAttribute[string](
		"process-title",
		dex.Indexed(dex.AttributeIndex{Type: dex.IndexFullText}),
	)
	// dex:indexed-attribute attribute-key:process-state index-key:process-state index-type:keyword value-type:string description:"Current process state"
	ProcessState = dex.DefineAttribute[string](
		"process-state",
		dex.Indexed(dex.AttributeIndex{Type: dex.IndexKeyword}),
	)
	ProcessReminderCount = dex.DefineAttribute[int64]("process-reminder-count")
	ProcessApprovals     = dex.DefineChannel[bool]("process-approvals")
)

// dex:group group-id:intake group-label:"Intake"
// dex:explanation text:"Store the request and initialize its durable process state."
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
	if err := ProcessState.Set(ctx, string(StateStarted)); err != nil {
		return nil, err
	}
	return dex.GoTo(ValidateRequest{}, input), nil
}

// dex:group group-id:intake group-label:"Intake"
// dex:explanation text:"Validate the request before opening it for approval."
type ValidateRequest struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (ValidateRequest) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || len(title) > 120 {
		return nil, fmt.Errorf("title must contain between 1 and 120 bytes")
	}
	if err := ProcessState.Set(ctx, string(StateValidated)); err != nil {
		return nil, err
	}
	if err := ProcessState.Set(ctx, string(StateWaitingForApproval)); err != nil {
		return nil, err
	}
	return dex.GoTo(WaitForApproval{}, input), nil
}

// dex:group group-id:review group-label:"Review"
// dex:explanation text:"Wait durably for an approval or the next reminder deadline."
type WaitForApproval struct{ dex.StepDefaults }

func (WaitForApproval) WaitFor(_ dex.Context, _ Input) (*dex.Wait, error) {
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
		if err := ProcessState.Set(ctx, string(StateApproved)); err != nil {
			return nil, err
		}
		return dex.GoTo(ExecuteApprovedAutomation{}, input), nil
	}
	if ctx.HasTimerFired() {
		return dex.GoTo(EmitReminder{}, input), nil
	}
	return nil, fmt.Errorf("approval wait completed without approval or reminder")
}

// dex:group group-id:review group-label:"Review"
// dex:explanation text:"Record a reminder before returning to the approval wait."
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
	if err := ProcessState.Set(ctx, string(StateReminderEmitted)); err != nil {
		return nil, err
	}
	return dex.GoTo(WaitForApproval{}, input), nil
}

// dex:group group-id:execution group-label:"Execution"
// dex:explanation text:"Execute the automation after a manager approves the request."
type ExecuteApprovedAutomation struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (ExecuteApprovedAutomation) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	if err := ProcessState.Set(ctx, string(StateExecuting)); err != nil {
		return nil, err
	}
	return dex.GoTo(CompleteProcess{}, input), nil
}

// dex:group group-id:close group-label:"Close"
// dex:explanation text:"Complete the process with its final durable result."
type CompleteProcess struct {
	dex.StepDefaultsNoWaitFor[Input]
}

func (CompleteProcess) Execute(ctx dex.Context, input Input) (*dex.StepDecision, error) {
	count, err := ProcessReminderCount.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := ProcessState.Set(ctx, string(StateCompleted)); err != nil {
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
		dex.DefineRPC(flow.GetDexSummary, nil),
		dex.DefineRPC(flow.GetDexDisplay, nil),
		dex.DefineRPC(flow.DescribeProcess, &dex.RPCOptions{}),
		dex.DefineRPC(flow.ApproveProcess, &dex.RPCOptions{
			Action: dex.DefineAction(
				"Approve",
				dex.WhenAttributeMatches(
					ProcessState,
					dex.AttributeMatchEqual(string(StateWaitingForApproval)),
					dex.AttributeMatchEqual(string(StateReminderEmitted)),
				),
				dex.ActionRequiresPermission("process.approve"),
			),
			IsTransactional: true,
			LockAttributes: []dex.AttributeLock{
				dex.LockAttribute(ProcessState),
			},
		}),
	}
}

// dex:field attribute-key:process-reminder-count value-type:int64 editable:false description:"Reminder count"
func (BasicProcessFlow) GetDexSummary(ctx dex.Context, _ dex.None) (*dex.RPCResult[map[string]any], error) {
	count, err := ProcessReminderCount.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &dex.RPCResult[map[string]any]{Output: map[string]any{
		"process-reminder-count": count,
	}}, nil
}

// dex:field attribute-key:process-title value-type:string editable:false description:"Process title"
// dex:field attribute-key:process-state value-type:string editable:false description:"Current process state"
// dex:field attribute-key:process-reminder-count value-type:int64 editable:false description:"Reminder count"
func (BasicProcessFlow) GetDexDisplay(ctx dex.Context, _ dex.None) (*dex.RPCResult[map[string]any], error) {
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
	return &dex.RPCResult[map[string]any]{Output: map[string]any{
		"process-title":          title,
		"process-state":          state,
		"process-reminder-count": count,
	}}, nil
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
	return &dex.RPCResult[Snapshot]{Output: Snapshot{Title: title, State: State(state), ReminderCount: count}}, nil
}

func (BasicProcessFlow) ApproveProcess(ctx dex.Context, _ dex.None) (*dex.RPCResult[dex.None], error) {
	state, err := ProcessState.Get(ctx)
	if err != nil {
		return nil, err
	}
	if state != string(StateWaitingForApproval) && state != string(StateReminderEmitted) {
		return nil, fmt.Errorf("%s", duplicateApprovalDetail)
	}
	if err := ProcessState.Set(ctx, string(StateApproved)); err != nil {
		return nil, err
	}
	if err := ProcessApprovals.Publish(ctx, true); err != nil {
		return nil, err
	}
	return &dex.RPCResult[dex.None]{}, nil
}

var BasicProcess = BasicProcessFlow{}

var _ dex.Flow = BasicProcess
var _ dex.Step[Input] = StartProcess{}
var _ dex.Step[Input] = ValidateRequest{}
var _ dex.Step[Input] = WaitForApproval{}
var _ dex.Step[Input] = EmitReminder{}
var _ dex.Step[Input] = ExecuteApprovedAutomation{}
var _ dex.Step[Input] = CompleteProcess{}
var _ dex.RPC[dex.None, map[string]any] = BasicProcess.GetDexSummary
var _ dex.RPC[dex.None, map[string]any] = BasicProcess.GetDexDisplay
var _ dex.RPC[dex.None, Snapshot] = BasicProcess.DescribeProcess
var _ dex.RPC[dex.None, dex.None] = BasicProcess.ApproveProcess
