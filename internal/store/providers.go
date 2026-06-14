package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ProviderRecord is a configured upstream LLM provider. Prefix and APIType are
// populated for provider nodes (w7-platnodes); plain providers keep them empty.
type ProviderRecord struct {
	ID        string
	Name      string
	Type      string
	BaseURL   string
	Enabled   bool
	Prefix    string
	APIType   string
	CreatedAt int64
	UpdatedAt int64
}

// CreateProvider inserts a provider, assigning its ID and timestamps.
func (s *Store) CreateProvider(p *ProviderRecord) error {
	now := time.Now().Unix()
	id, err := newID()
	if err != nil {
		return err
	}
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err = s.db.Exec(
		"INSERT INTO providers (id, name, type, base_url, enabled, prefix, api_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Type, p.BaseURL, boolToInt(p.Enabled), p.Prefix, p.APIType, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert provider %s: %w", p.Name, err)
	}
	return nil
}

// ListProviders returns all providers ordered by creation time.
func (s *Store) ListProviders() ([]*ProviderRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, name, type, base_url, enabled, prefix, api_type, created_at, updated_at FROM providers ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	var out []*ProviderRecord
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return out, nil
}

// GetProvider returns the provider with the given ID.
func (s *Store) GetProvider(id string) (*ProviderRecord, error) {
	row := s.db.QueryRow(
		"SELECT id, name, type, base_url, enabled, prefix, api_type, created_at, updated_at FROM providers WHERE id = ?", id)
	p, err := scanProvider(row)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateProvider persists name, type, base URL, and enabled state.
func (s *Store) UpdateProvider(p *ProviderRecord) error {
	res, err := s.db.Exec(
		"UPDATE providers SET name = ?, type = ?, base_url = ?, enabled = ?, prefix = ?, api_type = ?, updated_at = ? WHERE id = ?",
		p.Name, p.Type, p.BaseURL, boolToInt(p.Enabled), p.Prefix, p.APIType, time.Now().Unix(), p.ID,
	)
	if err != nil {
		return fmt.Errorf("update provider %s: %w", p.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteProvider removes the provider with the given ID.
func (s *Store) DeleteProvider(id string) error {
	res, err := s.db.Exec("DELETE FROM providers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete provider %s: %w", id, err)
	}
	return requireRowAffected(res)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProvider(row rowScanner) (*ProviderRecord, error) {
	var p ProviderRecord
	var enabled int
	err := row.Scan(&p.ID, &p.Name, &p.Type, &p.BaseURL, &enabled, &p.Prefix, &p.APIType, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan provider: %w", err)
	}
	p.Enabled = enabled != 0
	return &p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func requireRowAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
