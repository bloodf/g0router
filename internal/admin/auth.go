package admin

import (
	"encoding/json"
	"errors"
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

	token, err := h.sessions.Login(req.Username, req.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(ctx, fasthttp.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "login failed")
		return
	}

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
