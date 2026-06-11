package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User is a dashboard administrator account.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    int64
	UpdatedAt    int64
}

// CreateUser inserts a new user and returns it with its generated ID.
func (s *Store) CreateUser(username, passwordHash string) (*User, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err = s.db.Exec(
		"INSERT INTO users (id, username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		u.ID, u.Username, u.PasswordHash, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user %s: %w", username, err)
	}
	return u, nil
}

// GetUserByUsername returns the user with the given username.
func (s *Store) GetUserByUsername(username string) (*User, error) {
	return s.scanUser(s.db.QueryRow(
		"SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?", username))
}

// GetUserByID returns the user with the given ID.
func (s *Store) GetUserByID(id string) (*User, error) {
	return s.scanUser(s.db.QueryRow(
		"SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = ?", id))
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

// CountUsers returns the total number of users.
func (s *Store) CountUsers() (int, error) {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// FirstUser returns the first user in the table, ordered by creation time.
func (s *Store) FirstUser() (*User, error) {
	return s.scanUser(s.db.QueryRow(
		"SELECT id, username, password_hash, created_at, updated_at FROM users ORDER BY created_at ASC LIMIT 1"))
}

// UpdateUserPassword replaces the password hash for the given user ID.
func (s *Store) UpdateUserPassword(id, passwordHash string) error {
	res, err := s.db.Exec(
		"UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?",
		passwordHash, time.Now().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("update user password %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetUserPasswordHash updates the password hash for the given username.
func (s *Store) SetUserPasswordHash(username, passwordHash string) error {
	res, err := s.db.Exec(
		"UPDATE users SET password_hash = ?, updated_at = ? WHERE username = ?",
		passwordHash, time.Now().Unix(), username,
	)
	if err != nil {
		return fmt.Errorf("update user password hash %s: %w", username, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
