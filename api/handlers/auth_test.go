package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestAuthSetupSuccess(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123","display_name":"Admin User"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, s, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("setup status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
		} `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if resp.Data.Username != "admin" {
		t.Fatalf("username = %q, want admin", resp.Data.Username)
	}
	if resp.Data.Role != "admin" {
		t.Fatalf("role = %q, want admin", resp.Data.Role)
	}
	if resp.Data.DisplayName != "Admin User" {
		t.Fatalf("display_name = %q, want Admin User", resp.Data.DisplayName)
	}

	cookie := extractSessionCookie(t, ctx)
	if cookie == "" {
		t.Fatal("expected session cookie to be set")
	}
}

func TestAuthSetupAlreadyDone(t *testing.T) {
	s := newHandlerStore(t)

	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"username":"admin2","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, s, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("setup status = %d, want 409; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "setup already completed") {
		t.Fatalf("body = %s, want setup already completed", body)
	}
}

func TestAuthLoginSuccess(t *testing.T) {
	s := newHandlerStore(t)

	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, s, s, auth.NewLoginRateLimiter(), s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("login status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
		} `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if resp.Data.Username != "admin" {
		t.Fatalf("username = %q, want admin", resp.Data.Username)
	}

	cookie := extractSessionCookie(t, ctx)
	if cookie == "" {
		t.Fatal("expected session cookie to be set")
	}
}

func TestAuthLoginWrongPassword(t *testing.T) {
	s := newHandlerStore(t)

	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"wrong"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, s, s, auth.NewLoginRateLimiter(), s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("login status = %d, want 401; body=%s", ctx.Response.StatusCode(), body)
	}

	cookie := extractSessionCookie(t, ctx)
	if cookie != "" {
		t.Fatalf("expected no session cookie, got %q", cookie)
	}
}

func TestAuthLoginRateLimit(t *testing.T) {
	s := newHandlerStore(t)

	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	limiter := auth.NewLoginRateLimiter()
	ip := "127.0.0.1"

	// 5 failed attempts
	for i := 0; i < 5; i++ {
		ctx, _ := runHandlerWithIP(t, fasthttp.MethodPost, `{"username":"admin","password":"wrong"}`, ip, func(ctx *fasthttp.RequestCtx) {
			AuthLogin(ctx, s, s, limiter, s)
		})
		if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, ctx.Response.StatusCode())
		}
	}

	// 6th attempt should be rate limited
	ctx, body := runHandlerWithIP(t, fasthttp.MethodPost, `{"username":"admin","password":"wrong"}`, ip, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, s, s, limiter, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("6th attempt status = %d, want 429; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Error             string `json:"error"`
		RetryAfterSeconds int    `json:"retry_after_seconds"`
	}
	decodeJSON(t, body, &resp)
	if resp.RetryAfterSeconds <= 0 {
		t.Fatalf("retry_after_seconds = %d, want > 0", resp.RetryAfterSeconds)
	}
}

