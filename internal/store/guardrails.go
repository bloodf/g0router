package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// guardrailsRowID is the fixed sentinel id for the singleton guardrails config row.
const guardrailsRowID = 1

// Guardrails is the singleton guardrails configuration. The blocklist and PII
// types are policy data (not secrets) and are stored as JSON arrays.
type Guardrails struct {
	Enabled             bool
	Blocklist           []string
	PIIRedactionEnabled bool
	PIIRedactionTypes   []string
}

// GetGuardrails returns the singleton guardrails config. When no row exists yet
// it inserts a zero-value default row and returns it, so the config always exists.
func (s *Store) GetGuardrails() (*Guardrails, error) {
	g, err := s.scanGuardrails(s.db.QueryRow(
		`SELECT guardrails_enabled, guardrails_blocklist_json, pii_redaction_enabled, pii_redaction_types_json
		 FROM guardrails WHERE id = ?`, guardrailsRowID))
	if errors.Is(err, ErrNotFound) {
		def := &Guardrails{Blocklist: []string{}, PIIRedactionTypes: []string{}}
		if err := s.SetGuardrails(def); err != nil {
			return nil, err
		}
		return def, nil
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// SetGuardrails upserts the singleton guardrails config row.
func (s *Store) SetGuardrails(g *Guardrails) error {
	blocklistJSON, err := marshalStrings(g.Blocklist)
	if err != nil {
		return fmt.Errorf("marshal blocklist: %w", err)
	}
	typesJSON, err := marshalStrings(g.PIIRedactionTypes)
	if err != nil {
		return fmt.Errorf("marshal pii types: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO guardrails
		 (id, guardrails_enabled, guardrails_blocklist_json, pii_redaction_enabled, pii_redaction_types_json, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   guardrails_enabled = excluded.guardrails_enabled,
		   guardrails_blocklist_json = excluded.guardrails_blocklist_json,
		   pii_redaction_enabled = excluded.pii_redaction_enabled,
		   pii_redaction_types_json = excluded.pii_redaction_types_json,
		   updated_at = excluded.updated_at`,
		guardrailsRowID, boolToInt(g.Enabled), blocklistJSON, boolToInt(g.PIIRedactionEnabled), typesJSON, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("upsert guardrails: %w", err)
	}
	return nil
}

func (s *Store) scanGuardrails(row interface {
	Scan(dest ...any) error
}) (*Guardrails, error) {
	var g Guardrails
	var enabled, piiEnabled int
	var blocklistJSON, typesJSON string
	err := row.Scan(&enabled, &blocklistJSON, &piiEnabled, &typesJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan guardrails: %w", err)
	}
	g.Enabled = enabled != 0
	g.PIIRedactionEnabled = piiEnabled != 0
	if g.Blocklist, err = unmarshalStrings(blocklistJSON); err != nil {
		return nil, fmt.Errorf("unmarshal blocklist: %w", err)
	}
	if g.PIIRedactionTypes, err = unmarshalStrings(typesJSON); err != nil {
		return nil, fmt.Errorf("unmarshal pii types: %w", err)
	}
	return &g, nil
}

func marshalStrings(in []string) (string, error) {
	if in == nil {
		in = []string{}
	}
	b, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalStrings(raw string) ([]string, error) {
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
