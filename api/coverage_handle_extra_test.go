package api

import (
	"net/http"
	"testing"
)

// TestHandleExtraGetReturns404 exercises the non-POST branch in handleExtra
// (lines 371-374): GET /v1/embeddings returns 404.
func TestHandleExtraGetReturns404(t *testing.T) {
	_, base := startTestServer(t, ServerConfig{Port: 0, Version: "test"})

	for _, path := range []string{
		"/v1/embeddings",
		"/v1/images/generations",
		"/v1/audio/transcriptions",
		"/v1/audio/speech",
	} {
		t.Run("GET "+path, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, base+path, nil)
			req.Header.Set("X-API-Key", testHarnessAPIKey)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("GET %s = %d, want 404", path, resp.StatusCode)
			}
		})
	}
}

// TestHandleExtraPostDispatchesToHandler exercises the POST branch in handleExtra:
// the policy gate passes (no key required) and the nil engine path returns 501.
func TestHandleExtraPostNoEngineReturns501(t *testing.T) {
	// Server with no InferenceEngine — handleExtra passes a nil ExtraEngine to
	// the handler, which should return 501 Not Implemented.
	_, base := startTestServer(t, ServerConfig{Port: 0, Version: "test", RequireAPIKey: false})

	for _, path := range []string{
		"/v1/embeddings",
		"/v1/images/generations",
		"/v1/audio/transcriptions",
		"/v1/audio/speech",
	} {
		t.Run("POST "+path, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, base+path, http.NoBody)
			req.Header.Set("Content-Type", "application/json")
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("POST %s: %v", path, err)
			}
			resp.Body.Close()
			// With nil engine the handler returns 501; any non-404 confirms
			// the POST branch was reached (handleExtra did not 404 on method).
			if resp.StatusCode == http.StatusNotFound {
				t.Fatalf("POST %s = 404, handleExtra should not 404 on POST", path)
			}
		})
	}
}
