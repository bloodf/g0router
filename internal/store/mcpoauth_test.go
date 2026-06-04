package store

import (
	"errors"
	"testing"
	"time"
)

func TestMCPOAuthFlowIsSingleUseAndInstanceScoped(t *testing.T) {
	s := openTestStore(t)
	instance := createOAuthTestInstance(t, s, "linear-work")
	other := createOAuthTestInstance(t, s, "linear-personal")
	flow := &MCPOAuthFlow{
		InstanceID:         instance.ID,
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost/callback",
		AuthorizationURL:   "https://auth.example/authorize",
		ResourceURI:        "https://mcp.example",
		ClientID:           "client-123",
		ClientSecret:       "client-secret",
		ExpiresAt:          time.Now().Add(time.Hour),
	}

	if err := s.CreateMCPOAuthFlow(flow); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}
	if _, err := s.ConsumeMCPOAuthFlow(other.ID, "state-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong instance err = %v, want ErrNotFound", err)
	}

	consumed, err := s.ConsumeMCPOAuthFlow(instance.ID, "state-1")
	if err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow: %v", err)
	}
	if consumed.CodeVerifierSecret != "verifier" {
		t.Fatalf("verifier = %q, want verifier", consumed.CodeVerifierSecret)
	}
	if consumed.ClientID != "client-123" || consumed.ClientSecret != "client-secret" {
		t.Fatal("client credentials did not round trip")
	}
	if _, err := s.ConsumeMCPOAuthFlow(instance.ID, "state-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second consume err = %v, want ErrNotFound", err)
	}
}

func TestMCPOAuthAccountsAreScopedToInstance(t *testing.T) {
	s := openTestStore(t)
	work := createOAuthTestInstance(t, s, "atlassian-work")
	personal := createOAuthTestInstance(t, s, "atlassian-personal")

	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   work.ID,
		AccountLabel: "work",
		Subject:      "user-work",
		Email:        "work@example.com",
		Issuer:       "https://auth.atlassian.com",
		ResourceURI:  "https://mcp.atlassian.com",
		Scopes:       []string{"read:jira"},
		AccessToken:  "work-token",
		RefreshToken: "work-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
		AuthMetadata: map[string]string{"team": "work"},
	}); err != nil {
		t.Fatalf("Upsert work: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   personal.ID,
		AccountLabel: "personal",
		Subject:      "user-personal",
		ResourceURI:  "https://mcp.atlassian.com",
		Scopes:       []string{"read:jira"},
		AccessToken:  "personal-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("Upsert personal: %v", err)
	}

	workAccounts, err := s.ListMCPOAuthAccounts(work.ID)
	if err != nil {
		t.Fatalf("List work: %v", err)
	}
	personalAccounts, err := s.ListMCPOAuthAccounts(personal.ID)
	if err != nil {
		t.Fatalf("List personal: %v", err)
	}
	if len(workAccounts) != 1 || workAccounts[0].AccessToken != "work-token" {
		t.Fatalf("work accounts = %+v, want work token only", workAccounts)
	}
	if len(personalAccounts) != 1 || personalAccounts[0].AccessToken != "personal-token" {
		t.Fatalf("personal accounts = %+v, want personal token only", personalAccounts)
	}
}

func TestMCPOAuthAccountUpsertUpdatesSameInstanceAccount(t *testing.T) {
	s := openTestStore(t)
	instance := createOAuthTestInstance(t, s, "linear")
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "work",
		ResourceURI:  "https://mcp.linear.app",
		Scopes:       []string{"read"},
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Minute),
		AuthMetadata: map[string]string{"token_endpoint": "https://auth.example/token"},
	}); err != nil {
		t.Fatalf("Upsert initial: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "work",
		ResourceURI:  "https://mcp.linear.app",
		Scopes:       []string{"read", "write"},
		AccessToken:  "new-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		AuthMetadata: map[string]string{"token_endpoint": "https://auth.example/token"},
	}); err != nil {
		t.Fatalf("Upsert refresh: %v", err)
	}

	accounts, err := s.ListMCPOAuthAccounts(instance.ID)
	if err != nil {
		t.Fatalf("ListMCPOAuthAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].AccountLabel != "work" || accounts[0].AccessToken != "new-token" {
		t.Fatalf("accounts = %+v, want updated same account", accounts)
	}
}

func TestMCPOAuthValidAccountRejectsExpiredOrWrongResource(t *testing.T) {
	s := openTestStore(t)
	instance := createOAuthTestInstance(t, s, "linear")
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "expired",
		ResourceURI:  "https://mcp.linear.app",
		AccessToken:  "expired-token",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Upsert expired: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "wrong-resource",
		ResourceURI:  "https://other.example",
		AccessToken:  "wrong-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("Upsert wrong resource: %v", err)
	}

	_, err := s.GetValidMCPOAuthAccount(instance.ID, "https://mcp.linear.app")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func createOAuthTestInstance(t *testing.T, s *Store, name string) *MCPInstance {
	t.Helper()
	instance := &MCPInstance{
		Name:       name,
		ServerKey:  "linear",
		LaunchType: "http",
		Transport:  "streamable-http",
		URL:        strPtr("https://mcp.example/mcp"),
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	return instance
}
