package admin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/valyala/fasthttp"
)

func TestAuthSetupGuardsWhenUsersExist(t *testing.T) {
	env := newTestEnv(t) // seeds an admin user

	status, envl := call(t, env.handlers.AuthSetup, "POST", "/api/auth/setup",
		`{"username":"new","password":"pw123456"}`, nil, nil)
	if status != fasthttp.StatusConflict {
		t.Fatalf("setup-when-users-exist status = %d, want 409 (err=%q)", status, errMessage(t, envl))
	}
}

func TestAuthSetupFirstUser(t *testing.T) {
	// Build an env with NO seeded users.
	env := newEmptyUserEnv(t)

	status, envl := call(t, env.handlers.AuthSetup, "POST", "/api/auth/setup",
		`{"username":"root","password":"pw123456","display_name":"Root"}`, nil, nil)
	if status != fasthttp.StatusOK && status != fasthttp.StatusCreated {
		t.Fatalf("setup status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("setup did not return a token: %v", data)
	}
	// Returned user carries display_name/role and no password/hash.
	raw, _ := json.Marshal(data)
	if strings.Contains(string(raw), "pw123456") || strings.Contains(string(raw), "pbkdf2-sha256$") {
		t.Fatalf("setup response leaks credentials: %s", raw)
	}

	// A second setup now fails (a user exists).
	status, _ = call(t, env.handlers.AuthSetup, "POST", "/api/auth/setup",
		`{"username":"again","password":"pw123456"}`, nil, nil)
	if status != fasthttp.StatusConflict {
		t.Fatalf("second setup status = %d, want 409", status)
	}
}

func TestChangePassword(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	// Set a known hash so VerifyPassword has a deterministic baseline.
	hash, err := auth.HashPassword("oldpass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := env.store.UpdateUserPassword(admin.ID, hash); err != nil {
		t.Fatalf("UpdateUserPassword: %v", err)
	}
	admin, _ = env.store.GetUserByID(admin.ID)

	// Wrong current → 400.
	status, envl := call(t, env.handlers.ChangePassword, "PUT", "/api/auth/password",
		`{"current_password":"WRONG","new_password":"newpass1"}`, map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("wrong-current status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); !strings.Contains(msg, "Current password is incorrect") {
		t.Fatalf("error = %q", msg)
	}

	// Correct current → 200 and the new password works for login.
	status, envl = call(t, env.handlers.ChangePassword, "PUT", "/api/auth/password",
		`{"current_password":"oldpass","new_password":"newpass1"}`, map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("change status = %d, err = %q", status, errMessage(t, envl))
	}
	// Response never echoes a password.
	raw, _ := json.Marshal(envl)
	if strings.Contains(string(raw), "newpass1") || strings.Contains(string(raw), "oldpass") {
		t.Fatalf("change-password response leaks a password: %s", raw)
	}
	if _, err := env.sessions.Login("admin", "newpass1"); err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}

func TestUsersListCreateDelete(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	// List strips the hash.
	status, envl := call(t, env.handlers.ListUsers, "GET", "/api/auth/users", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	listRaw, _ := json.Marshal(dataField[[]map[string]any](t, envl))
	if strings.Contains(string(listRaw), "password") || strings.Contains(string(listRaw), "pbkdf2-sha256$") {
		t.Fatalf("list leaks password/hash: %s", listRaw)
	}

	// Create.
	status, envl = call(t, env.handlers.CreateUser, "POST", "/api/auth/users",
		`{"username":"editor","display_name":"Editor","role":"user","password":"pw123456"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	newID, _ := created["id"].(string)
	if newID == "" || created["username"] != "editor" || created["display_name"] != "Editor" || created["role"] != "user" {
		t.Fatalf("created = %v", created)
	}
	rawCreated, _ := json.Marshal(created)
	if strings.Contains(string(rawCreated), "pw123456") || strings.Contains(string(rawCreated), "pbkdf2-sha256$") {
		t.Fatalf("create response leaks credentials: %s", rawCreated)
	}

	// Duplicate username → 409.
	status, _ = call(t, env.handlers.CreateUser, "POST", "/api/auth/users",
		`{"username":"editor","password":"pw123456"}`, nil, nil)
	if status != fasthttp.StatusConflict {
		t.Fatalf("dup-username status = %d, want 409", status)
	}

	// Delete missing → 404.
	status, _ = call(t, env.handlers.DeleteUser, "DELETE", "/api/auth/users/missing", "",
		map[string]any{"id": "missing", userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}

	// Delete the editor → 200.
	status, _ = call(t, env.handlers.DeleteUser, "DELETE", "/api/auth/users/"+newID, "",
		map[string]any{"id": newID, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}

	// Now only admin remains: delete-last-user is refused (400).
	status, envl = call(t, env.handlers.DeleteUser, "DELETE", "/api/auth/users/"+admin.ID, "",
		map[string]any{"id": admin.ID, userKey: admin}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("delete-last-user status = %d, want 400 (err=%q)", status, errMessage(t, envl))
	}
}

func newEmptyUserEnv(t *testing.T) *testEnv {
	t.Helper()
	env := newTestEnv(t)
	// Remove the seeded admin so CountUsers()==0 for the setup test.
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if err := env.store.DeleteUser(admin.ID); err != nil {
		t.Fatalf("DeleteUser seed admin: %v", err)
	}
	return env
}
