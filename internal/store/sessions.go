package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Session is an authenticated dashboard session.
type Session struct {
	Token     string
	UserID    string
	ExpiresAt int64
	CreatedAt int64
}

// CreateSession stores a session token for a user.
func (s *Store) CreateSession(token, userID string, expiresAt int64) error {
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)",
		token, userID, expiresAt, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// GetSession returns the session for token if it exists and has not expired.
func (s *Store) GetSession(token string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(
		"SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = ?", token,
	).Scan(&sess.Token, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}
	if sess.ExpiresAt <= time.Now().Unix() {
		return nil, ErrNotFound
	}
	return &sess, nil
}

// DeleteSession removes a session token.
func (s *Store) DeleteSession(token string) error {
	if _, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes all sessions past their expiry.
func (s *Store) DeleteExpiredSessions() error {
	if _, err := s.db.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now().Unix()); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}
