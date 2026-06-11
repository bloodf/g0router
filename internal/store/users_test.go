package store

import (
	"errors"
	"testing"
)

func TestSetUserPasswordHash(t *testing.T) {
	st := newTestStore(t)

	u, err := st.CreateUser("admin", "original-hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := st.SetUserPasswordHash("admin", "new-hash"); err != nil {
		t.Fatalf("SetUserPasswordHash: %v", err)
	}

	got, err := st.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.PasswordHash != "new-hash" {
		t.Fatalf("PasswordHash = %q, want %q", got.PasswordHash, "new-hash")
	}
	if got.ID != u.ID {
		t.Fatal("user ID changed")
	}

	// Empty hash is allowed (default-password path).
	if err := st.SetUserPasswordHash("admin", ""); err != nil {
		t.Fatalf("SetUserPasswordHash empty: %v", err)
	}
	got, err = st.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername after empty: %v", err)
	}
	if got.PasswordHash != "" {
		t.Fatalf("PasswordHash = %q, want empty", got.PasswordHash)
	}

	// Unknown user returns ErrNotFound.
	if err := st.SetUserPasswordHash("nobody", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown user err = %v, want ErrNotFound", err)
	}
}
