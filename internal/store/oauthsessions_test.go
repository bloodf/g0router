package store

import (
	"errors"
	"testing"
	"time"
)

func TestOAuthSessionIsSingleUseAndStateScoped(t *testing.T) {
	s := openTestStore(t)
	session := &OAuthSession{
		State:        "state-1",
		Provider:     "anthropic",
		CodeVerifier: "verifier-1",
		RedirectURI:  "http://localhost/oauth/callback",
		AccountLabel: "work",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	if err := s.CreateOAuthSession(session); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}

	consumed, err := s.ConsumeOAuthSession("state-1")
	if err != nil {
		t.Fatalf("ConsumeOAuthSession: %v", err)
	}
	if consumed.Provider != "anthropic" || consumed.CodeVerifier != "verifier-1" || consumed.AccountLabel != "work" {
		t.Fatalf("consumed = %+v, want stored provider/verifier/account label", consumed)
	}
	if _, err := s.ConsumeOAuthSession("state-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second consume err = %v, want ErrNotFound", err)
	}
}

func TestOAuthSessionCanBeReadBeforeSingleUseConsume(t *testing.T) {
	s := openTestStore(t)
	session := &OAuthSession{
		State:        "poll-state",
		Provider:     "cursor",
		CodeVerifier: "poll-verifier",
		AccountLabel: "cursor-work",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	if err := s.CreateOAuthSession(session); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}

	read, err := s.GetOAuthSession("poll-state")
	if err != nil {
		t.Fatalf("GetOAuthSession: %v", err)
	}
	if read.Provider != "cursor" || read.CodeVerifier != "poll-verifier" || read.AccountLabel != "cursor-work" {
		t.Fatalf("read = %+v, want stored provider/verifier/account label", read)
	}

	consumed, err := s.ConsumeOAuthSession("poll-state")
	if err != nil {
		t.Fatalf("ConsumeOAuthSession: %v", err)
	}
	if consumed.CodeVerifier != "poll-verifier" {
		t.Fatalf("consumed verifier = %q", consumed.CodeVerifier)
	}
}

func TestOAuthSessionRejectsExpiredState(t *testing.T) {
	s := openTestStore(t)
	if err := s.CreateOAuthSession(&OAuthSession{
		State:        "expired-state",
		Provider:     "anthropic",
		CodeVerifier: "verifier",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}

	_, err := s.ConsumeOAuthSession("expired-state")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
