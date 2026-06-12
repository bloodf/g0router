package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ModelAlias is a persisted name-to-target model alias.
type ModelAlias struct {
	Name      string
	Target    string
	CreatedAt int64
}

// CreateAlias inserts or replaces a model alias record after verifying the
// write would not introduce a cycle in the alias graph.
func (s *Store) CreateAlias(name, target string) error {
	if name == "" {
		return errors.New("alias name must not be empty")
	}
	if target == "" {
		return errors.New("alias target must not be empty")
	}
	if name == target {
		return fmt.Errorf("alias %q would create a cycle (self-loop)", name)
	}

	// DFS cycle detection at write time: starting from target, follow the
	// existing chain. If we reach name, the new alias would close a cycle.
	seen := make(map[string]bool)
	cur := target
	for {
		if cur == name {
			return fmt.Errorf("alias %q -> %q would create a cycle", name, target)
		}
		if seen[cur] {
			break
		}
		seen[cur] = true
		var next string
		err := s.db.QueryRow("SELECT target FROM model_aliases WHERE name = ?", cur).Scan(&next)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return fmt.Errorf("cycle check for %q: %w", name, err)
		}
		cur = next
	}

	now := time.Now().Unix()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO model_aliases (name, target, created_at) VALUES (?, ?, ?)",
		name, target, now,
	)
	if err != nil {
		return fmt.Errorf("insert alias %s: %w", name, err)
	}
	return nil
}

// ListAliases returns all model aliases ordered by name.
func (s *Store) ListAliases() ([]*ModelAlias, error) {
	rows, err := s.db.Query("SELECT name, target, created_at FROM model_aliases ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("query aliases: %w", err)
	}
	defer rows.Close()

	var out []*ModelAlias
	for rows.Next() {
		a, err := scanAlias(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate aliases: %w", err)
	}
	return out, nil
}

// DeleteAlias removes the model alias with the given name.
func (s *Store) DeleteAlias(name string) error {
	res, err := s.db.Exec("DELETE FROM model_aliases WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("delete alias %s: %w", name, err)
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

// ResolveChain follows alias chains and returns the final name.
// If the chain ends at an unknown name, that name is returned unchanged.
func (s *Store) ResolveChain(name string) (string, error) {
	seen := make(map[string]bool)
	cur := name
	for {
		if seen[cur] {
			// Cycle in stored data (should not happen after valid writes). Return current.
			return cur, nil
		}
		seen[cur] = true
		var target string
		err := s.db.QueryRow(`SELECT target FROM model_aliases WHERE name = ?`, cur).Scan(&target)
		if errors.Is(err, sql.ErrNoRows) {
			return cur, nil
		}
		if err != nil {
			return "", fmt.Errorf("resolve chain %q: %w", name, err)
		}
		cur = target
	}
}

func scanAlias(row interface {
	Scan(dest ...any) error
}) (*ModelAlias, error) {
	var a ModelAlias
	err := row.Scan(&a.Name, &a.Target, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan alias: %w", err)
	}
	return &a, nil
}
