package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
)

// VirtualKey is a persisted virtual key with provider routing rules and quotas.
type VirtualKey struct {
	schemas.VirtualKey
	Key       string
	IsActive  bool
	TeamID    string
	CreatedAt int64
	UpdatedAt int64
}

// virtualKeyConfig is the snake_case JSON blob stored in config_json.
type virtualKeyConfig struct {
	ProviderConfigs []schemas.ProviderConfig `json:"provider_configs"`
	Budget          *schemas.Budget          `json:"budget,omitempty"`
	RateLimitRPM    *int                     `json:"rate_limit_rpm,omitempty"`
	RateLimit       *schemas.RateLimit       `json:"rate_limit,omitempty"`
}

func marshalVirtualKeyConfig(vk *VirtualKey) (string, error) {
	cfg := virtualKeyConfig{
		ProviderConfigs: vk.ProviderConfigs,
		Budget:          vk.Budget,
		RateLimitRPM:    vk.RateLimitRPM,
		RateLimit:       vk.RateLimit,
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal virtual key config: %w", err)
	}
	return string(b), nil
}

func unmarshalVirtualKeyConfig(raw string, vk *VirtualKey) error {
	var cfg virtualKeyConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return fmt.Errorf("unmarshal virtual key config: %w", err)
	}
	vk.ProviderConfigs = cfg.ProviderConfigs
	vk.Budget = cfg.Budget
	vk.RateLimitRPM = cfg.RateLimitRPM
	vk.RateLimit = cfg.RateLimit
	return nil
}

// CreateVirtualKey inserts a new virtual key record and generates its unique key value.
func (s *Store) CreateVirtualKey(in *VirtualKey) (*VirtualKey, error) {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return nil, err
	}
	key, err := newID()
	if err != nil {
		return nil, fmt.Errorf("generate virtual key: %w", err)
	}

	cfgJSON, err := marshalVirtualKeyConfig(in)
	if err != nil {
		return nil, err
	}

	vk := &VirtualKey{
		VirtualKey: schemas.VirtualKey{
			ID:              id,
			Name:            in.Name,
			ProviderConfigs: in.ProviderConfigs,
			Budget:          in.Budget,
			RateLimitRPM:    in.RateLimitRPM,
		},
		Key:       "g0vk-" + key,
		IsActive:  true,
		TeamID:    in.TeamID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Store the VK value encrypted at rest: `key` holds sha256hex(raw) for
	// lookup, `key_enc` holds the reversible AES ciphertext (bf-gov-5). The
	// returned struct keeps .Key = raw (one-time reveal in the create response).
	keyEnc, err := s.cipher.Encrypt(vk.Key)
	if err != nil {
		return nil, fmt.Errorf("encrypt virtual key value: %w", err)
	}

	_, err = s.db.Exec(
		"INSERT INTO virtual_keys (id, key, key_enc, name, config_json, is_active, team_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		vk.ID, sha256hex(vk.Key), keyEnc, vk.Name, cfgJSON, boolToInt(vk.IsActive), vk.TeamID, vk.CreatedAt, vk.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert virtual key %s: %w", vk.Name, err)
	}
	return vk, nil
}

// ListVirtualKeys returns all virtual keys ordered by creation time (newest first).
func (s *Store) ListVirtualKeys() ([]*VirtualKey, error) {
	rows, err := s.db.Query(
		"SELECT id, key, key_enc, name, config_json, is_active, team_id, created_at, updated_at FROM virtual_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query virtual keys: %w", err)
	}
	defer rows.Close()

	var out []*VirtualKey
	for rows.Next() {
		vk, err := s.scanVirtualKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, vk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate virtual keys: %w", err)
	}
	return out, nil
}

// GetVirtualKeyByID returns the virtual key with the given id.
func (s *Store) GetVirtualKeyByID(id string) (*VirtualKey, error) {
	return s.scanVirtualKey(s.db.QueryRow(
		"SELECT id, key, key_enc, name, config_json, is_active, team_id, created_at, updated_at FROM virtual_keys WHERE id = ?", id))
}

// GetVirtualKeyByKey returns the virtual key with the given raw key value. The
// raw value is hashed before the lookup, since the `key` column stores
// sha256hex(raw), not the plaintext (bf-gov-5).
func (s *Store) GetVirtualKeyByKey(key string) (*VirtualKey, error) {
	return s.scanVirtualKey(s.db.QueryRow(
		"SELECT id, key, key_enc, name, config_json, is_active, team_id, created_at, updated_at FROM virtual_keys WHERE key = ?", sha256hex(key)))
}

// UpdateVirtualKey updates all mutable fields of the virtual key with the given id.
func (s *Store) UpdateVirtualKey(in *VirtualKey) error {
	now := time.Now().Unix()
	cfgJSON, err := marshalVirtualKeyConfig(in)
	if err != nil {
		return err
	}

	res, err := s.db.Exec(
		"UPDATE virtual_keys SET name = ?, config_json = ?, is_active = ?, team_id = ?, updated_at = ? WHERE id = ?",
		in.Name, cfgJSON, boolToInt(in.IsActive), in.TeamID, now, in.ID,
	)
	if err != nil {
		return fmt.Errorf("update virtual key %s: %w", in.ID, err)
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

// DeleteVirtualKey removes the virtual key with the given id.
func (s *Store) DeleteVirtualKey(id string) error {
	res, err := s.db.Exec("DELETE FROM virtual_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete virtual key %s: %w", id, err)
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

// backfillVirtualKeyEncryption migrates legacy rows (raw plaintext in `key`,
// empty `key_enc`) to the at-rest scheme: key_enc = Encrypt(raw), key =
// sha256hex(raw). Idempotent via the key_enc='' guard; invoked from Open(). See
// backfillEnc for the shared collect-then-update mechanics.
func (s *Store) backfillVirtualKeyEncryption() error {
	return s.backfillEnc("virtual_keys", "key", "key_enc", func(raw string) (string, string, error) {
		enc, err := s.cipher.Encrypt(raw)
		return sha256hex(raw), enc, err // key column repurposed as lookup hash
	})
}

// scanVirtualKey is a method so it can decrypt key_enc with s.cipher. The `key`
// column (sha256hex lookup hash) is scanned into a throwaway local and never
// surfaced; .Key is always the decrypted plaintext raw value (bf-gov-5).
func (s *Store) scanVirtualKey(row interface {
	Scan(dest ...any) error
}) (*VirtualKey, error) {
	var vk VirtualKey
	var isActive int
	var cfgJSON string
	var keyHash, keyEnc string
	err := row.Scan(&vk.ID, &keyHash, &keyEnc, &vk.Name, &cfgJSON, &isActive, &vk.TeamID, &vk.CreatedAt, &vk.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan virtual key: %w", err)
	}
	// Fail-closed: a non-empty key_enc that fails to decrypt (wrong secret /
	// corruption) makes the VK unreadable rather than silently succeeding,
	// matching every other *_enc column (oauthsessions, connections).
	if vk.Key, err = s.cipher.Decrypt(keyEnc); err != nil {
		return nil, fmt.Errorf("decrypt virtual key value: %w", err)
	}
	vk.IsActive = isActive != 0
	if err := unmarshalVirtualKeyConfig(cfgJSON, &vk); err != nil {
		return nil, err
	}
	return &vk, nil
}
