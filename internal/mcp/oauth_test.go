package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOAuthEngineCompletesCallbackForMatchingInstance(t *testing.T) {
	store := newFakeOAuthStore()
	engine := NewOAuthEngine(store, OAuthHTTPClient(nil))
	flow := OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}
	if err := store.CreateFlow(flow); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	account, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if account.InstanceID != "inst-1" || account.AccessToken == "" {
		t.Fatalf("account = %+v, want token for inst-1", account)
	}
	if _, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1"); err == nil {
		t.Fatal("state should be single-use")
	}
}

func TestOAuthEngineRejectsMismatchedInstanceState(t *testing.T) {
	store := newFakeOAuthStore()
	engine := NewOAuthEngine(store, OAuthHTTPClient(nil))
	if err := store.CreateFlow(OAuthFlow{InstanceID: "inst-1", State: "state-1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	_, err := engine.CompleteCallback(context.Background(), "inst-2", "https://callback.example?code=ok&state=state-1")
	if err == nil {
		t.Fatal("mismatched instance should fail")
	}
}

func TestOAuthEngineAddsBearerAndProtocolHeaders(t *testing.T) {
	var gotAuth string
	var gotProtocol string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotProtocol = r.Header.Get("MCP-Protocol-Version")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewOAuthEngine(newFakeOAuthStore(), server.Client())
	err := engine.AuthorizeRequest(&http.Request{Header: http.Header{}}, OAuthAccount{
		InstanceID:  "inst-1",
		AccessToken: "token-1",
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("AuthorizeRequest: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	if err := engine.AuthorizeRequest(req, OAuthAccount{AccessToken: "token-1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("AuthorizeRequest req: %v", err)
	}
	if _, err := server.Client().Do(req); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotAuth != "Bearer token-1" {
		t.Fatalf("Authorization = %q, want bearer", gotAuth)
	}
	if gotProtocol == "" {
		t.Fatal("MCP protocol header is empty")
	}
}

func TestOAuthEngineRequiresReauthForExpiredOrWrongResource(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), OAuthHTTPClient(nil))
	req, _ := http.NewRequest(http.MethodGet, "https://mcp.example", nil)

	err := engine.AuthorizeRequest(req, OAuthAccount{AccessToken: "token", ExpiresAt: time.Now().Add(-time.Minute)})
	if err != ErrReauthRequired {
		t.Fatalf("expired err = %v, want ErrReauthRequired", err)
	}
	err = engine.AuthorizeRequest(req, OAuthAccount{AccessToken: "token", ResourceURI: "https://other.example", ExpiresAt: time.Now().Add(time.Hour)})
	if err != ErrReauthRequired {
		t.Fatalf("wrong resource err = %v, want ErrReauthRequired", err)
	}
}

func TestStdioCredentialsReturnRedactedEnv(t *testing.T) {
	env := StdioCredentialEnv(OAuthAccount{AccessToken: "token", RefreshToken: "refresh"})

	if env.Actual["MCP_ACCESS_TOKEN"] != "token" {
		t.Fatalf("actual token = %q, want token", env.Actual["MCP_ACCESS_TOKEN"])
	}
	if env.Redacted["MCP_ACCESS_TOKEN"] != RedactedValue {
		t.Fatalf("redacted token = %q, want redacted", env.Redacted["MCP_ACCESS_TOKEN"])
	}
	if env.Redacted["MCP_REFRESH_TOKEN"] != RedactedValue {
		t.Fatalf("redacted refresh = %q, want redacted", env.Redacted["MCP_REFRESH_TOKEN"])
	}
}

type fakeOAuthStore struct {
	flows    map[string]OAuthFlow
	accounts []OAuthAccount
}

func newFakeOAuthStore() *fakeOAuthStore {
	return &fakeOAuthStore{flows: make(map[string]OAuthFlow)}
}

func (s *fakeOAuthStore) CreateFlow(flow OAuthFlow) error {
	s.flows[flow.InstanceID+"|"+flow.State] = flow
	return nil
}

func (s *fakeOAuthStore) ConsumeFlow(instanceID, state string) (OAuthFlow, error) {
	key := instanceID + "|" + state
	flow, ok := s.flows[key]
	if !ok {
		return OAuthFlow{}, ErrOAuthFlowNotFound
	}
	delete(s.flows, key)
	return flow, nil
}

func (s *fakeOAuthStore) SaveAccount(account OAuthAccount) error {
	s.accounts = append(s.accounts, account)
	return nil
}
