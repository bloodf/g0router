package openai

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestFileUploadSendsMultipartAndReturnsObject verifies FileUpload builds a
// multipart/form-data outbound body whose file part round-trips and whose
// purpose value is present, then parses the JSON FileObject (ESC-MULTIPART-UPLOAD).
func TestFileUploadSendsMultipartAndReturnsObject(t *testing.T) {
	fileBytes := []byte("line1\nline2\n")
	var gotFile []byte
	var gotPurpose string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/files" {
			t.Errorf("path = %q, want /v1/files", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth = %q, want Bearer test-key", got)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("inbound Content-Type = %q, want multipart/form-data", ct)
		}
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			t.Fatalf("parse media type: %v", err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		form, err := mr.ReadForm(1 << 20)
		if err != nil {
			t.Fatalf("read form: %v", err)
		}
		if fhs := form.File["file"]; len(fhs) == 1 {
			f, _ := fhs[0].Open()
			gotFile, _ = io.ReadAll(f)
		}
		if pv := form.Value["purpose"]; len(pv) == 1 {
			gotPurpose = pv[0]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"file-abc","object":"file","bytes":12,"purpose":"batch","filename":"data.jsonl"}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.FileUpload(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.FileUploadRequest{
		File: fileBytes, Filename: "data.jsonl", Purpose: "batch",
	})
	if perr != nil {
		t.Fatalf("FileUpload error: %v", perr.Message)
	}
	if string(gotFile) != string(fileBytes) {
		t.Errorf("file part = %q, want round-trip of input", gotFile)
	}
	if gotPurpose != "batch" {
		t.Errorf("purpose = %q, want batch", gotPurpose)
	}
	if resp.ID != "file-abc" {
		t.Errorf("ID = %q, want file-abc", resp.ID)
	}
}

// TestFileUploadUpstreamError verifies an upstream non-200 becomes a *ProviderError.
func TestFileUploadUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.FileUpload(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.FileUploadRequest{File: []byte("x"), Filename: "x", Purpose: "batch"})
	if perr == nil {
		t.Fatal("expected *ProviderError, got nil")
	}
	if perr.StatusCode != 400 {
		t.Errorf("status = %d, want 400", perr.StatusCode)
	}
}

// TestFileListReturnsResponse verifies FileList issues a GET and decodes the list.
func TestFileListReturnsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/files" {
			t.Errorf("path = %q, want /v1/files", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[{"id":"file-1","object":"file"},{"id":"file-2","object":"file"}]}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.FileList(&schemas.GatewayContext{}, schemas.Key{Value: "k"})
	if perr != nil {
		t.Fatalf("FileList error: %v", perr.Message)
	}
	if len(resp.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(resp.Data))
	}
}

// TestFileRetrieveBuildsPathAndReturnsObject verifies the upstream URI carries
// the file id and the JSON FileObject is decoded.
func TestFileRetrieveBuildsPathAndReturnsObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/files/file-xyz" {
			t.Errorf("path = %q, want /v1/files/file-xyz", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"file-xyz","object":"file"}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.FileRetrieve(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, "file-xyz")
	if perr != nil {
		t.Fatalf("FileRetrieve error: %v", perr.Message)
	}
	if resp.ID != "file-xyz" {
		t.Errorf("ID = %q, want file-xyz", resp.ID)
	}
}

// TestFileDeleteBuildsPathAndReturnsResponse verifies DELETE on the id path.
func TestFileDeleteBuildsPathAndReturnsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/files/file-del" {
			t.Errorf("path = %q, want /v1/files/file-del", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"file-del","object":"file","deleted":true}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.FileDelete(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, "file-del")
	if perr != nil {
		t.Fatalf("FileDelete error: %v", perr.Message)
	}
	if !resp.Deleted {
		t.Errorf("Deleted = false, want true")
	}
}

// TestFileContentReturnsRawBytes verifies FileContent copies the upstream body
// verbatim and does NOT decode it as JSON (ESC-FILE-CONTENT-BYTES).
func TestFileContentReturnsRawBytes(t *testing.T) {
	content := []byte("{\"id\":\"req-1\"}\n{\"id\":\"req-2\"}\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/files/file-c/content" {
			t.Errorf("path = %q, want /v1/files/file-c/content", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	got, perr := p.FileContent(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, "file-c")
	if perr != nil {
		t.Fatalf("FileContent error: %v", perr.Message)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

// TestFileContentUpstreamError verifies non-200 surfaces a *ProviderError.
func TestFileContentUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"no file","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.FileContent(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, "missing")
	if perr == nil {
		t.Fatal("expected *ProviderError, got nil")
	}
	if perr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", perr.StatusCode)
	}
}
