package store

import (
	"reflect"
	"testing"
	"time"
)

func TestUpdateAPIKeyPolicyRoundTrip(t *testing.T) {
	s := openTestStore(t)

	created, _, err := s.CreateAPIKey("policy", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	exp := time.Now().Add(time.Hour).Unix()
	rpm := 60
	tpm := 10000
	cap := 5.50
	policy := APIKeyPolicy{
		ExpiresAt:        &exp,
		Scopes:           []string{"gpt-*", "claude-*"},
		RateLimitRPM:     &rpm,
		RateLimitTPM:     &tpm,
		DailySpendCapUSD: &cap,
	}
	if err := s.UpdateAPIKeyPolicy(created.ID, policy); err != nil {
		t.Fatalf("UpdateAPIKeyPolicy: %v", err)
	}

	got, err := s.GetAPIKey(created.ID)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got.ExpiresAt == nil || *got.ExpiresAt != exp {
		t.Errorf("ExpiresAt = %v, want %d", got.ExpiresAt, exp)
	}
	if !reflect.DeepEqual(got.Scopes, []string{"gpt-*", "claude-*"}) {
		t.Errorf("Scopes = %v", got.Scopes)
	}
	if got.RateLimitRPM == nil || *got.RateLimitRPM != rpm {
		t.Errorf("RateLimitRPM = %v", got.RateLimitRPM)
	}
	if got.RateLimitTPM == nil || *got.RateLimitTPM != tpm {
		t.Errorf("RateLimitTPM = %v", got.RateLimitTPM)
	}
	if got.DailySpendCapUSD == nil || *got.DailySpendCapUSD != cap {
		t.Errorf("DailySpendCapUSD = %v", got.DailySpendCapUSD)
	}
}

func TestUpdateAPIKeyPolicyReflectedInValidateAndList(t *testing.T) {
	s := openTestStore(t)

	created, raw, err := s.CreateAPIKey("policy", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	rpm := 30
	if err := s.UpdateAPIKeyPolicy(created.ID, APIKeyPolicy{Scopes: []string{"gpt-*"}, RateLimitRPM: &rpm}); err != nil {
		t.Fatalf("UpdateAPIKeyPolicy: %v", err)
	}

	validated, ok, err := s.ValidateAPIKey(raw, "test-secret")
	if err != nil || !ok {
		t.Fatalf("ValidateAPIKey ok=%v err=%v", ok, err)
	}
	if validated.RateLimitRPM == nil || *validated.RateLimitRPM != rpm {
		t.Errorf("validated RateLimitRPM = %v", validated.RateLimitRPM)
	}
	if len(validated.Scopes) != 1 || validated.Scopes[0] != "gpt-*" {
		t.Errorf("validated Scopes = %v", validated.Scopes)
	}

	keys, err := s.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 || len(keys[0].Scopes) != 1 {
		t.Fatalf("listed scopes = %v", keys[0].Scopes)
	}
}

func TestUpdateAPIKeyPolicyRejectsNegative(t *testing.T) {
	s := openTestStore(t)
	created, _, err := s.CreateAPIKey("policy", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	neg := -1
	if err := s.UpdateAPIKeyPolicy(created.ID, APIKeyPolicy{RateLimitRPM: &neg}); err == nil {
		t.Fatal("negative rpm should fail")
	}
	negTok := -5
	if err := s.UpdateAPIKeyPolicy(created.ID, APIKeyPolicy{RateLimitTPM: &negTok}); err == nil {
		t.Fatal("negative tpm should fail")
	}
	negCap := -0.5
	if err := s.UpdateAPIKeyPolicy(created.ID, APIKeyPolicy{DailySpendCapUSD: &negCap}); err == nil {
		t.Fatal("negative cap should fail")
	}
}

func TestUpdateAPIKeyPolicyUnknownID(t *testing.T) {
	s := openTestStore(t)
	if err := s.UpdateAPIKeyPolicy("missing", APIKeyPolicy{}); err == nil {
		t.Fatal("unknown id should fail")
	}
}

func TestGetAPIKeyExpiredStillReturned(t *testing.T) {
	s := openTestStore(t)
	created, _, err := s.CreateAPIKey("policy", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	past := time.Now().Add(-time.Hour).Unix()
	if err := s.UpdateAPIKeyPolicy(created.ID, APIKeyPolicy{ExpiresAt: &past}); err != nil {
		t.Fatalf("UpdateAPIKeyPolicy: %v", err)
	}
	got, err := s.GetAPIKey(created.ID)
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got.ExpiresAt == nil || *got.ExpiresAt != past {
		t.Errorf("ExpiresAt = %v, want %d", got.ExpiresAt, past)
	}
}
