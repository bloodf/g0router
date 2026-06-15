package openai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestBatchCreateSendsJSONAndReturnsBatch verifies BatchCreate POSTs the JSON
// body (input_file_id/endpoint/completion_window round-trip) and decodes Batch.
func TestBatchCreateSendsJSONAndReturnsBatch(t *testing.T) {
	var got schemas.BatchCreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/batches" {
			t.Errorf("path = %q, want /v1/batches", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth = %q, want Bearer test-key", auth)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"batch_1","object":"batch","endpoint":"/v1/chat/completions","input_file_id":"file-in","completion_window":"24h","status":"validating"}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.BatchCreate(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.BatchCreateRequest{
		InputFileID: "file-in", Endpoint: "/v1/chat/completions", CompletionWindow: "24h",
	})
	if perr != nil {
		t.Fatalf("BatchCreate error: %v", perr.Message)
	}
	if got.InputFileID != "file-in" || got.Endpoint != "/v1/chat/completions" || got.CompletionWindow != "24h" {
		t.Errorf("body = %+v, want round-trip", got)
	}
	if resp.ID != "batch_1" {
		t.Errorf("ID = %q, want batch_1", resp.ID)
	}
}

// TestBatchCreateUpstreamError verifies non-200 surfaces a *ProviderError.
func TestBatchCreateUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.BatchCreate(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.BatchCreateRequest{InputFileID: "f", Endpoint: "/v1/chat/completions", CompletionWindow: "24h"})
	if perr == nil {
		t.Fatal("expected *ProviderError, got nil")
	}
	if perr.StatusCode != 400 {
		t.Errorf("status = %d, want 400", perr.StatusCode)
	}
}

// TestBatchListReturnsResponse verifies BatchList issues a GET and decodes.
func TestBatchListReturnsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/batches" {
			t.Errorf("path = %q, want /v1/batches", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[{"id":"batch_1","object":"batch"},{"id":"batch_2","object":"batch"}]}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.BatchList(&schemas.GatewayContext{}, schemas.Key{Value: "k"})
	if perr != nil {
		t.Fatalf("BatchList error: %v", perr.Message)
	}
	if len(resp.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(resp.Data))
	}
}

// TestBatchRetrieveBuildsPathAndReturnsBatch verifies the upstream URI carries
// the batch id and the JSON Batch is decoded.
func TestBatchRetrieveBuildsPathAndReturnsBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/batches/batch_xyz" {
			t.Errorf("path = %q, want /v1/batches/batch_xyz", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"batch_xyz","object":"batch","status":"completed"}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.BatchRetrieve(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, "batch_xyz")
	if perr != nil {
		t.Fatalf("BatchRetrieve error: %v", perr.Message)
	}
	if resp.ID != "batch_xyz" {
		t.Errorf("ID = %q, want batch_xyz", resp.ID)
	}
}

// TestBatchCancelBuildsPathAndReturnsBatch verifies POST on the cancel path.
func TestBatchCancelBuildsPathAndReturnsBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/batches/batch_c/cancel" {
			t.Errorf("path = %q, want /v1/batches/batch_c/cancel", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"batch_c","object":"batch","status":"cancelling"}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.BatchCancel(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, "batch_c")
	if perr != nil {
		t.Fatalf("BatchCancel error: %v", perr.Message)
	}
	if resp.Status != "cancelling" {
		t.Errorf("Status = %q, want cancelling", resp.Status)
	}
}
