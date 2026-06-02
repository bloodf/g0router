package oauth

import (
	"context"
	"strings"
	"testing"
)

func TestAlibabaFlowExchangesAPIKey(t *testing.T) {
	flow := NewAlibabaFlow()

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if flow.ProviderID() != ProviderID("alibaba") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("alibaba") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.AuthURL == "" {
		t.Error("auth url is empty")
	}

	token, err := flow.Exchange(context.Background(), session, "alibaba-key")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if token.Provider != ProviderID("alibaba") {
		t.Errorf("token provider = %q", token.Provider)
	}
	if token.AccessToken != "alibaba-key" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.TokenType != "api_key" {
		t.Errorf("token type = %q", token.TokenType)
	}
}

func TestAlibabaFlowRejectsProviderMismatch(t *testing.T) {
	flow := NewAlibabaFlow()

	_, err := flow.Exchange(context.Background(), AuthSession{Provider: ProviderID("zhipu")}, "alibaba-key")
	if err == nil {
		t.Fatal("exchange error is nil")
	}
	if !strings.Contains(err.Error(), "provider mismatch") {
		t.Fatalf("error = %q, want provider mismatch", err.Error())
	}
}

func TestAlibabaFlowPollUnsupported(t *testing.T) {
	flow := NewAlibabaFlow()

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("alibaba")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
