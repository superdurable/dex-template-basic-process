package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/superdurable/dex-template-basic-process/internal/api/generated"
	"github.com/superdurable/dex-template-basic-process/internal/process"
)

type Handler struct{ processes *process.Service }

func NewHandler(processes *process.Service) (*generated.Server, error) {
	handler := &Handler{processes: processes}
	return generated.NewServer(handler,
		generated.WithErrorHandler(writeGeneratedError),
		generated.WithNotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusNotFound, "not_found", "unknown API route")
		}),
		generated.WithMethodNotAllowed(func(w http.ResponseWriter, _ *http.Request, allowed string) {
			w.Header().Set("Allow", allowed)
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
		}),
	)
}

func (handler *Handler) GetHealth(context.Context) (*generated.HealthResponse, error) {
	return &generated.HealthResponse{Status: generated.HealthResponseStatusOk}, nil
}

func (handler *Handler) CreateFlow(ctx context.Context, request *generated.CreateFlowRequest) (generated.CreateFlowRes, error) {
	view, err := handler.processes.Start(ctx, request.Title)
	if err != nil {
		response := generated.CreateFlowServiceUnavailable(errorResponse("flow_start_failed", "unable to start flow"))
		return &response, nil
	}
	return generatedView(view), nil
}

func (handler *Handler) GetFlow(ctx context.Context, params generated.GetFlowParams) (generated.GetFlowRes, error) {
	view, err := handler.processes.Get(ctx, params.FlowId)
	switch {
	case err == nil:
		return generatedView(view), nil
	case errors.Is(err, process.ErrUnknownFlow):
		response := generated.GetFlowNotFound(errorResponse("unknown_flow", "flow was not found"))
		return &response, nil
	default:
		response := generated.GetFlowServiceUnavailable(errorResponse("flow_operation_failed", "flow operation failed"))
		return &response, nil
	}
}

func (handler *Handler) ApproveFlow(ctx context.Context, _ *generated.ApprovalRequest, params generated.ApproveFlowParams) (generated.ApproveFlowRes, error) {
	view, err := handler.processes.Approve(ctx, params.FlowId)
	switch {
	case err == nil:
		return generatedView(view), nil
	case errors.Is(err, process.ErrUnknownFlow):
		response := generated.ApproveFlowNotFound(errorResponse("unknown_flow", "flow was not found"))
		return &response, nil
	case errors.Is(err, process.ErrDuplicateApproval):
		response := generated.ApproveFlowConflict(errorResponse("duplicate_approval", "approval is not pending"))
		return &response, nil
	default:
		response := generated.ApproveFlowServiceUnavailable(errorResponse("flow_operation_failed", "flow operation failed"))
		return &response, nil
	}
}

func generatedView(view process.View) *generated.FlowView {
	response := &generated.FlowView{FlowId: view.FlowID, Title: view.Title, State: generated.ProcessState(view.State), ReminderCount: view.ReminderCount}
	if view.Result != "" {
		response.Result = generated.NewOptString(view.Result)
	}
	return response
}

func errorResponse(code, message string) generated.ErrorResponse {
	return generated.ErrorResponse{Error: code, Message: message}
}

func writeGeneratedError(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	_ = err
	writeError(w, http.StatusBadRequest, "invalid_request", "request does not match the OpenAPI contract")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(generated.ErrorResponse{Error: code, Message: message})
}
