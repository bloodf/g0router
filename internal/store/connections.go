package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type AuthType string

const (
	AuthTypeOAuth  AuthType = "oauth"
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeNoAuth AuthType = "noauth"
)

type Connection struct {
	ID                   string
	Provider             string
	Name                 string
	AuthType             AuthType
	AccessToken          *string
	RefreshToken         *string
	ExpiresAt            *int64
	APIKey               *string
	IsActive             bool
	ProviderSpecificData map[string]any
	AccountID            *string
	Email                *string
	UnavailableUntil     *int64
	BackoffLevel         int
	ModelLocks           map[string]int64
	NeedsReauth          bool
	LastRefreshError     *string
	CreatedAt            string
	UpdatedAt            string
}

func (s *Store) CreateConnection(conn *Connection) error {
	providerData, err := encodeJSON(conn.ProviderSpecificData)
	if err != nil {
		return fmt.Errorf("encode provider data: %w", err)
	}
	modelLocks, err := encodeJSON(conn.ModelLocks)
	if err != nil {
		return fmt.Errorf("encode model locks: %w", err)
	}

	row := s.db.QueryRow(
		`INSERT INTO connections (
			provider, name, auth_type, access_token, refresh_token, expires_at,
			api_key, is_active, provider_specific_data, account_id, email,
			unavailable_until, backoff_level, model_locks,
			needs_reauth, last_refresh_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at`,
		conn.Provider,
		conn.Name,
		string(conn.AuthType),
		conn.AccessToken,
		conn.RefreshToken,
		conn.ExpiresAt,
		conn.APIKey,
		boolToInt(conn.IsActive),
		providerData,
		conn.AccountID,
		conn.Email,
		conn.UnavailableUntil,
		conn.BackoffLevel,
		modelLocks,
		boolToInt(conn.NeedsReauth),
		conn.LastRefreshError,
	)
	if err := row.Scan(&conn.ID, &conn.CreatedAt, &conn.UpdatedAt); err != nil {
		return fmt.Errorf("insert connection: %w", err)
	}

	return nil
}