func TestAuthLogout(t *testing.T) {
	s := newHandlerStore(t)

	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a session manually
	rawToken := "testtoken123"
	userID, _ := strconv.ParseInt(user.ID, 10, 64)
	if err := s.CreateDashboardSession(userID, rawToken, "test-agent", "127.0.0.1", time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	ctx, body := runHandlerWithCookie(t, fasthttp.MethodPost, "", "g0router_session="+rawToken, func(ctx *fasthttp.RequestCtx) {
		AuthLogout(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("logout status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
	if len(body) != 0 {
		t.Fatalf("logout body should be empty, got %s", body)
	}

	// Verify cookie is cleared
	cookie := extractSessionCookie(t, ctx)
	if cookie != "" {
		t.Fatalf("expected cleared cookie, got %q", cookie)
	}

	// Verify session is deleted
	tokenHash := hashRawToken(rawToken)
	_, err = s.GetDashboardSessionByTokenHash(tokenHash)
	if err == nil {
		t.Fatal("expected session to be deleted")
	}
}

func TestAuthStatusNoUsers(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, s, s, false)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data struct {
			RequireLogin  bool   `json:"require_login"`
			HasUsers      bool   `json:"has_users"`
			Authenticated bool   `json:"authenticated"`
			Username      string `json:"username"`
			DisplayName   string `json:"display_name"`
			Role          string `json:"role"`
		} `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if resp.Data.RequireLogin {
		t.Fatal("require_login should be false")
	}
	if resp.Data.HasUsers {
		t.Fatal("has_users should be false")
	}
	if resp.Data.Authenticated {
		t.Fatal("authenticated should be false")
	}
}

func TestAuthStatusAuthenticated(t *testing.T) {
	s := newHandlerStore(t)

	user, err := s.CreateDashboardUser("admin", "password123", "Admin User", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a session manually
	rawToken := "testtoken456"
	userID, _ := strconv.ParseInt(user.ID, 10, 64)
	if err := s.CreateDashboardSession(userID, rawToken, "test-agent", "127.0.0.1", time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	ctx, body := runHandlerWithCookie(t, fasthttp.MethodGet, "", "g0router_session="+rawToken, func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, s, s, true)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data struct {
			RequireLogin  bool   `json:"require_login"`
			HasUsers      bool   `json:"has_users"`
			Authenticated bool   `json:"authenticated"`
			Username      string `json:"username"`
			DisplayName   string `json:"display_name"`
			Role          string `json:"role"`
		} `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if !resp.Data.RequireLogin {
		t.Fatal("require_login should be true")
	}
	if !resp.Data.HasUsers {
		t.Fatal("has_users should be true")
	}
	if !resp.Data.Authenticated {
		t.Fatal("authenticated should be true")
	}
	if resp.Data.Username != "admin" {
		t.Fatalf("username = %q, want admin", resp.Data.Username)
	}
	if resp.Data.DisplayName != "Admin User" {
		t.Fatalf("display_name = %q, want Admin User", resp.Data.DisplayName)
	}
	if resp.Data.Role != "admin" {
		t.Fatalf("role = %q, want admin", resp.Data.Role)
	}
}

func extractSessionCookie(t *testing.T, ctx *fasthttp.RequestCtx) string {
	t.Helper()
	cookieBytes := ctx.Response.Header.Peek("Set-Cookie")
	if len(cookieBytes) == 0 {
		return ""
	}
	var c fasthttp.Cookie
	if err := c.ParseBytes(cookieBytes); err != nil {
		t.Fatalf("parse cookie: %v", err)
	}
	return string(c.Value())
}

func runHandlerWithIP(t *testing.T, method, body, ip string, handler func(*fasthttp.RequestCtx)) (*fasthttp.RequestCtx, []byte) {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	if body != "" {
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBodyString(body)
	}
	if ip != "" {
		ctx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(ip)})
	}
	handler(&ctx)
	return &ctx, ctx.Response.Body()
}

func runHandlerWithCookie(t *testing.T, method, body, cookieHeader string, handler func(*fasthttp.RequestCtx)) (*fasthttp.RequestCtx, []byte) {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	if body != "" {
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBodyString(body)
	}
	if cookieHeader != "" {
		ctx.Request.Header.Set("Cookie", cookieHeader)
	}
	handler(&ctx)
	return &ctx, ctx.Response.Body()
}

func runHandlerWithSession(t *testing.T, method, body, userID, role string, handler func(*fasthttp.RequestCtx)) (*fasthttp.RequestCtx, []byte) {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	if body != "" {
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBodyString(body)
	}
	if userID != "" {
		ctx.SetUserValue("g0router.session_user_id", userID)
		ctx.SetUserValue("g0router.session_role", role)
	}
	handler(&ctx)
	return &ctx, ctx.Response.Body()
}

func lastAuditEntry(t *testing.T, s *store.Store, action string) *store.AuditEntry {
	t.Helper()
	entries, _, err := s.ListAudit(store.AuditFilter{Action: &action, Limit: 1})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) == 0 {
		return nil
	}
	return &entries[0]
}

func hashRawToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}

