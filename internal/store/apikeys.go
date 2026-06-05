package store

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type APIKey struct {
	ID               string
	Name             string
	Prefix           string
	IsActive         bool
	LastUsedAt       *string
	CreatedAt        string
	ExpiresAt        *int64
	Scopes           []string
	RateLimitRPM     *int
	RateLimitTPM     *int
	DailySpendCapUSD *float64
}

// APIKeyPolicy carries the per-key enforcement settings applied to inference
// requests. Nil pointers mean "unset" (no limit / never expires); an empty
// Scopes slice means all models are allowed.
type APIKeyPolicy struct {
	ExpiresAt        *int64
	Scopes           []string
	RateLimitRPM     *int
	RateLimitTPM     *int
	DailySpendCapUSD *float64
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

// UpdateAPIKeyPolicy validates and persists the per-key policy. Non-negative
// rate limits and spend cap are enforced. The key must exist.
func (s *Store) UpdateAPIKeyPolicy(id string, policy APIKeyPolicy) error {
	if err := validateAPIKeyPolicy(policy); err != nil {
		return err
	}

	var scopes *string
	if len(policy.Scopes) > 0 {
		encoded, err := json.Marshal(policy.Scopes)
		if err != nil {
			return fmt.Errorf("encode scopes: %w", err)
		}
		value := string(encoded)
		scopes = &value
	}

	res, err := s.db.Exec(
		`UPDATE api_keys
		SET expires_at = ?, scopes = ?, rate_limit_rpm = ?, rate_limit_tpm = ?, daily_spend_cap_usd = ?
		WHERE id = ?`,
		policy.ExpiresAt,
		scopes,
		policy.RateLimitRPM,
		policy.RateLimitTPM,
		policy.DailySpendCapUSD,
		id,
	)
	if err != nil {
		return fmt.Errorf("update api key policy: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update api key policy rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update api key policy: key %q not found", id)
	}
	return nil
}

func validateAPIKeyPolicy(policy APIKeyPolicy) error {
	if policy.RateLimitRPM != nil && *policy.RateLimitRPM < 0 {
		return fmt.Errorf("rate_limit_rpm must be non-negative")
	}
	if policy.RateLimitTPM != nil && *policy.RateLimitTPM < 0 {
		return fmt.Errorf("rate_limit_tpm must be non-negative")
	}
	if policy.DailySpendCapUSD != nil && *policy.DailySpendCapUSD < 0 {
		return fmt.Errorf("daily_spend_cap_usd must be non-negative")
	}
	return nil
}

const apiKeySelectColumns = `id, name, prefix, is_active, last_used_at, created_at,
	expires_at, scopes, rate_limit_rpm, rate_limit_tpm, daily_spend_cap_usd`

func scanAPIKey(scan func(dest ...any) error) (*APIKey, error) {
	var key APIKey
	var (
		expiresAt sql.NullInt64
		scopes    sql.NullString
		rpm       sql.NullInt64
		tpm       sql.NullInt64
		cap       sql.NullFloat64
	)
	if err := scan(
		&key.ID, &key.Name, &key.Prefix, &key.IsActive, &key.LastUsedAt, &key.CreatedAt,
		&expiresAt, &scopes, &rpm, &tpm, &cap,
	); err != nil {
		return nil, err
	}
	key.ExpiresAt = int64PtrFromNull(expiresAt)
	if err := decodeJSON(scopes, &key.Scopes); err != nil {
		return nil, fmt.Errorf("decode scopes: %w", err)
	}
	if rpm.Valid {
		v := int(rpm.Int64)
		key.RateLimitRPM = &v
	}
	if tpm.Valid {
		v := int(tpm.Int64)
		key.RateLimitTPM = &v
	}
	if cap.Valid {
		v := cap.Float64
		key.DailySpendCapUSD = &v
	}
	return &key, nil
}

func (s *Store) GetAPIKey(id string) (*APIKey, error) {
	row := s.db.QueryRow(
		`SELECT `+apiKeySelectColumns+` FROM api_keys WHERE id = ?`,
		id,
	)
	key, err := scanAPIKey(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get api key: key %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return key, nil
}

func (s *Store) ValidateAPIKey(raw, secret string) (*APIKey, bool, error) {
	keyHash := hashAPIKey(raw, secret)

	row := s.db.QueryRow(
		`SELECT `+apiKeySelectColumns+`
		FROM api_keys
		WHERE key_hash = ? AND is_active = 1`,
		keyHash,
	)
	key, err := scanAPIKey(row.Scan)
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

	return key, true, nil
}

func (s *Store) ListAPIKeys() ([]APIKey, error) {
	rows, err := s.db.Query(
		`SELECT ` + apiKeySelectColumns + `
		FROM api_keys
		ORDER BY created_at, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, *key)
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
