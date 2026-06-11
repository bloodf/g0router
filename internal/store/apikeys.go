package store

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// APIKey is a persisted dashboard API key.
type APIKey struct {
	ID        string
	Key       string
	Name      string
	MachineID string
	IsActive  bool
	CreatedAt int64
}

// apiKeyGenerator creates a new formatted API key and its machine identifier.
type apiKeyGenerator func(dataDir string) (key, machineID string, err error)

// defaultAPIKeySecret matches the secret used by internal/auth for CRCs.
const defaultAPIKeySecret = "endpoint-proxy-api-key-secret"

// SetAPIKeyGenerator replaces the key generator used by CreateAPIKey.
// Production code wires in auth.GenerateAPIKey + auth.MachineID to keep the
// store package free of the auth→store import cycle.
func (s *Store) SetAPIKeyGenerator(fn apiKeyGenerator) {
	s.apiKeyGenerator = fn
}

// defaultAPIKeyGenerator produces a valid key when no external generator is set.
// It is used by tests that do not need the exact auth.MachineID derivation.
func defaultAPIKeyGenerator(dataDir string) (string, string, error) {
	machineID := deriveMachineID(dataDir)
	keyID, err := generateAPIKeyID()
	if err != nil {
		return "", "", err
	}
	crc := computeAPIKeyCRC(machineID, keyID, defaultAPIKeySecret)
	return fmt.Sprintf("sk-%s-%s-%s", machineID, keyID, crc), machineID, nil
}

// deriveMachineID returns a stable 16-character hex identifier for the data dir.
func deriveMachineID(dataDir string) string {
	h := sha256.New()
	h.Write([]byte(dataDir))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// generateAPIKeyID returns a 6-character random identifier using lowercase
// letters and digits.
func generateAPIKeyID() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("generate key id: %w", err)
	}
	var sb strings.Builder
	for _, x := range b {
		if int(x) >= 252 {
			continue
		}
		sb.WriteByte(chars[int(x)%len(chars)])
		if sb.Len() == 6 {
			break
		}
	}
	if sb.Len() < 6 {
		return generateAPIKeyID()
	}
	return sb.String(), nil
}

// computeAPIKeyCRC returns the first 8 hex characters of an HMAC-SHA256 over
// machineID+keyID using the supplied secret.
func computeAPIKeyCRC(machineID, keyID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(machineID + keyID))
	return hex.EncodeToString(mac.Sum(nil))[:8]
}

// DataDir returns the directory containing the SQLite database, derived from
// the database file path. It is used by callers that need to read persisted
// machine-id and CLI-secret files.
func (s *Store) DataDir() string {
	var seq int
	var name, path string
	if err := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err == nil && path != "" {
		return filepath.Dir(path)
	}
	return ""
}

// CreateAPIKey inserts a new API key record. It generates the formatted key
// and the machine identifier internally; callers cannot supply raw keys.
func (s *Store) CreateAPIKey(name string) (*APIKey, error) {
	dataDir := s.DataDir()
	key, machineID, err := s.apiKeyGenerator(dataDir)
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}

	k := &APIKey{
		ID:        id,
		Key:       key,
		Name:      name,
		MachineID: machineID,
		IsActive:  true,
		CreatedAt: now,
	}
	_, err = s.db.Exec(
		"INSERT INTO api_keys (id, key, name, machine_id, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		k.ID, k.Key, k.Name, k.MachineID, boolToInt(k.IsActive), k.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert api key %s: %w", name, err)
	}
	return k, nil
}

// ListAPIKeys returns all API keys ordered by creation time (newest first).
func (s *Store) ListAPIKeys() ([]*APIKey, error) {
	rows, err := s.db.Query(
		"SELECT id, key, name, machine_id, is_active, created_at FROM api_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var out []*APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return out, nil
}

// GetAPIKeyByID returns the API key with the given id.
func (s *Store) GetAPIKeyByID(id string) (*APIKey, error) {
	return scanAPIKey(s.db.QueryRow(
		"SELECT id, key, name, machine_id, is_active, created_at FROM api_keys WHERE id = ?", id))
}

// GetAPIKeyByKey returns the API key with the exact key value.
func (s *Store) GetAPIKeyByKey(key string) (*APIKey, error) {
	return scanAPIKey(s.db.QueryRow(
		"SELECT id, key, name, machine_id, is_active, created_at FROM api_keys WHERE key = ?", key))
}

// SetAPIKeyActive updates the is_active flag for the given key id.
func (s *Store) SetAPIKeyActive(id string, active bool) error {
	res, err := s.db.Exec(
		"UPDATE api_keys SET is_active = ? WHERE id = ?",
		boolToInt(active), id,
	)
	if err != nil {
		return fmt.Errorf("update api key active %s: %w", id, err)
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

// DeleteAPIKey removes the API key with the given id.
func (s *Store) DeleteAPIKey(id string) error {
	res, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete api key %s: %w", id, err)
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

func scanAPIKey(row interface {
	Scan(dest ...any) error
}) (*APIKey, error) {
	var k APIKey
	var isActive int
	err := row.Scan(&k.ID, &k.Key, &k.Name, &k.MachineID, &isActive, &k.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan api key: %w", err)
	}
	k.IsActive = isActive != 0
	return &k, nil
}

