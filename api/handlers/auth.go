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
}

type dashboardSessionStore interface {
	CreateDashboardSession(userID int64, rawToken, userAgent, ip string, expiresAt time.Time) error
	GetDashboardSessionByTokenHash(tokenHash string) (*store.DashboardSession, error)
	DeleteDashboardSession(tokenHash string) error
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

	rawToken, err := generateSessionToken()
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

	rawToken, err := generateSessionToken()
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

func clientIPFromCtx(ctx *fasthttp.RequestCtx) string {
	if ip, ok := ctx.UserValue("g0router.client_ip").(string); ok && ip != "" {
		return ip
	}
	return ctx.RemoteIP().String()
}

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
