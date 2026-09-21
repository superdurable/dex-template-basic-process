package mockserver

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/superdurable/dex-template-basic-process/internal/api/generated"
)

const completionMessage = "approved automation completed"

const (
	intakeValidatedAfter = 300 * time.Millisecond
	intakeWaitingAfter   = 700 * time.Millisecond
	executionAfter       = 300 * time.Millisecond
	completionAfter      = 900 * time.Millisecond
)

type Operation string

const (
	OperationCreate  Operation = "create"
	OperationGet     Operation = "get"
	OperationApprove Operation = "approve"
)

var (
	ErrUnknownFlow       = errors.New("unknown flow")
	ErrDuplicateApproval = errors.New("approval is not pending")
	ErrInjectedFailure   = errors.New("mock operation failure")
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type flowRecord struct {
	view          generated.FlowView
	phaseStarted  time.Time
	phase         string
	manualControl bool
}

type Store struct {
	mu              sync.Mutex
	clock           Clock
	flows           map[string]*flowRecord
	pendingFailures map[Operation]bool
}

type ControlView struct {
	Mode            string              `json:"mode"`
	Flow            *generated.FlowView `json:"flow,omitempty"`
	PendingFailures []Operation         `json:"pendingFailures"`
}

func NewStore(clock Clock) *Store {
	if clock == nil {
		clock = systemClock{}
	}
	return &Store{
		clock:           clock,
		flows:           make(map[string]*flowRecord),
		pendingFailures: make(map[Operation]bool),
	}
}

func (store *Store) Create(title string) (generated.FlowView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.consumeFailure(OperationCreate) {
		return generated.FlowView{}, ErrInjectedFailure
	}
	flowID := "process-" + uuid.NewString()
	record := &flowRecord{
		view: generated.FlowView{
			FlowId: flowID, Title: title,
			State: generated.ProcessStateStarted,
		},
		phaseStarted: store.clock.Now(),
		phase:        "intake",
	}
	store.flows[flowID] = record
	return cloneView(record.view), nil
}

func (store *Store) Get(flowID string) (generated.FlowView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.consumeFailure(OperationGet) {
		return generated.FlowView{}, ErrInjectedFailure
	}
	record, ok := store.flows[flowID]
	if !ok {
		return generated.FlowView{}, ErrUnknownFlow
	}
	store.materialize(record)
	return cloneView(record.view), nil
}

func (store *Store) Approve(flowID string) (generated.FlowView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.consumeFailure(OperationApprove) {
		return generated.FlowView{}, ErrInjectedFailure
	}
	record, ok := store.flows[flowID]
	if !ok {
		return generated.FlowView{}, ErrUnknownFlow
	}
	store.materialize(record)
	if record.view.State != generated.ProcessStateWaitingForApproval && record.view.State != generated.ProcessStateReminderEmitted {
		return generated.FlowView{}, ErrDuplicateApproval
	}
	record.view.State = generated.ProcessStateApproved
	record.phaseStarted = store.clock.Now()
	record.phase = "execution"
	record.manualControl = false
	return cloneView(record.view), nil
}

func (store *Store) Reset() ControlView {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.flows = make(map[string]*flowRecord)
	store.pendingFailures = make(map[Operation]bool)
	return store.controlView(nil)
}

func (store *Store) InjectFailure(operation Operation, flowID string) (ControlView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !slices.Contains([]Operation{OperationCreate, OperationGet, OperationApprove}, operation) {
		return ControlView{}, errors.New("unsupported failure operation")
	}
	store.pendingFailures[operation] = true
	record := store.lookupAndMaterialize(flowID)
	if record == nil {
		return store.controlView(nil), nil
	}
	return store.controlView(&record.view), nil
}

func (store *Store) Advance(flowID string) (ControlView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, ok := store.flows[flowID]
	if !ok {
		return ControlView{}, ErrUnknownFlow
	}
	store.materialize(record)
	next := map[generated.ProcessState]generated.ProcessState{
		generated.ProcessStateStarted:            generated.ProcessStateValidated,
		generated.ProcessStateValidated:          generated.ProcessStateWaitingForApproval,
		generated.ProcessStateWaitingForApproval: generated.ProcessStateApproved,
		generated.ProcessStateReminderEmitted:    generated.ProcessStateApproved,
		generated.ProcessStateApproved:           generated.ProcessStateExecuting,
		generated.ProcessStateExecuting:          generated.ProcessStateCompleted,
	}
	if state, ok := next[record.view.State]; ok {
		record.view.State = state
		if state == generated.ProcessStateCompleted {
			record.view.Result = generated.NewOptString(completionMessage)
		}
	}
	record.phase = ""
	record.manualControl = true
	return store.controlView(&record.view), nil
}

func (store *Store) EmitReminder(flowID string) (ControlView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record, ok := store.flows[flowID]
	if !ok {
		return ControlView{}, ErrUnknownFlow
	}
	store.materialize(record)
	if record.view.State != generated.ProcessStateWaitingForApproval && record.view.State != generated.ProcessStateReminderEmitted {
		return ControlView{}, errors.New("flow is not waiting for approval")
	}
	record.view.ReminderCount++
	record.view.State = generated.ProcessStateReminderEmitted
	record.phase = ""
	record.manualControl = true
	return store.controlView(&record.view), nil
}

func (store *Store) Control(flowID string) (ControlView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if flowID == "" {
		return store.controlView(nil), nil
	}
	record := store.lookupAndMaterialize(flowID)
	if record == nil {
		return ControlView{}, ErrUnknownFlow
	}
	return store.controlView(&record.view), nil
}

func (store *Store) consumeFailure(operation Operation) bool {
	if !store.pendingFailures[operation] {
		return false
	}
	delete(store.pendingFailures, operation)
	return true
}

func (store *Store) lookupAndMaterialize(flowID string) *flowRecord {
	if flowID == "" {
		return nil
	}
	record := store.flows[flowID]
	if record != nil {
		store.materialize(record)
	}
	return record
}

func (store *Store) materialize(record *flowRecord) {
	if record.manualControl {
		return
	}
	elapsed := store.clock.Now().Sub(record.phaseStarted)
	switch record.phase {
	case "intake":
		switch {
		case elapsed >= intakeWaitingAfter:
			record.view.State = generated.ProcessStateWaitingForApproval
			record.phase = ""
		case elapsed >= intakeValidatedAfter:
			record.view.State = generated.ProcessStateValidated
		}
	case "execution":
		switch {
		case elapsed >= completionAfter:
			record.view.State = generated.ProcessStateCompleted
			record.view.Result = generated.NewOptString(completionMessage)
			record.phase = ""
		case elapsed >= executionAfter:
			record.view.State = generated.ProcessStateExecuting
		}
	}
}

func (store *Store) controlView(flow *generated.FlowView) ControlView {
	pending := make([]Operation, 0, len(store.pendingFailures))
	for _, operation := range []Operation{OperationCreate, OperationGet, OperationApprove} {
		if store.pendingFailures[operation] {
			pending = append(pending, operation)
		}
	}
	view := ControlView{Mode: "mock", PendingFailures: pending}
	if flow != nil {
		cloned := cloneView(*flow)
		view.Flow = &cloned
	}
	return view
}

func cloneView(view generated.FlowView) generated.FlowView { return view }
