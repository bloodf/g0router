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

func TestCreateUserFullPersistsDisplayNameAndRole(t *testing.T) {
	st := newTestStore(t)

	u, err := st.CreateUserFull("alice", "hash-a", "Alice Admin", "admin")
	if err != nil {
		t.Fatalf("CreateUserFull: %v", err)
	}
	if u.DisplayName != "Alice Admin" || u.Role != "admin" {
		t.Fatalf("CreateUserFull returned %+v, want display_name/role set", u)
	}

	got, err := st.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.DisplayName != "Alice Admin" || got.Role != "admin" {
		t.Fatalf("persisted user = %+v, want display_name/role", got)
	}

	// CreateUser keeps the existing two-arg signature and defaults role.
	u2, err := st.CreateUser("bob", "hash-b")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u2.Username != "bob" {
		t.Fatalf("CreateUser username = %q", u2.Username)
	}
}

func TestListUsers(t *testing.T) {
	st := newTestStore(t)

	if _, err := st.CreateUserFull("admin", "h1", "Administrator", "admin"); err != nil {
		t.Fatalf("CreateUserFull admin: %v", err)
	}
	if _, err := st.CreateUserFull("editor", "h2", "Editor", "user"); err != nil {
		t.Fatalf("CreateUserFull editor: %v", err)
	}

	users, err := st.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	// Ordered by created_at ASC: admin first.
	if users[0].Username != "admin" || users[1].Username != "editor" {
		t.Fatalf("order = %s, %s", users[0].Username, users[1].Username)
	}
	if users[1].DisplayName != "Editor" || users[1].Role != "user" {
		t.Fatalf("user fields = %+v", users[1])
	}
}

func TestDeleteUser(t *testing.T) {
	st := newTestStore(t)

	u, err := st.CreateUser("admin", "h")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := st.DeleteUser(u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if _, err := st.GetUserByID(u.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted user err = %v, want ErrNotFound", err)
	}

	// Unknown id returns ErrNotFound.
	if err := st.DeleteUser("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown err = %v, want ErrNotFound", err)
	}
}
