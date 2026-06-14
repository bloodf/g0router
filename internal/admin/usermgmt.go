package admin

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// userMgmtDTO is the user-management view of a user. It NEVER carries the
// password or its hash.
type userMgmtDTO struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func toUserMgmtDTO(u *store.User) userMgmtDTO {
	return userMgmtDTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
}

// AuthSetup handles POST /api/auth/setup. It bootstraps the first admin user
// and is reachable without a session, self-guarding on CountUsers()==0.
func (h *Handlers) AuthSetup(ctx *fasthttp.RequestCtx) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "username and password are required")
		return
	}

	n, err := h.store.CountUsers()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "count users")
		return
	}
	if n > 0 {
		writeError(ctx, fasthttp.StatusConflict, "setup already completed")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "hash password")
		return
	}
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}
	user, err := h.store.CreateUserFull(req.Username, hash, displayName, "admin")
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create user")
		return
	}

	// Auto-authenticate the new admin (mirrors Login).
	token, err := h.sessions.Login(req.Username, req.Password)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "establish session")
		return
	}
	setSessionCookie(ctx, token)

	h.recordAudit(ctx, "setup", user.Username, "Completed initial setup for "+user.Username)
	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"token": token,
		"user":  toUserMgmtDTO(user),
	})
}

// ChangePassword handles PUT /api/auth/password for the authenticated user.
func (h *Handlers) ChangePassword(ctx *fasthttp.RequestCtx) {
	user, ok := ctx.UserValue(userKey).(*store.User)
	if !ok || user == nil {
		writeError(ctx, fasthttp.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.NewPassword == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "new_password is required")
		return
	}
	if !currentPasswordValid(user.PasswordHash, req.CurrentPassword) {
		writeError(ctx, fasthttp.StatusBadRequest, "Current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "hash password")
		return
	}
	if err := h.store.UpdateUserPassword(user.ID, hash); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update password")
		return
	}
	h.recordAudit(ctx, "change_password", user.Username, "Changed password for "+user.Username)
	writeData(ctx, fasthttp.StatusOK, map[string]any{})
}

// ListUsers handles GET /api/auth/users. The hash is never returned.
func (h *Handlers) ListUsers(ctx *fasthttp.RequestCtx) {
	users, err := h.store.ListUsers()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list users")
		return
	}
	out := make([]userMgmtDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toUserMgmtDTO(u))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateUser handles POST /api/auth/users.
func (h *Handlers) CreateUser(ctx *fasthttp.RequestCtx) {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		Password    string `json:"password"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "username and password are required")
		return
	}

	if _, err := h.store.GetUserByUsername(req.Username); err == nil {
		writeError(ctx, fasthttp.StatusConflict, "username already exists")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusInternalServerError, "lookup user")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "hash password")
		return
	}
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}
	role := req.Role
	if role == "" {
		role = "user"
	}
	user, err := h.store.CreateUserFull(req.Username, hash, displayName, role)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create user")
		return
	}
	h.recordAudit(ctx, "create_user", user.Username, "Created user "+user.Username)
	writeData(ctx, fasthttp.StatusCreated, toUserMgmtDTO(user))
}

// DeleteUser handles DELETE /api/auth/users/{id}. It refuses to remove the
// last remaining user so the dashboard cannot lock itself out.
func (h *Handlers) DeleteUser(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}

	target, err := h.store.GetUserByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load user")
		return
	}

	n, err := h.store.CountUsers()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "count users")
		return
	}
	if n <= 1 {
		writeError(ctx, fasthttp.StatusBadRequest, "cannot delete the last user")
		return
	}

	if err := h.store.DeleteUser(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "user not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete user")
		return
	}
	h.recordAudit(ctx, "delete_user", target.Username, "Deleted user "+target.Username)
	writeData(ctx, fasthttp.StatusOK, map[string]any{})
}

// currentPasswordValid verifies the supplied current password against the
// stored hash, also accepting the default/INITIAL_PASSWORD path when no hash
// is set (mirrors Sessions.Login, session.go:61-73).
func currentPasswordValid(hash, current string) bool {
	if hash == "" {
		initial := os.Getenv("INITIAL_PASSWORD")
		if initial == "" {
			initial = "123456"
		}
		return current == initial
	}
	return auth.VerifyPassword(hash, current)
}

// setSessionCookie writes the session cookie, mirroring Login (auth.go:110-118).
func setSessionCookie(ctx *fasthttp.RequestCtx, token string) {
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	cookie.SetKey(sessionCookieName)
	cookie.SetValue(token)
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	cookie.SetExpire(time.Now().Add(7 * 24 * time.Hour))
	ctx.Response.Header.SetCookie(cookie)
}
