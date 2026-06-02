package oauth

import (
	"context"
	"strings"
	"testing"
)

func TestMiniMaxFlowExchangesAPIKey(t *testing.T) {
	flow := NewMiniMaxFlow()

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if flow.ProviderID() != ProviderID("minimax") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("minimax") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.AuthURL == "" {
		t.Error("auth url is empty")
	}

	token, err := flow.Exchange(context.Background(), session, "minimax-key")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if token.Provider != ProviderID("minimax") {
		t.Errorf("token provider = %q", token.Provider)
	}
	if token.AccessToken != "minimax-key" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.TokenType != "api_key" {
		t.Errorf("token type = %q", token.TokenType)
	}
}

func TestMiniMaxFlowRejectsEmptyAPIKey(t *testing.T) {
	flow := NewMiniMaxFlow()

	_, err := flow.Exchange(context.Background(), AuthSession{Provider: ProviderID("minimax")}, " ")
	if err == nil {
		t.Fatal("exchange error is nil")
	}
	if !strings.Contains(err.Error(), "api key") {
		t.Fatalf("error = %q, want api key context", err.Error())
	}
}

func TestMiniMaxFlowPollUnsupported(t *testing.T) {
	flow := NewMiniMaxFlow()

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("minimax")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
