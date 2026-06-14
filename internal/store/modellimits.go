package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ModelLimit is an admin per-model rate/token limit. The id is an integer to
// mirror the numeric dashboard UI ModelLimit.id (ESC-IDTYPE). AllowedKeyIDs is
// persisted as a JSON blob in key_ids_json.
type ModelLimit struct {
	ID            int64
	Model         string
	MaxTokens     int
	MaxRPM        int
	AllowedKeyIDs []string
	CreatedAt     int64
	UpdatedAt     int64
}

func marshalKeyIDs(ids []string) (string, error) {
	if ids == nil {
		ids = []string{}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", fmt.Errorf("marshal allowed key ids: %w", err)
	}
	return string(b), nil
}

// CreateModelLimit inserts a new model limit and returns it with its
// autoincrement id assigned.
func (s *Store) CreateModelLimit(in *ModelLimit) (*ModelLimit, error) {
	now := time.Now().Unix()
	keyIDs, err := marshalKeyIDs(in.AllowedKeyIDs)
	if err != nil {
		return nil, err
	}
	res, err := s.db.Exec(
		"INSERT INTO model_limits (model, max_tokens, max_rpm, key_ids_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		in.Model, in.MaxTokens, in.MaxRPM, keyIDs, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert model limit %s: %w", in.Model, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("model limit last insert id: %w", err)
	}
	allowed := in.AllowedKeyIDs
	if allowed == nil {
		allowed = []string{}
	}
	return &ModelLimit{
		ID:            id,
		Model:         in.Model,
		MaxTokens:     in.MaxTokens,
		MaxRPM:        in.MaxRPM,
		AllowedKeyIDs: allowed,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// ListModelLimits returns all model limits ordered by creation time.
func (s *Store) ListModelLimits() ([]*ModelLimit, error) {
	rows, err := s.db.Query(
		"SELECT id, model, max_tokens, max_rpm, key_ids_json, created_at, updated_at FROM model_limits ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("query model limits: %w", err)
	}
	defer rows.Close()

	var out []*ModelLimit
	for rows.Next() {
		ml, err := scanModelLimit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ml)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model limits: %w", err)
	}
	return out, nil
}

// GetModelLimitByID returns the model limit with the given id.
func (s *Store) GetModelLimitByID(id int64) (*ModelLimit, error) {
	return scanModelLimit(s.db.QueryRow(
		"SELECT id, model, max_tokens, max_rpm, key_ids_json, created_at, updated_at FROM model_limits WHERE id = ?", id))
}

// UpdateModelLimit persists the mutable fields of the model limit.
func (s *Store) UpdateModelLimit(in *ModelLimit) error {
	keyIDs, err := marshalKeyIDs(in.AllowedKeyIDs)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(
		"UPDATE model_limits SET model = ?, max_tokens = ?, max_rpm = ?, key_ids_json = ?, updated_at = ? WHERE id = ?",
		in.Model, in.MaxTokens, in.MaxRPM, keyIDs, time.Now().Unix(), in.ID,
	)
	if err != nil {
		return fmt.Errorf("update model limit %d: %w", in.ID, err)
	}
	return requireRowAffected(res)
}

// DeleteModelLimit removes the model limit with the given id.
func (s *Store) DeleteModelLimit(id int64) error {
	res, err := s.db.Exec("DELETE FROM model_limits WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete model limit %d: %w", id, err)
	}
	return requireRowAffected(res)
}

func scanModelLimit(row rowScanner) (*ModelLimit, error) {
	var ml ModelLimit
	var keyIDsJSON string
	err := row.Scan(&ml.ID, &ml.Model, &ml.MaxTokens, &ml.MaxRPM, &keyIDsJSON, &ml.CreatedAt, &ml.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan model limit: %w", err)
	}
	if err := json.Unmarshal([]byte(keyIDsJSON), &ml.AllowedKeyIDs); err != nil {
		return nil, fmt.Errorf("unmarshal allowed key ids: %w", err)
	}
	if ml.AllowedKeyIDs == nil {
		ml.AllowedKeyIDs = []string{}
	}
	return &ml, nil
}