func TestAuthPasswordChangeWrongCurrent(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"wrong","new_password":"newpass123"}`, user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, s, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("password change status = %d, want 403; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAuthPasswordChangeSuccess(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID, _ := strconv.ParseInt(user.ID, 10, 64)

	oldToken := "oldsessiontoken"
	if err := s.CreateDashboardSession(userID, oldToken, "agent", "127.0.0.1", time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"password123","new_password":"newpass123"}`, user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, s, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("password change status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if resp.Data.Username != "admin" {
		t.Fatalf("username = %q, want admin", resp.Data.Username)
	}

	_, err = s.GetDashboardSessionByRawToken(oldToken)
	if err == nil {
		t.Fatal("expected old session to be invalidated")
	}

	cookie := extractSessionCookie(t, ctx)
	if cookie == "" {
		t.Fatal("expected new session cookie to be set")
	}

	updated, _ := s.GetDashboardUser(user.ID)
	if updated == nil || !s.VerifyDashboardUserPassword(updated, "newpass123") {
		t.Fatal("expected new password to verify")
	}
}

func TestAuthPasswordChangeShortPassword(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"password123","new_password":"short"}`, user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, s, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("password change status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "password") {
		t.Fatalf("body should mention password error, got %s", body)
	}
}

func TestAuthUsersListAdmin(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := s.CreateDashboardUser("user1", "password123", "User One", "user"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodGet, "", user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("users list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data []authUserResponse `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if len(resp.Data) != 2 {
		t.Fatalf("users count = %d, want 2", len(resp.Data))
	}
}

func TestAuthUsersListNonAdmin(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("user1", "password123", "User One", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodGet, "", user.ID, "user", func(ctx *fasthttp.RequestCtx) {
		AuthUsersList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("users list status = %d, want 403; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAuthUsersCreateAdmin(t *testing.T) {
	s := newHandlerStore(t)
	admin, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"newuser","password":"password123","display_name":"New User"}`, admin.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("users create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp struct {
		Data authUserResponse `json:"data"`
	}
	decodeJSON(t, body, &resp)
	if resp.Data.Username != "newuser" {
		t.Fatalf("username = %q, want newuser", resp.Data.Username)
	}
	if resp.Data.Role != "user" {
		t.Fatalf("role = %q, want user", resp.Data.Role)
	}
}

func TestAuthUsersCreateNonAdmin(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("user1", "password123", "User One", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"newuser","password":"password123"}`, user.ID, "user", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("users create status = %d, want 403; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAuthUsersCreateDuplicate(t *testing.T) {
	s := newHandlerStore(t)
	admin, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := s.CreateDashboardUser("existing", "password123", "Existing", "user"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"existing","password":"password123"}`, admin.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("users create status = %d, want 409; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAuthUsersDeleteAdmin(t *testing.T) {
	s := newHandlerStore(t)
	admin, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	user, err := s.CreateDashboardUser("user1", "password123", "User One", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	userID, _ := strconv.ParseInt(user.ID, 10, 64)
	if err := s.CreateDashboardSession(userID, "usertoken", "agent", "127.0.0.1", time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodDelete, "", admin.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, s, s, s, user.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("users delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
	if len(body) != 0 {
		t.Fatalf("delete body should be empty, got %s", body)
	}

	_, err = s.GetDashboardUser(user.ID)
	if err == nil {
		t.Fatal("expected user to be deleted")
	}

	_, err = s.GetDashboardSessionByRawToken("usertoken")
	if err == nil {
		t.Fatal("expected user sessions to be deleted")
	}
}

func TestAuthUsersDeleteLastAdmin(t *testing.T) {
	s := newHandlerStore(t)
	admin, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodDelete, "", admin.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, s, s, s, admin.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("users delete status = %d, want 409; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "cannot delete last admin") {
		t.Fatalf("body = %s, want cannot delete last admin", body)
	}
}

func TestAuthUsersDeleteNonAdmin(t *testing.T) {
	s := newHandlerStore(t)
	user1, err := s.CreateDashboardUser("user1", "password123", "User One", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	user2, err := s.CreateDashboardUser("user2", "password123", "User Two", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodDelete, "", user1.ID, "user", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, s, s, s, user2.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("users delete status = %d, want 403; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestAuthPasswordChangeAudit(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"password123","new_password":"newpass123"}`, user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, s, s, s)
	})

	entry := lastAuditEntry(t, s, "auth.password_change")
	if entry == nil {
		t.Fatal("expected audit entry for password change")
	}
	if entry.Target != "admin" {
		t.Fatalf("audit target = %q, want admin", entry.Target)
	}
}

