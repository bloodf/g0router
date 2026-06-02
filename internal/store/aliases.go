package store

import (
	"database/sql"
	"fmt"
)

type ModelAlias struct {
	Alias    string
	Provider string
	Model    string
}

func (s *Store) SetModelAlias(alias ModelAlias) error {
	_, err := s.db.Exec(
		`INSERT INTO model_aliases (alias, provider, model)
		VALUES (?, ?, ?)
		ON CONFLICT(alias) DO UPDATE SET
			provider = excluded.provider,
			model = excluded.model`,
		alias.Alias,
		alias.Provider,
		alias.Model,
	)
	if err != nil {
		return fmt.Errorf("upsert model alias: %w", err)
	}
	return nil
}

func (s *Store) ResolveModelAlias(alias string) (ModelAlias, error) {
	modelAlias, err := scanModelAlias(s.db.QueryRow(
		"SELECT alias, provider, model FROM model_aliases WHERE alias = ?",
		alias,
	))
	if err != nil {
		return ModelAlias{}, err
	}
	return modelAlias, nil
}

func (s *Store) ListModelAliases() ([]ModelAlias, error) {
	rows, err := s.db.Query("SELECT alias, provider, model FROM model_aliases ORDER BY alias")
	if err != nil {
		return nil, fmt.Errorf("query model aliases: %w", err)
	}
	defer rows.Close()

	var aliases []ModelAlias
	for rows.Next() {
		alias, err := scanModelAlias(rows)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model aliases: %w", err)
	}

	return aliases, nil
}

func (s *Store) DeleteModelAlias(alias string) error {
	result, err := s.db.Exec("DELETE FROM model_aliases WHERE alias = ?", alias)
	if err != nil {
		return fmt.Errorf("delete model alias: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func scanModelAlias(scanner connectionScanner) (ModelAlias, error) {
	var alias ModelAlias
	err := scanner.Scan(&alias.Alias, &alias.Provider, &alias.Model)
	if err == sql.ErrNoRows {
		return ModelAlias{}, ErrNotFound
	}
	if err != nil {
		return ModelAlias{}, fmt.Errorf("scan model alias: %w", err)
	}
	return alias, nil
}
