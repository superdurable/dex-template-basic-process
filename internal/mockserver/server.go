package mockserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/superdurable/dex-template-basic-process/internal/api/generated"
)

const operationDelay = 300 * time.Millisecond

type waitFunc func(context.Context, time.Duration) error

type Handler struct {
	store *Store
	wait  waitFunc
}

type ControlRequest struct {
	Action string `json:"action"`
	FlowID string `json:"flowId,omitempty"`
}

func New(store *Store) (http.Handler, error) {
	return newWithWait(store, waitForDuration)
}

func newWithWait(store *Store, wait waitFunc) (http.Handler, error) {
	if store == nil {
		return nil, errors.New("mock store is required")
	}
	handler := &Handler{store: store, wait: wait}
	apiServer, err := generated.NewServer(handler,
		generated.WithErrorHandler(writeGeneratedError),
		generated.WithNotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusNotFound, "not_found", "unknown API route")
		}),
		generated.WithMethodNotAllowed(func(w http.ResponseWriter, _ *http.Request, allowed string) {
			w.Header().Set("Allow", allowed)
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create mock OpenAPI server: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", apiServer)
	mux.HandleFunc("/__mock__/control", handler.handleControl)
	return mux, nil
}

func (handler *Handler) GetHealth(context.Context) (*generated.HealthResponse, error) {
	return &generated.HealthResponse{Status: generated.HealthResponseStatusOk}, nil
}

func (handler *Handler) CreateFlow(ctx context.Context, request *generated.CreateFlowRequest) (generated.CreateFlowRes, error) {
	if err := handler.wait(ctx, operationDelay); err != nil {
		return nil, err
	}
	view, err := handler.store.Create(request.Title)
	if errors.Is(err, ErrInjectedFailure) {
		response := generated.CreateFlowServiceUnavailable(errorResponse("mock_create_failed", "mock start failure"))
		return &response, nil
	}
	return &view, err
}

func (handler *Handler) GetFlow(_ context.Context, params generated.GetFlowParams) (generated.GetFlowRes, error) {
	view, err := handler.store.Get(params.FlowId)
	switch {
	case err == nil:
		return &view, nil
	case errors.Is(err, ErrUnknownFlow):
		response := generated.GetFlowNotFound(errorResponse("unknown_flow", "flow was not found"))
		return &response, nil
	case errors.Is(err, ErrInjectedFailure):
		response := generated.GetFlowServiceUnavailable(errorResponse("mock_get_failed", "mock refresh failure"))
		return &response, nil
	default:
		return nil, err
	}
}

func (handler *Handler) ApproveFlow(ctx context.Context, _ *generated.ApprovalRequest, params generated.ApproveFlowParams) (generated.ApproveFlowRes, error) {
	if err := handler.wait(ctx, operationDelay); err != nil {
		return nil, err
	}
	view, err := handler.store.Approve(params.FlowId)
	switch {
	case err == nil:
		return &view, nil
	case errors.Is(err, ErrUnknownFlow):
		response := generated.ApproveFlowNotFound(errorResponse("unknown_flow", "flow was not found"))
		return &response, nil
	case errors.Is(err, ErrDuplicateApproval):
		response := generated.ApproveFlowConflict(errorResponse("duplicate_approval", "approval is not pending"))
		return &response, nil
	case errors.Is(err, ErrInjectedFailure):
		response := generated.ApproveFlowServiceUnavailable(errorResponse("mock_approval_failed", "mock approval failure"))
		return &response, nil
	default:
		return nil, err
	}
}

func (handler *Handler) handleControl(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if request.Method == http.MethodGet {
		view, err := handler.store.Control(request.URL.Query().Get("flowId"))
		writeControlResponse(w, view, err)
		return
	}
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
		return
	}
	defer request.Body.Close()
	var control ControlRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&control); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_control", "mock control request is invalid")
		return
	}
	var (
		view ControlView
		err  error
	)
	switch control.Action {
	case "reset":
		view = handler.store.Reset()
	case "advance":
		view, err = handler.store.Advance(control.FlowID)
	case "emit_reminder":
		view, err = handler.store.EmitReminder(control.FlowID)
	case "fail_next_create":
		view, err = handler.store.InjectFailure(OperationCreate, control.FlowID)
	case "fail_next_get":
		view, err = handler.store.InjectFailure(OperationGet, control.FlowID)
	case "fail_next_approve":
		view, err = handler.store.InjectFailure(OperationApprove, control.FlowID)
	default:
		writeError(w, http.StatusBadRequest, "unknown_control", "mock control action is not supported")
		return
	}
	writeControlResponse(w, view, err)
}

func writeControlResponse(w http.ResponseWriter, view ControlView, err error) {
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, view)
	case errors.Is(err, ErrUnknownFlow):
		writeError(w, http.StatusNotFound, "unknown_flow", "flow was not found")
	default:
		writeError(w, http.StatusConflict, "invalid_control_state", err.Error())
	}
}

func waitForDuration(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func errorResponse(code, message string) generated.ErrorResponse {
	return generated.ErrorResponse{Error: code, Message: message}
}

func writeGeneratedError(_ context.Context, w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusBadRequest, "invalid_request", "request does not match the OpenAPI contract")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, generated.ErrorResponse{Error: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
