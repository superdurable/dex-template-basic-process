package mockserver

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/superdurable/dex-template-basic-process/internal/api/generated"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *fakeClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
}

func TestStoreAutomaticLifecycleAndReminder(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	store := NewStore(clock)
	created, err := store.Create("Review the launch checklist")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.State != generated.ProcessStateStarted {
		t.Fatalf("created state = %s", created.State)
	}
	clock.Advance(intakeValidatedAfter)
	validated, err := store.Get(created.FlowId)
	if err != nil || validated.State != generated.ProcessStateValidated {
		t.Fatalf("validated = %+v, %v", validated, err)
	}
	clock.Advance(intakeWaitingAfter - intakeValidatedAfter)
	waiting, err := store.Get(created.FlowId)
	if err != nil || waiting.State != generated.ProcessStateWaitingForApproval {
		t.Fatalf("waiting = %+v, %v", waiting, err)
	}
	reminded, err := store.EmitReminder(created.FlowId)
	if err != nil {
		t.Fatalf("emit reminder: %v", err)
	}
	if reminded.Flow.State != generated.ProcessStateReminderEmitted || reminded.Flow.ReminderCount != 1 {
		t.Fatalf("reminded = %+v", reminded.Flow)
	}
	approved, err := store.Approve(created.FlowId)
	if err != nil || approved.State != generated.ProcessStateApproved {
		t.Fatalf("approved = %+v, %v", approved, err)
	}
	clock.Advance(executionAfter)
	executing, err := store.Get(created.FlowId)
	if err != nil || executing.State != generated.ProcessStateExecuting {
		t.Fatalf("executing = %+v, %v", executing, err)
	}
	clock.Advance(completionAfter - executionAfter)
	completed, err := store.Get(created.FlowId)
	if err != nil || completed.State != generated.ProcessStateCompleted || completed.Result.Or("") != completionMessage {
		t.Fatalf("completed = %+v, %v", completed, err)
	}
	if _, err := store.Approve(created.FlowId); !errors.Is(err, ErrDuplicateApproval) {
		t.Fatalf("duplicate approval error = %v", err)
	}
}

func TestStoreManualControlAndReset(t *testing.T) {
	store := NewStore(&fakeClock{now: time.Now()})
	created, err := store.Create("Manual process")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	want := []generated.ProcessState{
		generated.ProcessStateValidated,
		generated.ProcessStateWaitingForApproval,
		generated.ProcessStateApproved,
		generated.ProcessStateExecuting,
		generated.ProcessStateCompleted,
	}
	for _, state := range want {
		view, advanceErr := store.Advance(created.FlowId)
		if advanceErr != nil {
			t.Fatalf("advance to %s: %v", state, advanceErr)
		}
		if view.Flow.State != state {
			t.Fatalf("advanced state = %s, want %s", view.Flow.State, state)
		}
	}
	store.Reset()
	if _, err := store.Get(created.FlowId); !errors.Is(err, ErrUnknownFlow) {
		t.Fatalf("get after reset = %v", err)
	}
}

func TestStoreFailuresAreConsumedOnce(t *testing.T) {
	store := NewStore(&fakeClock{now: time.Now()})
	if _, err := store.InjectFailure(OperationCreate, ""); err != nil {
		t.Fatalf("inject create: %v", err)
	}
	if _, err := store.Create("Fails once"); !errors.Is(err, ErrInjectedFailure) {
		t.Fatalf("first create error = %v", err)
	}
	created, err := store.Create("Succeeds next")
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	for _, operation := range []Operation{OperationGet, OperationApprove} {
		if _, err := store.InjectFailure(operation, created.FlowId); err != nil {
			t.Fatalf("inject %s: %v", operation, err)
		}
	}
	if _, err := store.Get(created.FlowId); !errors.Is(err, ErrInjectedFailure) {
		t.Fatalf("get error = %v", err)
	}
	if _, err := store.Get(created.FlowId); err != nil {
		t.Fatalf("second get: %v", err)
	}
	if _, err := store.Approve(created.FlowId); !errors.Is(err, ErrInjectedFailure) {
		t.Fatalf("approve error = %v", err)
	}
}

func TestStoreAllowsConcurrentReads(t *testing.T) {
	store := NewStore(nil)
	created, err := store.Create("Concurrent reads")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, getErr := store.Get(created.FlowId); getErr != nil {
				t.Errorf("get: %v", getErr)
			}
		}()
	}
	wait.Wait()
}
