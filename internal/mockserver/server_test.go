package mockserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/superdurable/dex-template-basic-process/internal/api/generated"
)

func TestServerCoversApplicationAndControlRoutes(t *testing.T) {
	handler, err := newWithWait(NewStore(nil), func(context.Context, time.Duration) error { return nil })
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	health := request(t, handler, http.MethodGet, "/api/health", nil)
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", health.StatusCode)
	}
	health.Body.Close()

	createdResponse := request(t, handler, http.MethodPost, "/api/flows", map[string]any{"title": "Mock review"})
	if createdResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createdResponse.StatusCode, readBody(t, createdResponse))
	}
	var created generated.FlowView
	decode(t, createdResponse, &created)

	advance := request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "advance", FlowID: created.FlowId})
	if advance.StatusCode != http.StatusOK {
		t.Fatalf("advance status = %d", advance.StatusCode)
	}
	advance.Body.Close()
	advance = request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "advance", FlowID: created.FlowId})
	advance.Body.Close()

	approve := request(t, handler, http.MethodPost, "/api/flows/"+created.FlowId+"/approvals", map[string]any{"approved": true})
	if approve.StatusCode != http.StatusOK {
		t.Fatalf("approve status = %d, body = %s", approve.StatusCode, readBody(t, approve))
	}
	approve.Body.Close()

	reset := request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "reset"})
	if reset.StatusCode != http.StatusOK {
		t.Fatalf("reset status = %d", reset.StatusCode)
	}
	reset.Body.Close()

	missing := request(t, handler, http.MethodGet, "/api/flows/"+created.FlowId, nil)
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d", missing.StatusCode)
	}
	missing.Body.Close()
}

func TestServerMatchesEveryApplicationResponseStatus(t *testing.T) {
	handler, err := newWithWait(NewStore(nil), func(context.Context, time.Duration) error { return nil })
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	assertStatus(t, request(t, handler, http.MethodPost, "/api/flows", map[string]any{"title": ""}), http.StatusBadRequest)
	createdResponse := request(t, handler, http.MethodPost, "/api/flows", map[string]any{"title": "Contract review"})
	if createdResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createdResponse.StatusCode, readBody(t, createdResponse))
	}
	var created generated.FlowView
	decode(t, createdResponse, &created)

	assertStatus(t, request(t, handler, http.MethodGet, "/api/flows/invalid", nil), http.StatusBadRequest)
	assertStatus(t, request(t, handler, http.MethodGet, "/api/flows/"+created.FlowId, nil), http.StatusOK)
	assertStatus(t, request(t, handler, http.MethodGet, "/api/flows/process-00000000-0000-4000-8000-000000000000", nil), http.StatusNotFound)
	assertStatus(t, request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "fail_next_get", FlowID: created.FlowId}), http.StatusOK)
	assertStatus(t, request(t, handler, http.MethodGet, "/api/flows/"+created.FlowId, nil), http.StatusServiceUnavailable)

	approvalURL := "/api/flows/" + created.FlowId + "/approvals"
	assertStatus(t, request(t, handler, http.MethodPost, approvalURL, map[string]any{"approved": false}), http.StatusBadRequest)
	assertStatus(t, request(t, handler, http.MethodPost, "/api/flows/process-00000000-0000-4000-8000-000000000000/approvals", map[string]any{"approved": true}), http.StatusNotFound)
	assertStatus(t, request(t, handler, http.MethodPost, approvalURL, map[string]any{"approved": true}), http.StatusConflict)
	assertStatus(t, request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "advance", FlowID: created.FlowId}), http.StatusOK)
	assertStatus(t, request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "advance", FlowID: created.FlowId}), http.StatusOK)
	assertStatus(t, request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "fail_next_approve", FlowID: created.FlowId}), http.StatusOK)
	assertStatus(t, request(t, handler, http.MethodPost, approvalURL, map[string]any{"approved": true}), http.StatusServiceUnavailable)
	assertStatus(t, request(t, handler, http.MethodPost, approvalURL, map[string]any{"approved": true}), http.StatusOK)
	assertStatus(t, request(t, handler, http.MethodPost, approvalURL, map[string]any{"approved": true}), http.StatusConflict)
}

func TestServerReturnsInjectedFailuresAndRejectsUnknownControls(t *testing.T) {
	handler, err := newWithWait(NewStore(nil), func(context.Context, time.Duration) error { return nil })
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	inject := request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "fail_next_create"})
	inject.Body.Close()
	failed := request(t, handler, http.MethodPost, "/api/flows", map[string]any{"title": "Fails"})
	if failed.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("failed create status = %d", failed.StatusCode)
	}
	failed.Body.Close()
	unknown := request(t, handler, http.MethodPost, "/__mock__/control", ControlRequest{Action: "unsupported"})
	if unknown.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown control status = %d", unknown.StatusCode)
	}
	unknown.Body.Close()
}

func request(t *testing.T, handler http.Handler, method, url string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, url, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Result()
}

func decode(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(contents)
}

func assertStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode != want {
		contents, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d, body = %s", response.StatusCode, want, contents)
	}
}
