package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

const sessionCookieName = "g0_session"

// userKey is the request user value under which the authenticated user is stored.
const userKey = "admin_user"

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func toUserDTO(u *store.User) userDTO {
	return userDTO{ID: u.ID, Username: u.Username}
}

const resetHint = "Forgot password? Reset to default via g0router CLI: g0router reset-password"

func (h *Handlers) oidcConfigured(settings map[string]string) bool {
	return strings.TrimSpace(settings["oidc_issuer_url"]) != "" &&
		strings.TrimSpace(settings["oidc_client_id"]) != "" &&
		strings.TrimSpace(settings["oidc_client_secret"]) != ""
}

// Login handles POST /api/auth/login.
func (h *Handlers) Login(ctx *fasthttp.RequestCtx) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "username and password are required")
		return
	}

	ip := auth.ClientIP(
		string(ctx.Request.Header.Peek("X-Forwarded-For")),
		string(ctx.Request.Header.Peek("X-Real-Ip")),
	)

	locked, retryAfter := h.limiter.CheckLock(ip)
	if locked {
		h.writeLockout(ctx, retryAfter)
		return
	}

	settings, err := h.store.GetSettings()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load settings")
		return
	}

	// Block tunnel/tailscale dashboard login when access is disabled.
	if isTunnelRequest(ctx, settings) && settings["tunnelDashboardAccess"] != "true" {
		writeError(ctx, fasthttp.StatusForbidden, "Dashboard access via tunnel is disabled")
		return
	}

	authMode := settings["auth_mode"]
	if authMode == "" {
		authMode = "password"
	}
	if authMode == "oidc" && h.oidcConfigured(settings) {
		writeError(ctx, fasthttp.StatusForbidden, "Password login is disabled. Use OIDC sign in.")
		return
	}

	token, err := h.sessions.Login(req.Username, req.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		h.limiter.RecordFail(ip)
		postLocked, postRetryAfter := h.limiter.CheckLock(ip)
		if postLocked {
			h.writeLockout(ctx, postRetryAfter)
			return
		}
		writeError(ctx, fasthttp.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "login failed")
		return
	}

	h.limiter.RecordSuccess(ip)

	user, err := h.sessions.Validate(token)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "login failed")
		return
	}

	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	cookie.SetKey(sessionCookieName)
	cookie.SetValue(token)
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	cookie.SetExpire(time.Now().Add(7 * 24 * time.Hour))
	ctx.Response.Header.SetCookie(cookie)

	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"token": token,
		"user":  toUserDTO(user),
	})
}

func (h *Handlers) writeLockout(ctx *fasthttp.RequestCtx, retryAfter int) {
	msg := fmt.Sprintf("Too many failed attempts. Try again in %ds. %s", retryAfter, resetHint)
	b, _ := json.Marshal(map[string]any{
		"data": nil,
		"error": map[string]any{
			"message":      msg,
			"retry_after":  retryAfter,
			"reset_hint":   resetHint,
		},
	})
	ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
	ctx.SetContentType("application/json")
	ctx.SetBody(b)
	ctx.Response.Header.Set("Retry-After", strconv.Itoa(retryAfter))
}

// Logout handles POST /api/auth/logout.
func (h *Handlers) Logout(ctx *fasthttp.RequestCtx) {
	token := requestToken(ctx)
	if token != "" {
		if err := h.sessions.Logout(token); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "logout failed")
			return
		}
	}
	ctx.Response.Header.DelClientCookie(sessionCookieName)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"logged_out": true})
}

// Me handles GET /api/auth/me. Requires RequireSession.
func (h *Handlers) Me(ctx *fasthttp.RequestCtx) {
	user, ok := ctx.UserValue(userKey).(*store.User)
	if !ok {
		writeError(ctx, fasthttp.StatusUnauthorized, "unauthorized")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toUserDTO(user))
}

// Status handles GET /api/auth/status.
func (h *Handlers) Status(ctx *fasthttp.RequestCtx) {
	settings, err := h.store.GetSettings()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load settings")
		return
	}
	authMode := settings["auth_mode"]
	if authMode == "" {
		authMode = "password"
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"auth_mode": authMode,
	})
}

// RequireSession is middleware that rejects requests without a valid
// session token (Authorization: Bearer or the session cookie).
func (h *Handlers) RequireSession(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		user, err := h.sessions.Validate(requestToken(ctx))
		if err != nil {
			writeError(ctx, fasthttp.StatusUnauthorized, "unauthorized")
			return
		}
		ctx.SetUserValue(userKey, user)
		next(ctx)
	}
}

// requestToken extracts the session token from the Authorization header
// or the session cookie.
func requestToken(ctx *fasthttp.RequestCtx) string {
	header := string(ctx.Request.Header.Peek("Authorization"))
	if after, ok := strings.CutPrefix(header, "Bearer "); ok && after != "" {
		return after
	}
	return string(ctx.Request.Header.Cookie(sessionCookieName))
}

// isTunnelRequest reports whether the request Host matches the tunnelUrl or
// tailscaleUrl hostname.
func isTunnelRequest(ctx *fasthttp.RequestCtx, settings map[string]string) bool {
	host := hostName(string(ctx.Host()))
	tunnelHost := urlHostname(settings["tunnelUrl"])
	tailscaleHost := urlHostname(settings["tailscaleUrl"])
	return (tunnelHost != "" && host == tunnelHost) || (tailscaleHost != "" && host == tailscaleHost)
}

// hostName strips port and IPv6 brackets from a host string, matching the
// dashboardGuard.js :85-89 helper.
func hostName(host string) string {
	if host == "" {
		return ""
	}
	h := strings.Split(host, ":")[0]
	h = strings.TrimPrefix(h, "[")
	h = strings.TrimSuffix(h, "]")
	return strings.ToLower(h)
}

// urlHostname extracts the hostname from a URL setting value (e.g. https://host:port).
func urlHostname(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
