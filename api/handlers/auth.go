package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type authSetupRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type authLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authPasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type authUsersCreateRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type authStatusResponse struct {
	RequireLogin  bool   `json:"require_login"`
	HasUsers      bool   `json:"has_users"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	Role          string `json:"role,omitempty"`
}

type authUserResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type rateLimitResponse struct {
	Error             string `json:"error"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}

type dashboardUserStore interface {
	ListDashboardUsers() ([]store.DashboardUser, error)
	CreateDashboardUser(username, password, displayName, role string) (*store.DashboardUser, error)
	GetDashboardUserByUsername(username string) (*store.DashboardUser, error)
	VerifyDashboardUserPassword(user *store.DashboardUser, password string) bool
	GetDashboardUser(id string) (*store.DashboardUser, error)
	UpdateDashboardUserPassword(id string, newPassword string) error
	DeleteDashboardUser(id string) error
}

type dashboardSessionStore interface {
	CreateDashboardSession(userID int64, rawToken, userAgent, ip string, expiresAt time.Time) error
	GetDashboardSessionByTokenHash(tokenHash string) (*store.DashboardSession, error)
	DeleteDashboardSession(tokenHash string) error
	DeleteDashboardSessionsByUserID(userID int64) error
}

type auditWriter interface {
	AppendAudit(store.AuditEntry) error
}

func newAuthUserResponse(u store.DashboardUser) authUserResponse {
	return authUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
}

