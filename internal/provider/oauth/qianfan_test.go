package oauth

import (
	"context"
	"strings"
	"testing"
)

func TestQianfanFlowExchangesAPIKey(t *testing.T) {
	flow := NewQianfanFlow()

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if flow.ProviderID() != ProviderID("qianfan") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("qianfan") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.AuthURL == "" {
		t.Error("auth url is empty")
	}

	token, err := flow.Exchange(context.Background(), session, "qianfan-key")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if token.Provider != ProviderID("qianfan") {
		t.Errorf("token provider = %q", token.Provider)
	}
	if token.AccessToken != "qianfan-key" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.TokenType != "api_key" {
		t.Errorf("token type = %q", token.TokenType)
	}
}

func TestQianfanFlowRejectsEmptyAPIKey(t *testing.T) {
	flow := NewQianfanFlow()

	_, err := flow.Exchange(context.Background(), AuthSession{Provider: ProviderID("qianfan")}, "")
	if err == nil {
		t.Fatal("exchange error is nil")
	}
	if !strings.Contains(err.Error(), "api key") {
		t.Fatalf("error = %q, want api key context", err.Error())
	}
}

func TestQianfanFlowPollUnsupported(t *testing.T) {
	flow := NewQianfanFlow()

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("qianfan")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