func TestAuthUsersCreateAudit(t *testing.T) {
	s := newHandlerStore(t)
	admin, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"newuser","password":"password123"}`, admin.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, s, s)
	})

	entry := lastAuditEntry(t, s, "auth.user.create")
	if entry == nil {
		t.Fatal("expected audit entry for user create")
	}
	if entry.Target != "newuser" {
		t.Fatalf("audit target = %q, want newuser", entry.Target)
	}
}

func TestAuthUsersDeleteAudit(t *testing.T) {
	s := newHandlerStore(t)
	admin, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	user, err := s.CreateDashboardUser("user1", "password123", "User One", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	runHandlerWithSession(t, fasthttp.MethodDelete, "", admin.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, s, s, s, user.ID)
	})

	entry := lastAuditEntry(t, s, "auth.user.delete")
	if entry == nil {
		t.Fatal("expected audit entry for user delete")
	}
	if entry.Target != "user1" {
		t.Fatalf("audit target = %q, want user1", entry.Target)
	}
}

// --- Fake stores for error-path coverage ---

type fakeAuthUserStore struct {
	listUsers         []store.DashboardUser
	listErr           error
	createUser        *store.DashboardUser
	createErr         error
	getByUsername     *store.DashboardUser
	getByUsernameErr  error
	getUser           *store.DashboardUser
	getUserErr        error
	verifyResult      bool
	updatePasswordErr error
	deleteErr         error
}

func (f *fakeAuthUserStore) ListDashboardUsers() ([]store.DashboardUser, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listUsers, nil
}

func (f *fakeAuthUserStore) CreateDashboardUser(username, password, displayName, role string) (*store.DashboardUser, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createUser != nil {
		return f.createUser, nil
	}
	return &store.DashboardUser{
		ID:          "1",
		Username:    username,
		DisplayName: displayName,
		Role:        role,
	}, nil
}

func (f *fakeAuthUserStore) GetDashboardUserByUsername(username string) (*store.DashboardUser, error) {
	if f.getByUsernameErr != nil {
		return nil, f.getByUsernameErr
	}
	return f.getByUsername, nil
}

func (f *fakeAuthUserStore) VerifyDashboardUserPassword(user *store.DashboardUser, password string) bool {
	return f.verifyResult
}

func (f *fakeAuthUserStore) GetDashboardUser(id string) (*store.DashboardUser, error) {
	if f.getUserErr != nil {
		return nil, f.getUserErr
	}
	return f.getUser, nil
}

func (f *fakeAuthUserStore) UpdateDashboardUserPassword(id string, newPassword string) error {
	return f.updatePasswordErr
}

func (f *fakeAuthUserStore) DeleteDashboardUser(id string) error {
	return f.deleteErr
}

func (f *fakeAuthUserStore) CreateDashboardSession(userID int64, rawToken, userAgent, ip string, expiresAt time.Time) error {
	return nil
}

func (f *fakeAuthUserStore) GetDashboardSessionByTokenHash(tokenHash string) (*store.DashboardSession, error) {
	return nil, nil
}

func (f *fakeAuthUserStore) DeleteDashboardSession(tokenHash string) error {
	return nil
}

func (f *fakeAuthUserStore) DeleteDashboardSessionsByUserID(userID int64) error {
	return nil
}

func (f *fakeAuthUserStore) AppendAudit(entry store.AuditEntry) error {
	return nil
}

type fakeAuthSessionStore struct {
	createErr         error
	getSession        *store.DashboardSession
	getSessionErr     error
	deleteErr         error
	deleteByUserIDErr error
}

func (f *fakeAuthSessionStore) CreateDashboardSession(userID int64, rawToken, userAgent, ip string, expiresAt time.Time) error {
	return f.createErr
}

func (f *fakeAuthSessionStore) GetDashboardSessionByTokenHash(tokenHash string) (*store.DashboardSession, error) {
	if f.getSessionErr != nil {
		return nil, f.getSessionErr
	}
	return f.getSession, nil
}

func (f *fakeAuthSessionStore) DeleteDashboardSession(tokenHash string) error {
	return f.deleteErr
}

func (f *fakeAuthSessionStore) DeleteDashboardSessionsByUserID(userID int64) error {
	return f.deleteByUserIDErr
}

type fakeAuthAuditWriter struct {
	appendErr error
}

func (f *fakeAuthAuditWriter) AppendAudit(entry store.AuditEntry) error {
	return f.appendErr
}

type lateFailGetUserStore struct {
	*store.Store
	callCount int
}

func (f *lateFailGetUserStore) GetDashboardUser(id string) (*store.DashboardUser, error) {
	f.callCount++
	if f.callCount == 2 {
		return nil, errors.New("boom")
	}
	return f.Store.GetDashboardUser(id)
}

// --- AuthSetup error branches ---

func TestAuthSetupStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, nil, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthSetupListUsersError(t *testing.T) {
	users := &fakeAuthUserStore{listErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthSetupInvalidJSON(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthSetupCreateUserGenericError(t *testing.T) {
	users := &fakeAuthUserStore{createErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthSetupUserExists(t *testing.T) {
	users := &fakeAuthUserStore{createErr: store.ErrDashboardUserExists}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("status = %d, want 409", ctx.Response.StatusCode())
	}
}

func TestAuthSetupInvalidPassword(t *testing.T) {
	users := &fakeAuthUserStore{createErr: store.ErrInvalidDashboardUserPassword}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"short"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthSetupGenerateTokenError(t *testing.T) {
	users := &fakeAuthUserStore{}
	sessions := &fakeAuthSessionStore{}
	audit := &fakeAuthAuditWriter{}

	generateSessionTokenFunc = func() (string, error) { return "", errors.New("entropy exhausted") }
	defer func() { generateSessionTokenFunc = generateSessionToken }()

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, sessions, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthSetupCreateSessionError(t *testing.T) {
	users := &fakeAuthUserStore{}
	sessions := &fakeAuthSessionStore{createErr: errors.New("boom")}
	audit := &fakeAuthAuditWriter{}

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, sessions, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthSetupAuditError(t *testing.T) {
	users := &fakeAuthUserStore{}
	sessions := &fakeAuthSessionStore{}
	audit := &fakeAuthAuditWriter{appendErr: errors.New("audit boom")}

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, sessions, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201", ctx.Response.StatusCode())
	}
}

func TestAuthSetupParseUserIDError(t *testing.T) {
	users := &fakeAuthUserStore{createUser: &store.DashboardUser{ID: "not-a-number", Username: "admin", Role: "admin"}}
	sessions := &fakeAuthSessionStore{}
	audit := &fakeAuthAuditWriter{}

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthSetup(ctx, users, sessions, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

// --- AuthLogin error branches ---

func TestAuthLoginStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, nil, nil, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthLoginInvalidJSON(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, users, auth.NewLoginRateLimiter(), users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthLoginGetUserByUsernameError(t *testing.T) {
	users := &fakeAuthUserStore{getByUsernameErr: errors.New("boom")}
	limiter := auth.NewLoginRateLimiter()

	ctx, _ := runHandlerWithIP(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, "127.0.0.1", func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, users, limiter, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}

	// Verify failure was recorded by limiter (6th attempt should be 429)
	for i := 0; i < 4; i++ {
		runHandlerWithIP(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, "127.0.0.1", func(ctx *fasthttp.RequestCtx) {
			AuthLogin(ctx, users, users, limiter, users)
		})
	}
	ctx2, _ := runHandlerWithIP(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, "127.0.0.1", func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, users, limiter, users)
	})
	if ctx2.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx2.Response.StatusCode())
	}
}

func TestAuthLoginCreateSessionError(t *testing.T) {
	users := &fakeAuthUserStore{
		getByUsername: &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult:  true,
	}
	sessions := &fakeAuthSessionStore{createErr: errors.New("boom")}
	limiter := auth.NewLoginRateLimiter()

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, sessions, limiter, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthLoginGenerateTokenError(t *testing.T) {
	users := &fakeAuthUserStore{
		getByUsername: &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult:  true,
	}
	sessions := &fakeAuthSessionStore{}
	limiter := auth.NewLoginRateLimiter()

	generateSessionTokenFunc = func() (string, error) { return "", errors.New("entropy exhausted") }
	defer func() { generateSessionTokenFunc = generateSessionToken }()

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, sessions, limiter, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthLoginParseUserIDError(t *testing.T) {
	users := &fakeAuthUserStore{
		getByUsername: &store.DashboardUser{ID: "not-a-number", Username: "admin", Role: "admin"},
		verifyResult:  true,
	}
	sessions := &fakeAuthSessionStore{}
	limiter := auth.NewLoginRateLimiter()

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, sessions, limiter, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthLoginAuditError(t *testing.T) {
	users := &fakeAuthUserStore{
		getByUsername: &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult:  true,
	}
	sessions := &fakeAuthSessionStore{}
	limiter := auth.NewLoginRateLimiter()
	audit := &fakeAuthAuditWriter{appendErr: errors.New("audit boom")}

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"admin","password":"password123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthLogin(ctx, users, sessions, limiter, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

// --- AuthLogout error branches ---

func TestAuthLogoutStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AuthLogout(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthLogoutNoCookie(t *testing.T) {
	sessions := &fakeAuthSessionStore{}
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AuthLogout(ctx, sessions)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", ctx.Response.StatusCode())
	}
	if len(body) != 0 {
		t.Fatalf("body should be empty, got %s", body)
	}
}

func TestAuthLogoutDeleteSessionError(t *testing.T) {
	sessions := &fakeAuthSessionStore{deleteErr: errors.New("boom")}
	ctx, body := runHandlerWithCookie(t, fasthttp.MethodPost, "", "g0router_session=token123", func(ctx *fasthttp.RequestCtx) {
		AuthLogout(ctx, sessions)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", ctx.Response.StatusCode())
	}
	if len(body) != 0 {
		t.Fatalf("body should be empty, got %s", body)
	}
}

// --- AuthStatus error branches ---

func TestAuthStatusStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, nil, nil, false)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthStatusListUsersError(t *testing.T) {
	users := &fakeAuthUserStore{listErr: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, users, users, false)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthStatusSessionLookupError(t *testing.T) {
	users := &fakeAuthUserStore{}
	sessions := &fakeAuthSessionStore{getSessionErr: errors.New("boom")}

	ctx, _ := runHandlerWithCookie(t, fasthttp.MethodGet, "", "g0router_session=token123", func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, users, sessions, false)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestAuthStatusExpiredSession(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser: &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
	}
	sessions := &fakeAuthSessionStore{
		getSession: &store.DashboardSession{
			UserID:    1,
			ExpiresAt: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339),
		},
	}

	ctx, _ := runHandlerWithCookie(t, fasthttp.MethodGet, "", "g0router_session=token123", func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, users, sessions, true)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	var resp struct {
		Data struct {
			Authenticated bool `json:"authenticated"`
		} `json:"data"`
	}
	decodeJSON(t, ctx.Response.Body(), &resp)
	if resp.Data.Authenticated {
		t.Fatal("authenticated should be false for expired session")
	}
}

func TestAuthStatusSessionUserNotFound(t *testing.T) {
	users := &fakeAuthUserStore{getUserErr: errors.New("boom")}
	sessions := &fakeAuthSessionStore{
		getSession: &store.DashboardSession{
			UserID:    1,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		},
	}

	ctx, _ := runHandlerWithCookie(t, fasthttp.MethodGet, "", "g0router_session=token123", func(ctx *fasthttp.RequestCtx) {
		AuthStatus(ctx, users, sessions, true)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

// --- AuthPasswordChange error branches ---

func TestAuthPasswordChangeStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, nil, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeNoSession(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeInvalidJSON(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeGetUserError(t *testing.T) {
	users := &fakeAuthUserStore{getUserErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeUpdatePasswordError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser:           &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult:      true,
		updatePasswordErr: errors.New("boom"),
	}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeDeleteSessionsError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser:      &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult: true,
	}
	sessions := &fakeAuthSessionStore{deleteByUserIDErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, sessions, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeCreateSessionError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser:      &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult: true,
	}
	sessions := &fakeAuthSessionStore{createErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, sessions, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeGenerateTokenError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser:      &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult: true,
	}
	sessions := &fakeAuthSessionStore{}

	generateSessionTokenFunc = func() (string, error) { return "", errors.New("entropy exhausted") }
	defer func() { generateSessionTokenFunc = generateSessionToken }()

	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, sessions, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeParseUserIDError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser:      &store.DashboardUser{ID: "1", Username: "admin", Role: "admin"},
		verifyResult: true,
	}
	sessions := &fakeAuthSessionStore{}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"old","new_password":"newpass123"}`, "not-a-number", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, sessions, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