// AuthSetup creates the first admin user and establishes a session.
func AuthSetup(ctx *fasthttp.RequestCtx, users dashboardUserStore, sessions dashboardSessionStore, audit auditWriter) {
	if isStoreNil(users) || isStoreNil(sessions) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	existing, err := users.ListDashboardUsers()
	if err != nil {
		log.Printf("list dashboard users: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to check setup status")
		return
	}
	if len(existing) > 0 {
		writeError(ctx, fasthttp.StatusConflict, "setup already completed")
		return
	}

	var req authSetupRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	user, err := users.CreateDashboardUser(req.Username, req.Password, req.DisplayName, "admin")
	if err != nil {
		if errors.Is(err, store.ErrDashboardUserExists) {
			writeError(ctx, fasthttp.StatusConflict, "username already exists")
			return
		}
		if errors.Is(err, store.ErrInvalidDashboardUserPassword) {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		log.Printf("create dashboard user: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create user")
		return
	}

	rawToken, err := generateSessionTokenFunc()
	if err != nil {
		log.Printf("generate session token: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	userID, err := strconv.ParseInt(user.ID, 10, 64)
	if err != nil {
		log.Printf("parse user id: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err := sessions.CreateDashboardSession(userID, rawToken, string(ctx.UserAgent()), clientIPFromCtx(ctx), expiresAt); err != nil {
		log.Printf("create dashboard session: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	setSessionCookie(ctx, rawToken)

	if err := audit.AppendAudit(store.AuditEntry{
		Action:  "auth.setup",
		Target:  req.Username,
		Details: "first admin user created",
	}); err != nil {
		log.Printf("append audit: %v", err)
	}

	writeJSON(ctx, fasthttp.StatusCreated, map[string]any{"data": newAuthUserResponse(*user)})
}

// AuthLogin verifies credentials and establishes a session.
func AuthLogin(ctx *fasthttp.RequestCtx, users dashboardUserStore, sessions dashboardSessionStore, limiter *auth.LoginRateLimiter, audit auditWriter) {
	if isStoreNil(users) || isStoreNil(sessions) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	ip := clientIPFromCtx(ctx)
	if limiter != nil {
		allowed, retryAfter := limiter.Check(ip)
		if !allowed {
			body, err := json.Marshal(rateLimitResponse{
				Error:             "too many login attempts",
				RetryAfterSeconds: retryAfter,
			})
			if err != nil {
				writeError(ctx, fasthttp.StatusInternalServerError, "marshal response")
				return
			}
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
			ctx.SetBody(body)
			return
		}
	}

	var req authLoginRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	user, err := users.GetDashboardUserByUsername(req.Username)
	if err != nil {
		if limiter != nil {
			limiter.RecordFailure(ip)
		}
		writeError(ctx, fasthttp.StatusUnauthorized, "invalid credentials")
		return
	}

	if !users.VerifyDashboardUserPassword(user, req.Password) {
		if limiter != nil {
			limiter.RecordFailure(ip)
		}
		writeError(ctx, fasthttp.StatusUnauthorized, "invalid credentials")
		return
	}

	rawToken, err := generateSessionTokenFunc()
	if err != nil {
		log.Printf("generate session token: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	userID, err := strconv.ParseInt(user.ID, 10, 64)
	if err != nil {
		log.Printf("parse user id: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err := sessions.CreateDashboardSession(userID, rawToken, string(ctx.UserAgent()), ip, expiresAt); err != nil {
		log.Printf("create dashboard session: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	setSessionCookie(ctx, rawToken)

	if err := audit.AppendAudit(store.AuditEntry{
		Action: "auth.login",
		Target: req.Username,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": newAuthUserResponse(*user)})
}

// AuthLogout deletes the current session and clears the cookie.
func AuthLogout(ctx *fasthttp.RequestCtx, sessions dashboardSessionStore) {
	if isStoreNil(sessions) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	rawToken := sessionCookieValue(ctx)
	if rawToken != "" {
		tokenHash := hashSessionToken(rawToken)
		if err := sessions.DeleteDashboardSession(tokenHash); err != nil {
			log.Printf("delete dashboard session: %v", err)
		}
	}

	clearSessionCookie(ctx)
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// AuthStatus returns the current authentication status.
func AuthStatus(ctx *fasthttp.RequestCtx, users dashboardUserStore, sessions dashboardSessionStore, requireLogin bool) {
	if isStoreNil(users) || isStoreNil(sessions) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	resp := authStatusResponse{
		RequireLogin: requireLogin,
	}

	existing, err := users.ListDashboardUsers()
	if err != nil {
		log.Printf("list dashboard users: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to check status")
		return
	}
	resp.HasUsers = len(existing) > 0

	rawToken := sessionCookieValue(ctx)
	if rawToken != "" {
		tokenHash := hashSessionToken(rawToken)
		session, err := sessions.GetDashboardSessionByTokenHash(tokenHash)
		if err == nil && session != nil {
			expiresAt, parseErr := time.Parse(time.RFC3339, session.ExpiresAt)
			if parseErr == nil && time.Now().UTC().Before(expiresAt) {
				user, err := users.GetDashboardUser(strconv.FormatInt(session.UserID, 10))
				if err == nil && user != nil {
					resp.Authenticated = true
					resp.Username = user.Username
					resp.DisplayName = user.DisplayName
					resp.Role = user.Role
				}
			}
		}
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": resp})
}

// AuthPasswordChange updates the current user's password and invalidates other sessions.
func AuthPasswordChange(ctx *fasthttp.RequestCtx, users dashboardUserStore, sessions dashboardSessionStore, audit auditWriter) {
	if isStoreNil(users) || isStoreNil(sessions) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	userID, ok := ctx.UserValue("g0router.session_user_id").(string)
	if !ok || userID == "" {
		writeError(ctx, fasthttp.StatusUnauthorized, "session required")
		return
	}

	user, err := users.GetDashboardUser(userID)
	if err != nil {
		log.Printf("get dashboard user: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to load user")
		return
	}

	var req authPasswordChangeRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	if !users.VerifyDashboardUserPassword(user, req.CurrentPassword) {
		writeError(ctx, fasthttp.StatusForbidden, "incorrect current password")
		return
	}

	if err := users.UpdateDashboardUserPassword(userID, req.NewPassword); err != nil {
		if errors.Is(err, store.ErrInvalidDashboardUserPassword) {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		log.Printf("update dashboard user password: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to update password")
		return
	}

	uidInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		log.Printf("parse user id: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to update password")
		return
	}

	if err := sessions.DeleteDashboardSessionsByUserID(uidInt); err != nil {
		log.Printf("delete dashboard sessions by user id: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to invalidate sessions")
		return
	}

	rawToken, err := generateSessionTokenFunc()
	if err != nil {
		log.Printf("generate session token: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err := sessions.CreateDashboardSession(uidInt, rawToken, string(ctx.UserAgent()), clientIPFromCtx(ctx), expiresAt); err != nil {
		log.Printf("create dashboard session: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create session")
		return
	}

	setSessionCookie(ctx, rawToken)

	if err := audit.AppendAudit(store.AuditEntry{
		Action: "auth.password_change",
		Target: user.Username,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}

	updatedUser, err := users.GetDashboardUser(userID)
	if err != nil {
		log.Printf("get updated user: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to load updated user")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": newAuthUserResponse(*updatedUser)})
}

// AuthUsersList returns all dashboard users (admin only).
func AuthUsersList(ctx *fasthttp.RequestCtx, users dashboardUserStore) {
	if isStoreNil(users) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	role, _ := ctx.UserValue("g0router.session_role").(string)
	if role != "admin" {
		writeError(ctx, fasthttp.StatusForbidden, "admin access required")
		return
	}

	list, err := users.ListDashboardUsers()
	if err != nil {
		log.Printf("list dashboard users: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list users")
		return
	}

	resp := make([]authUserResponse, 0, len(list))
	for _, u := range list {
		resp = append(resp, newAuthUserResponse(u))
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": resp})
}

// AuthUsersCreate creates a new dashboard user (admin only).
func AuthUsersCreate(ctx *fasthttp.RequestCtx, users dashboardUserStore, audit auditWriter) {
	if isStoreNil(users) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	role, _ := ctx.UserValue("g0router.session_role").(string)
	if role != "admin" {
		writeError(ctx, fasthttp.StatusForbidden, "admin access required")
		return
	}

	var req authUsersCreateRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	user, err := users.CreateDashboardUser(req.Username, req.Password, req.DisplayName, req.Role)
	if err != nil {
		if errors.Is(err, store.ErrDashboardUserExists) {
			writeError(ctx, fasthttp.StatusConflict, "username already exists")
			return
		}
		if errors.Is(err, store.ErrInvalidDashboardUserPassword) || errors.Is(err, store.ErrInvalidDashboardUserRole) {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		log.Printf("create dashboard user: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create user")
		return
	}

	if err := audit.AppendAudit(store.AuditEntry{
		Action: "auth.user.create",
		Target: req.Username,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}

	writeJSON(ctx, fasthttp.StatusCreated, map[string]any{"data": newAuthUserResponse(*user)})
}

// AuthUsersDelete deletes a dashboard user (admin only), with last-admin guard.
func AuthUsersDelete(ctx *fasthttp.RequestCtx, users dashboardUserStore, sessions dashboardSessionStore, audit auditWriter, id string) {
	if isStoreNil(users) || isStoreNil(sessions) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	role, _ := ctx.UserValue("g0router.session_role").(string)
	if role != "admin" {
		writeError(ctx, fasthttp.StatusForbidden, "admin access required")
		return
	}

	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "user id required")
		return
	}

	user, err := users.GetDashboardUser(id)
	if err != nil {
		log.Printf("get dashboard user: %v", err)
		writeError(ctx, fasthttp.StatusNotFound, "user not found")
		return
	}

	if user.Role == "admin" {
		allUsers, err := users.ListDashboardUsers()
		if err != nil {
			log.Printf("list dashboard users: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to check admin count")
			return
		}
		adminCount := 0
		for _, u := range allUsers {
			if u.Role == "admin" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			writeError(ctx, fasthttp.StatusConflict, "cannot delete last admin")
			return
		}
	}

	userID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Printf("parse user id: %v", err)
		writeError(ctx, fasthttp.StatusBadRequest, "invalid user id")
		return
	}

	if err := sessions.DeleteDashboardSessionsByUserID(userID); err != nil {
		log.Printf("delete dashboard sessions by user id: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete user sessions")
		return
	}

	if err := users.DeleteDashboardUser(id); err != nil {
		log.Printf("delete dashboard user: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete user")
		return
	}

	if err := audit.AppendAudit(store.AuditEntry{
		Action: "auth.user.delete",
		Target: user.Username,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

func clientIPFromCtx(ctx *fasthttp.RequestCtx) string {
	if ip, ok := ctx.UserValue("g0router.client_ip").(string); ok && ip != "" {
		return ip
	}
	return ctx.RemoteIP().String()
}

var generateSessionTokenFunc = generateSessionToken

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func setSessionCookie(ctx *fasthttp.RequestCtx, rawToken string) {
	var c fasthttp.Cookie
	c.SetKey("g0router_session")
	c.SetValue(rawToken)
	c.SetHTTPOnly(true)
	c.SetSameSite(fasthttp.CookieSameSiteStrictMode)
	c.SetPath("/")
	c.SetSecure(ctx.IsTLS())
	c.SetMaxAge(7 * 24 * 60 * 60)
	ctx.Response.Header.SetCookie(&c)
}

func clearSessionCookie(ctx *fasthttp.RequestCtx) {
	var c fasthttp.Cookie
	c.SetKey("g0router_session")
	c.SetValue("")
	c.SetHTTPOnly(true)
	c.SetSameSite(fasthttp.CookieSameSiteStrictMode)
	c.SetPath("/")
	c.SetSecure(ctx.IsTLS())
	c.SetMaxAge(-1)
	ctx.Response.Header.SetCookie(&c)
}

func sessionCookieValue(ctx *fasthttp.RequestCtx) string {
	return string(ctx.Request.Header.Cookie("g0router_session"))
}

func hashSessionToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}
