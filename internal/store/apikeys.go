package store

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type APIKey struct {
	ID         string
	Name       string
	Prefix     string
	IsActive   bool
	LastUsedAt *string
	CreatedAt  string
}

func (s *Store) CreateAPIKey(name, secret string) (*APIKey, string, error) {
	raw, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}

	keyHash := hashAPIKey(raw, secret)
	prefix := raw[:8]

	var key APIKey
	err = s.db.QueryRow(
		`INSERT INTO api_keys (name, key_hash, prefix)
		VALUES (?, ?, ?)
		RETURNING id, name, prefix, is_active, last_used_at, created_at`,
		name,
		keyHash,
		prefix,
	).Scan(&key.ID, &key.Name, &key.Prefix, &key.IsActive, &key.LastUsedAt, &key.CreatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("create api key: %w", err)
	}

	return &key, raw, nil
}

func (s *Store) ValidateAPIKey(raw, secret string) (*APIKey, bool, error) {
	keyHash := hashAPIKey(raw, secret)

	var key APIKey
	err := s.db.QueryRow(
		`SELECT id, name, prefix, is_active, last_used_at, created_at
		FROM api_keys
		WHERE key_hash = ? AND is_active = 1`,
		keyHash,
	).Scan(&key.ID, &key.Name, &key.Prefix, &key.IsActive, &key.LastUsedAt, &key.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("find api key: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", now, key.ID); err != nil {
		return nil, false, fmt.Errorf("update api key last used: %w", err)
	}
	key.LastUsedAt = &now

	return &key, true, nil
}

func (s *Store) ListAPIKeys() ([]APIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, name, prefix, is_active, last_used_at, created_at
		FROM api_keys
		ORDER BY created_at, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.Name, &key.Prefix, &key.IsActive, &key.LastUsedAt, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}

	return keys, nil
}

func (s *Store) DeleteAPIKey(id string) error {
	if _, err := s.db.Exec("UPDATE api_keys SET is_active = 0 WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	return nil
}

func generateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return "g0r_" + hex.EncodeToString(buf), nil
}

func hashAPIKey(raw, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}
