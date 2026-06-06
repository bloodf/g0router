package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type DashboardSession struct {
	TokenHash  string
	UserID     int64
	CreatedAt  string
	LastSeenAt string
	ExpiresAt  string
	UserAgent  string
	IP         string
}

func hashRawToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}

func (s *Store) CreateDashboardSession(userID int64, rawToken, userAgent, ip string, expiresAt time.Time) error {
	tokenHash := hashRawToken(rawToken)

	_, err := s.db.Exec(
		`INSERT INTO dashboard_sessions (token_hash, user_id, expires_at, user_agent, ip)
		VALUES (?, ?, ?, ?, ?)`,
		tokenHash,
		userID,
		expiresAt.UTC().Format("2006-01-02 15:04:05"),
		emptyStringNil(userAgent),
		emptyStringNil(ip),
	)
	if err != nil {
		return fmt.Errorf("create dashboard session: %w", err)
	}
	return nil
}

func scanDashboardSession(scan func(dest ...any) error) (*DashboardSession, error) {
	var session DashboardSession
	var userAgent, ip sql.NullString

	if err := scan(
		&session.TokenHash,
		&session.UserID,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&userAgent,
		&ip,
	); err != nil {
		return nil, err
	}

	session.UserAgent = stringValueFromNull(userAgent)
	session.IP = stringValueFromNull(ip)
	return &session, nil
}

func (s *Store) GetDashboardSessionByTokenHash(tokenHash string) (*DashboardSession, error) {
	row := s.db.QueryRow(
		`SELECT token_hash, user_id, created_at, last_seen_at, expires_at, user_agent, ip
		FROM dashboard_sessions
		WHERE token_hash = ?`,
		tokenHash,
	)
	session, err := scanDashboardSession(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get dashboard session by token hash: session %q not found", tokenHash)
	}
	if err != nil {
		return nil, fmt.Errorf("get dashboard session by token hash: %w", err)
	}
	return session, nil
}

func (s *Store) GetDashboardSessionByRawToken(rawToken string) (*DashboardSession, error) {
	tokenHash := hashRawToken(rawToken)
	return s.GetDashboardSessionByTokenHash(tokenHash)
}

func (s *Store) TouchDashboardSession(tokenHash string) error {
	_, err := s.db.Exec(
		`UPDATE dashboard_sessions
		SET last_seen_at = CURRENT_TIMESTAMP
		WHERE token_hash = ?
		AND datetime(last_seen_at) < datetime('now', '-1 minute')`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("touch dashboard session: %w", err)
	}
	return nil
}

func (s *Store) DeleteDashboardSession(tokenHash string) error {
	res, err := s.db.Exec("DELETE FROM dashboard_sessions WHERE token_hash = ?", tokenHash)
	if err != nil {
		return fmt.Errorf("delete dashboard session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete dashboard session rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete dashboard session: session %q not found", tokenHash)
	}
	return nil
}

func (s *Store) DeleteDashboardSessionsByUserID(userID int64) error {
	_, err := s.db.Exec("DELETE FROM dashboard_sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("delete dashboard sessions by user id: %w", err)
	}
	return nil
}

func (s *Store) PurgeExpiredDashboardSessions() error {
	_, err := s.db.Exec("DELETE FROM dashboard_sessions WHERE datetime(expires_at) < datetime('now')")
	if err != nil {
		return fmt.Errorf("purge expired dashboard sessions: %w", err)
	}
	return nil
}
