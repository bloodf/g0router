package store

import (
	"errors"
	"strings"
	"testing"
)

func TestCreateDashboardUser(t *testing.T) {
	s := openTestStore(t)

	user, err := s.CreateDashboardUser("alice", "password123", "Alice", "admin")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}
	if user.ID == "" {
		t.Error("ID should be set")
	}
	if user.Username != "alice" {
		t.Errorf("Username = %q, want alice", user.Username)
	}
	if user.PasswordHash == "" {
		t.Error("PasswordHash should be set")
	}
	if user.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want Alice", user.DisplayName)
	}
	if user.Role != "admin" {
		t.Errorf("Role = %q, want admin", user.Role)
	}
	if user.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
}

func TestCreateDashboardUserDefaultsRole(t *testing.T) {
	s := openTestStore(t)

	user, err := s.CreateDashboardUser("bob", "password123", "Bob", "")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}
	if user.Role != "user" {
		t.Errorf("Role = %q, want user", user.Role)
	}
}

func TestCreateDashboardUserInvalidRole(t *testing.T) {
	s := openTestStore(t)

	_, err := s.CreateDashboardUser("charlie", "password123", "Charlie", "superuser")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestCreateDashboardUserEmptyPassword(t *testing.T) {
	s := openTestStore(t)

	cases := []string{
		"",
		"   ",
		"\t\n",
		"short",
		"1234567",
	}

	for _, pw := range cases {
		_, err := s.CreateDashboardUser("dave", pw, "Dave", "user")
		if err == nil {
			t.Fatalf("expected error for password %q", pw)
		}
	}
}

func TestCreateDashboardUserDuplicateUsername(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateDashboardUser("alice", "password123", "Alice", "user"); err != nil {
		t.Fatalf("first CreateDashboardUser: %v", err)
	}
	_, err := s.CreateDashboardUser("alice", "password456", "Alice2", "user")
	if err == nil {
		t.Fatal("second CreateDashboardUser should fail")
	}
	if !errors.Is(err, ErrDashboardUserExists) {
		t.Fatalf("expected ErrDashboardUserExists, got %v", err)
	}
}

func TestGetDashboardUser(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	got, err := s.GetDashboardUser(created.ID)
	if err != nil {
		t.Fatalf("GetDashboardUser: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Username != "alice" {
		t.Errorf("Username = %q, want alice", got.Username)
	}
}

func TestGetDashboardUserNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetDashboardUser("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestGetDashboardUserByUsername(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	got, err := s.GetDashboardUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetDashboardUserByUsername: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Username != "alice" {
		t.Errorf("Username = %q, want alice", got.Username)
	}
}

func TestGetDashboardUserByUsernameNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetDashboardUserByUsername("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestListDashboardUsers(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateDashboardUser("alice", "password123", "Alice", "user"); err != nil {
		t.Fatalf("CreateDashboardUser alice: %v", err)
	}
	if _, err := s.CreateDashboardUser("bob", "password123", "Bob", "user"); err != nil {
		t.Fatalf("CreateDashboardUser bob: %v", err)
	}

	users, err := s.ListDashboardUsers()
	if err != nil {
		t.Fatalf("ListDashboardUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}

	// Should be ordered by created_at
	if users[0].Username != "alice" {
		t.Errorf("first user = %q, want alice", users[0].Username)
	}
	if users[1].Username != "bob" {
		t.Errorf("second user = %q, want bob", users[1].Username)
	}
}

func TestUpdateDashboardUser(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	newName := "Alice Smith"
	newRole := "admin"
	updated, err := s.UpdateDashboardUser(created.ID, nil, &newName, &newRole)
	if err != nil {
		t.Fatalf("UpdateDashboardUser: %v", err)
	}
	if updated.DisplayName != "Alice Smith" {
		t.Errorf("DisplayName = %q, want Alice Smith", updated.DisplayName)
	}
	if updated.Role != "admin" {
		t.Errorf("Role = %q, want admin", updated.Role)
	}
	if updated.Username != "alice" {
		t.Errorf("Username = %q, want alice", updated.Username)
	}
}

func TestUpdateDashboardUserUsername(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	newUsername := "alice2"
	updated, err := s.UpdateDashboardUser(created.ID, &newUsername, nil, nil)
	if err != nil {
		t.Fatalf("UpdateDashboardUser: %v", err)
	}
	if updated.Username != "alice2" {
		t.Errorf("Username = %q, want alice2", updated.Username)
	}
}

func TestUpdateDashboardUserDuplicateUsername(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateDashboardUser("alice", "password123", "Alice", "user"); err != nil {
		t.Fatalf("CreateDashboardUser alice: %v", err)
	}
	bob, err := s.CreateDashboardUser("bob", "password123", "Bob", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser bob: %v", err)
	}

	newUsername := "alice"
	_, err = s.UpdateDashboardUser(bob.ID, &newUsername, nil, nil)
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
	if !errors.Is(err, ErrDashboardUserExists) {
		t.Fatalf("expected ErrDashboardUserExists, got %v", err)
	}
}

func TestUpdateDashboardUserInvalidRole(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	badRole := "superuser"
	_, err = s.UpdateDashboardUser(created.ID, nil, nil, &badRole)
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestDeleteDashboardUser(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	if err := s.DeleteDashboardUser(created.ID); err != nil {
		t.Fatalf("DeleteDashboardUser: %v", err)
	}

	_, err = s.GetDashboardUser(created.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestVerifyDashboardUserPassword(t *testing.T) {
	s := openTestStore(t)

	user, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	if !s.VerifyDashboardUserPassword(user, "password123") {
		t.Error("VerifyDashboardUserPassword should return true for correct password")
	}
	if s.VerifyDashboardUserPassword(user, "wrongpassword") {
		t.Error("VerifyDashboardUserPassword should return false for wrong password")
	}
}

func TestUpdateDashboardUserPassword(t *testing.T) {
	s := openTestStore(t)

	user, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	if err := s.UpdateDashboardUserPassword(user.ID, "newpassword456"); err != nil {
		t.Fatalf("UpdateDashboardUserPassword: %v", err)
	}

	updated, err := s.GetDashboardUser(user.ID)
	if err != nil {
		t.Fatalf("GetDashboardUser: %v", err)
	}

	if s.VerifyDashboardUserPassword(updated, "password123") {
		t.Error("old password should not work after update")
	}
	if !s.VerifyDashboardUserPassword(updated, "newpassword456") {
		t.Error("new password should work after update")
	}
}

func TestUpdateDashboardUserPasswordInvalid(t *testing.T) {
	s := openTestStore(t)

	user, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	cases := []string{
		"",
		"   ",
		"\t\n",
		"short",
		"1234567",
	}

	for _, pw := range cases {
		err := s.UpdateDashboardUserPassword(user.ID, pw)
		if err == nil {
			t.Fatalf("expected error for password %q", pw)
		}
	}
}

func TestDashboardUserPasswordHashIsBcrypt(t *testing.T) {
	s := openTestStore(t)

	user, err := s.CreateDashboardUser("alice", "password123", "Alice", "user")
	if err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	if !strings.HasPrefix(user.PasswordHash, "$2a$") && !strings.HasPrefix(user.PasswordHash, "$2b$") {
		t.Errorf("PasswordHash should start with bcrypt prefix, got %q", user.PasswordHash)
	}
}