// --- AuthUsersList error branches ---

func TestAuthUsersListStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AuthUsersList(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthUsersListNoSession(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AuthUsersList(ctx, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestAuthUsersListListUsersError(t *testing.T) {
	users := &fakeAuthUserStore{listErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodGet, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersList(ctx, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

// --- AuthUsersCreate error branches ---

func TestAuthUsersCreateStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"new","password":"pass123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthUsersCreateNoSession(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"username":"new","password":"pass123"}`, func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestAuthUsersCreateInvalidJSON(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPost, `{`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthUsersCreateGenericError(t *testing.T) {
	users := &fakeAuthUserStore{createErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"new","password":"pass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthUsersCreateInvalidPassword(t *testing.T) {
	users := &fakeAuthUserStore{createErr: store.ErrInvalidDashboardUserPassword}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"new","password":"short"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthUsersCreateInvalidRole(t *testing.T) {
	users := &fakeAuthUserStore{createErr: store.ErrInvalidDashboardUserRole}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"new","password":"pass123","role":"invalid"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, users, users)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthUsersCreateAuditError(t *testing.T) {
	users := &fakeAuthUserStore{}
	audit := &fakeAuthAuditWriter{appendErr: errors.New("audit boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPost, `{"username":"new","password":"pass123"}`, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersCreate(ctx, users, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201", ctx.Response.StatusCode())
	}
}

// --- AuthUsersDelete error branches ---

func TestAuthUsersDeleteStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, nil, nil, nil, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteNoSession(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, users, users, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteMissingID(t *testing.T) {
	users := &fakeAuthUserStore{}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, users, users, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteInvalidID(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser: &store.DashboardUser{ID: "not-a-number", Username: "user2", Role: "user"},
	}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, users, users, "not-a-number")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteGetUserError(t *testing.T) {
	users := &fakeAuthUserStore{getUserErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, users, users, "2")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteListUsersError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser: &store.DashboardUser{ID: "2", Username: "admin2", Role: "admin"},
		listErr: errors.New("boom"),
	}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, users, users, "2")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteDeleteSessionsError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser: &store.DashboardUser{ID: "2", Username: "user2", Role: "user"},
	}
	sessions := &fakeAuthSessionStore{deleteByUserIDErr: errors.New("boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, sessions, users, "2")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteDeleteUserError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser:   &store.DashboardUser{ID: "2", Username: "user2", Role: "user"},
		deleteErr: errors.New("boom"),
	}
	sessions := &fakeAuthSessionStore{}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, sessions, users, "2")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuthUsersDeleteAuditError(t *testing.T) {
	users := &fakeAuthUserStore{
		getUser: &store.DashboardUser{ID: "2", Username: "user2", Role: "user"},
	}
	sessions := &fakeAuthSessionStore{}
	audit := &fakeAuthAuditWriter{appendErr: errors.New("audit boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodDelete, "", "1", "admin", func(ctx *fasthttp.RequestCtx) {
		AuthUsersDelete(ctx, users, sessions, audit, "2")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeAuditErrorAtEnd(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	audit := &fakeAuthAuditWriter{appendErr: errors.New("audit boom")}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"password123","new_password":"newpass123"}`, user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, s, s, audit)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestAuthPasswordChangeGetUpdatedUserError(t *testing.T) {
	s := newHandlerStore(t)
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	users := &lateFailGetUserStore{Store: s}
	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPut, `{"current_password":"password123","new_password":"newpass123"}`, user.ID, "admin", func(ctx *fasthttp.RequestCtx) {
		AuthPasswordChange(ctx, users, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestClientIPFromCtxUserValue(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.SetUserValue("g0router.client_ip", "192.168.1.1")
	ip := clientIPFromCtx(&ctx)
	if ip != "192.168.1.1" {
		t.Fatalf("ip = %q, want 192.168.1.1", ip)
	}
}
