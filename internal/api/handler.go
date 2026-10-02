package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/superdurable/dex-template-basic-process/internal/api/generated"
)

// Handler owns the non-business application shell. Flow management is provided
// by the host's authenticated Studio or standalone Dex management UI.
type Handler struct{}

func NewHandler() (*generated.Server, error) {
	return generated.NewServer(&Handler{},
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

func (*Handler) NewError(context.Context, error) *generated.ErrorStatusCode {
	return &generated.ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response:   generated.ErrorResponse{Error: "application_unavailable", Message: "application is unavailable"},
	}
}

func (*Handler) GetHealth(context.Context) (*generated.HealthResponse, error) {
	return &generated.HealthResponse{Status: generated.HealthResponseStatusOk}, nil
}

func (*Handler) GetApplicationInfo(context.Context) (*generated.ApplicationInfo, error) {
	return &generated.ApplicationInfo{Name: "Dex Application"}, nil
}

func writeGeneratedError(_ context.Context, w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusBadRequest, "invalid_request", "request does not match the OpenAPI contract")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(generated.ErrorResponse{Error: code, Message: message})
}
