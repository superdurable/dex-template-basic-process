//go:build integration

package process_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/superdurable/dex-template-basic-process/internal/process"
	appRuntime "github.com/superdurable/dex-template-basic-process/internal/runtime"
)

func TestReminderWorkerRestartApprovalAndTerminalRejection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	runtime := startRuntime(t, logger)
	t.Log("first Worker started")
	view, err := runtime.Processes.Start(ctx, "Review the launch checklist")
	if err != nil {
		t.Fatalf("start Flow: %v", err)
	}
	t.Logf("Flow started: %s", view.FlowID)
	waitFor(t, ctx, func(attemptCtx context.Context) bool {
		current, getErr := runtime.Processes.Get(attemptCtx, view.FlowID)
		return getErr == nil && current.State == process.StateWaitingForApproval
	})
	t.Log("Flow is waiting for approval")

	skipReminderTimer(t, ctx, view.FlowID)
	t.Log("reminder Timer skipped")
	waitFor(t, ctx, func(attemptCtx context.Context) bool {
		current, getErr := runtime.Processes.Get(attemptCtx, view.FlowID)
		return getErr == nil && current.ReminderCount == 1 && current.State == process.StateReminderEmitted
	})
	t.Log("reminder observed")

	if err := runtime.Close(); err != nil {
		t.Fatalf("stop first Worker: %v", err)
	}
	t.Log("first Worker stopped")
	runtime = startRuntime(t, logger)
	t.Log("replacement Worker started")
	waitFor(t, ctx, func(attemptCtx context.Context) bool {
		current, getErr := runtime.Processes.Get(attemptCtx, view.FlowID)
		return getErr == nil && current.State == process.StateReminderEmitted
	})
	approved, err := runtime.Processes.Approve(ctx, view.FlowID)
	if err != nil {
		t.Fatalf("approve after Worker restart: %v", err)
	}
	if approved.ReminderCount != 1 {
		t.Fatalf("reminder count = %d, want 1", approved.ReminderCount)
	}
	t.Log("approval accepted")
	lastObservation := ""
	waitFor(t, ctx, func(attemptCtx context.Context) bool {
		current, getErr := runtime.Processes.Get(attemptCtx, view.FlowID)
		observation := fmt.Sprintf("state=%s result=%q error=%v", current.State, current.Result, getErr)
		if observation != lastObservation {
			t.Log(observation)
			lastObservation = observation
		}
		return getErr == nil && current.State == process.StateCompleted && current.Result != ""
	})
	t.Log("completion observed")
	if _, err := runtime.Processes.Approve(ctx, view.FlowID); err != process.ErrDuplicateApproval {
		t.Fatalf("terminal approval error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("stop replacement Worker: %v", err)
	}
}

func TestDirectApprovalCompletesWithoutReminder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	runtime := startRuntime(t, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	defer func() {
		if err := runtime.Close(); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	}()
	view, err := runtime.Processes.Start(ctx, "Approve without a reminder")
	if err != nil {
		t.Fatalf("start Flow: %v", err)
	}
	waitFor(t, ctx, func(attemptCtx context.Context) bool {
		current, getErr := runtime.Processes.Get(attemptCtx, view.FlowID)
		return getErr == nil && current.State == process.StateWaitingForApproval
	})
	if _, err := runtime.Processes.Approve(ctx, view.FlowID); err != nil {
		t.Fatalf("approve Flow: %v", err)
	}
	waitFor(t, ctx, func(attemptCtx context.Context) bool {
		current, getErr := runtime.Processes.Get(attemptCtx, view.FlowID)
		return getErr == nil && current.State == process.StateCompleted && current.ReminderCount == 0
	})
}

func startRuntime(t *testing.T, logger *slog.Logger) *appRuntime.Runtime {
	t.Helper()
	runtime, err := appRuntime.New(logger)
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	workerResult := runtime.StartWorker()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		connection, dialErr := net.DialTimeout("tcp", os.Getenv("DEX_WORKER_TARGET"), 100*time.Millisecond)
		if dialErr == nil {
			_ = connection.Close()
			return runtime
		}
		select {
		case err := <-workerResult:
			t.Fatalf("start Worker: %v", err)
		case <-deadline.C:
			t.Fatalf("wait for Worker listener: %v", dialErr)
		case <-ticker.C:
		}
	}
}

func skipReminderTimer(t *testing.T, ctx context.Context, flowID string) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastError error
	var lastOutput []byte
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		command := exec.CommandContext(attemptCtx, "dexcli", "flow", "skip-timer", flowID,
			"-server", os.Getenv("DEX_FLOW_SERVICE_ADDRESS"), "-step-type", "process.WaitForApproval",
			"-condition-id", process.ReminderTimerConditionID, "-timeout", "1s", "-yes")
		output, skipErr := command.CombinedOutput()
		cancel()
		if skipErr == nil {
			return
		}
		lastError = skipErr
		lastOutput = output
		select {
		case <-ctx.Done():
			t.Fatalf("skip reminder Timer before deadline: %v\n%s", lastError, lastOutput)
		case <-ticker.C:
		}
	}
}

func waitFor(t *testing.T, ctx context.Context, condition func(context.Context) bool) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, time.Second)
		matched := condition(attemptCtx)
		cancel()
		if matched {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(fmt.Errorf("deadline waiting for Flow convergence: %w", ctx.Err()))
		case <-ticker.C:
		}
	}
}
