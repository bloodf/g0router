package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// ErrInvalidCredentials is returned when login fails.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrUnauthorized is returned when a session token is missing, expired, or revoked.
var ErrUnauthorized = errors.New("auth: unauthorized")

// Sessions issues and validates dashboard session tokens backed by the store.
type Sessions struct {
	store *store.Store
	ttl   time.Duration
}

// NewSessions creates a session manager with the given token lifetime.
func NewSessions(st *store.Store, ttl time.Duration) *Sessions {
	return &Sessions{store: st, ttl: ttl}
}

// SeedAdmin creates the initial admin user if (and only if) no users exist.
// Returns true when a user was created.
func (s *Sessions) SeedAdmin(username, password string) (bool, error) {
	n, err := s.store.CountUsers()
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		return false, nil
	}
	hash, err := HashPassword(password)
	if err != nil {
		return false, fmt.Errorf("hash seed password: %w", err)
	}
	if _, err := s.store.CreateUser(username, hash); err != nil {
		return false, fmt.Errorf("create seed user: %w", err)
	}
	return true, nil
}

// Login verifies credentials and issues a new session token.
func (s *Sessions) Login(username, password string) (string, error) {
	user, err := s.store.GetUserByUsername(username)
	if errors.Is(err, store.ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("lookup user: %w", err)
	}
	if !VerifyPassword(user.PasswordHash, password) {
		return "", ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	expiresAt := time.Now().Add(s.ttl).Unix()
	if err := s.store.CreateSession(token, user.ID, expiresAt); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

// Validate returns the user owning the given session token.
func (s *Sessions) Validate(token string) (*store.User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}
	sess, err := s.store.GetSession(token)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("lookup session: %w", err)
	}
	user, err := s.store.GetUserByID(sess.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("lookup session user: %w", err)
	}
	return user, nil
}

// Logout revokes a session token.
func (s *Sessions) Logout(token string) error {
	if err := s.store.DeleteSession(token); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// randRead is a seam for injecting crypto/rand failures in tests.
var randRead = rand.Read

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