func (s *Store) GetConnection(id string) (*Connection, error) {
	conn, err := scanConnection(s.db.QueryRow(connectionSelectSQL()+" WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *Store) GetConnections(provider string) ([]*Connection, error) {
	return s.queryConnections(connectionSelectSQL()+" WHERE provider = ? ORDER BY created_at, id", provider)
}

func (s *Store) ListConnections() ([]*Connection, error) {
	return s.queryConnections(connectionSelectSQL() + " ORDER BY provider, created_at, id")
}

func (s *Store) GetActiveConnections(provider string) ([]*Connection, error) {
	return s.queryConnections(connectionSelectSQL()+" WHERE provider = ? AND is_active = 1 ORDER BY created_at, id", provider)
}

func (s *Store) UpdateConnection(conn *Connection) error {
	providerData, err := encodeJSON(conn.ProviderSpecificData)
	if err != nil {
		return fmt.Errorf("encode provider data: %w", err)
	}
	modelLocks, err := encodeJSON(conn.ModelLocks)
	if err != nil {
		return fmt.Errorf("encode model locks: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE connections SET
			provider = ?,
			name = ?,
			auth_type = ?,
			access_token = ?,
			refresh_token = ?,
			expires_at = ?,
			api_key = ?,
			is_active = ?,
			provider_specific_data = ?,
			account_id = ?,
			email = ?,
			unavailable_until = ?,
			backoff_level = ?,
			model_locks = ?,
			needs_reauth = ?,
			last_refresh_error = ?,
			updated_at = datetime('now')
		WHERE id = ?`,
		conn.Provider,
		conn.Name,
		string(conn.AuthType),
		conn.AccessToken,
		conn.RefreshToken,
		conn.ExpiresAt,
		conn.APIKey,
		boolToInt(conn.IsActive),
		providerData,
		conn.AccountID,
		conn.Email,
		conn.UnavailableUntil,
		conn.BackoffLevel,
		modelLocks,
		boolToInt(conn.NeedsReauth),
		conn.LastRefreshError,
		conn.ID,
	)
	if err != nil {
		return fmt.Errorf("update connection: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateConnectionCredentials(id string, accessToken, refreshToken *string, expiresAt *int64) error {
	result, err := s.db.Exec(
		`UPDATE connections SET
			access_token = ?,
			refresh_token = ?,
			expires_at = ?,
			updated_at = datetime('now')
		WHERE id = ?`,
		accessToken,
		refreshToken,
		expiresAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("update connection credentials: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteConnection(id string) error {
	result, err := s.db.Exec("DELETE FROM connections WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) queryConnections(query string, args ...any) ([]*Connection, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var conns []*Connection
	for rows.Next() {
		conn, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connections: %w", err)
	}

	return conns, nil
}

type connectionScanner interface {
	Scan(dest ...any) error
}

func scanConnection(scanner connectionScanner) (*Connection, error) {
	var conn Connection
	var accessToken sql.NullString
	var refreshToken sql.NullString
	var expiresAt sql.NullInt64
	var apiKey sql.NullString
	var isActive int
	var providerData sql.NullString
	var accountID sql.NullString
	var email sql.NullString
	var unavailableUntil sql.NullInt64
	var modelLocks sql.NullString
	var needsReauth int
	var lastRefreshError sql.NullString

	err := scanner.Scan(
		&conn.ID,
		&conn.Provider,
		&conn.Name,
		&conn.AuthType,
		&accessToken,
		&refreshToken,
		&expiresAt,
		&apiKey,
		&isActive,
		&providerData,
		&accountID,
		&email,
		&unavailableUntil,
		&conn.BackoffLevel,
		&modelLocks,
		&needsReauth,
		&lastRefreshError,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan connection: %w", err)
	}

	conn.AccessToken = stringPtrFromNull(accessToken)
	conn.RefreshToken = stringPtrFromNull(refreshToken)
	conn.ExpiresAt = int64PtrFromNull(expiresAt)
	conn.APIKey = stringPtrFromNull(apiKey)
	conn.IsActive = isActive != 0
	conn.AccountID = stringPtrFromNull(accountID)
	conn.Email = stringPtrFromNull(email)
	conn.UnavailableUntil = int64PtrFromNull(unavailableUntil)
	conn.NeedsReauth = needsReauth != 0
	conn.LastRefreshError = stringPtrFromNull(lastRefreshError)

	if err := decodeJSON(providerData, &conn.ProviderSpecificData); err != nil {
		return nil, fmt.Errorf("decode provider data: %w", err)
	}
	if err := decodeJSON(modelLocks, &conn.ModelLocks); err != nil {
		return nil, fmt.Errorf("decode model locks: %w", err)
	}

	return &conn, nil
}

func connectionSelectSQL() string {
	return `SELECT
		id, provider, name, auth_type, access_token, refresh_token, expires_at,
		api_key, is_active, provider_specific_data, account_id, email,
		unavailable_until, backoff_level, model_locks,
		needs_reauth, last_refresh_error,
		created_at, updated_at
		FROM connections`
}

const maxRefreshErrorLen = 200

// sanitizeRefreshError truncates reason to maxRefreshErrorLen chars. The
// caller is responsible for passing a non-secret string; this is a last-resort
// defence.
func sanitizeRefreshError(reason string) string {
	r := []rune(reason)
	if len(r) > maxRefreshErrorLen {
		return string(r[:maxRefreshErrorLen])
	}
	return reason
}

// MarkConnectionRefreshFailure sets needs_reauth=true and records a sanitized
// error reason for the given connection.
func (s *Store) MarkConnectionRefreshFailure(id string, reason string) error {
	safe := sanitizeRefreshError(reason)
	result, err := s.db.Exec(
		`UPDATE connections SET
			needs_reauth = 1,
			last_refresh_error = ?,
			updated_at = datetime('now')
		WHERE id = ?`,
		safe,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark connection refresh failure: %w", err)
	}
	return requireRowsAffected(result)
}

// ClearConnectionRefreshFailure clears the needs_reauth flag and error for
// the given connection (e.g. after a successful refresh).
func (s *Store) ClearConnectionRefreshFailure(id string) error {
	result, err := s.db.Exec(
		`UPDATE connections SET
			needs_reauth = 0,
			last_refresh_error = NULL,
			updated_at = datetime('now')
		WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("clear connection refresh failure: %w", err)
	}
	return requireRowsAffected(result)
}

func requireRowsAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func encodeJSON(value any) (*string, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	encoded := string(data)
	return &encoded, nil
}

func decodeJSON[T any](value sql.NullString, dest *T) error {
	if !value.Valid || value.String == "" {
		return nil
	}
	return json.Unmarshal([]byte(value.String), dest)
}

func stringPtrFromNull(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func int64PtrFromNull(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
