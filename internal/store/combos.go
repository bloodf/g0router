package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrInvalidComboStrategy is returned when a combo carries an unknown strategy.
var ErrInvalidComboStrategy = errors.New("invalid combo strategy")

// Combo selection strategies. "fallback" is the default and preserves the
// historical behavior of trying steps in stored order.
const (
	ComboStrategyFallback   = "fallback"
	ComboStrategyRoundRobin = "round_robin"
	ComboStrategyLeastUsed  = "least_used"
	ComboStrategyAuto       = "auto"
)

type ComboStep struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Combo struct {
	ID        string
	Name      string
	Steps     []ComboStep
	Strategy  string `json:"strategy"`
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}

// NormalizeComboStrategy defaults an empty strategy to fallback and rejects
// unknown values with a wrapped ErrInvalidComboStrategy.
func NormalizeComboStrategy(strategy string) (string, error) {
	switch strategy {
	case "":
		return ComboStrategyFallback, nil
	case ComboStrategyFallback, ComboStrategyRoundRobin, ComboStrategyLeastUsed, ComboStrategyAuto:
		return strategy, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidComboStrategy, strategy)
	}
}

func (s *Store) CreateCombo(combo *Combo) error {
	strategy, err := NormalizeComboStrategy(combo.Strategy)
	if err != nil {
		return err
	}
	combo.Strategy = strategy

	steps, err := encodeComboSteps(combo.Steps)
	if err != nil {
		return fmt.Errorf("encode combo steps: %w", err)
	}

	row := s.db.QueryRow(
		`INSERT INTO combos (name, steps, strategy, is_active)
		VALUES (?, ?, ?, ?)
		RETURNING id, created_at, updated_at`,
		combo.Name,
		steps,
		combo.Strategy,
		boolToInt(combo.IsActive),
	)
	if err := row.Scan(&combo.ID, &combo.CreatedAt, &combo.UpdatedAt); err != nil {
		return fmt.Errorf("insert combo: %w", err)
	}

	return nil
}

func (s *Store) GetCombo(id string) (*Combo, error) {
	combo, err := scanCombo(s.db.QueryRow(comboSelectSQL()+" WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return combo, nil
}

func (s *Store) GetActiveCombo(name string) (*Combo, error) {
	combo, err := scanCombo(s.db.QueryRow(comboSelectSQL()+" WHERE name = ? AND is_active = 1", name))
	if err != nil {
		return nil, err
	}
	return combo, nil
}

func (s *Store) ListCombos() ([]*Combo, error) {
	rows, err := s.db.Query(comboSelectSQL() + " ORDER BY rowid")
	if err != nil {
		return nil, fmt.Errorf("query combos: %w", err)
	}
	defer rows.Close()

	var combos []*Combo
	for rows.Next() {
		combo, err := scanCombo(rows)
		if err != nil {
			return nil, err
		}
		combos = append(combos, combo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate combos: %w", err)
	}

	return combos, nil
}

func (s *Store) UpdateCombo(combo *Combo) error {
	strategy, err := NormalizeComboStrategy(combo.Strategy)
	if err != nil {
		return err
	}
	combo.Strategy = strategy

	steps, err := encodeComboSteps(combo.Steps)
	if err != nil {
		return fmt.Errorf("encode combo steps: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE combos SET
			name = ?,
			steps = ?,
			strategy = ?,
			is_active = ?,
			updated_at = datetime('now')
		WHERE id = ?`,
		combo.Name,
		steps,
		combo.Strategy,
		boolToInt(combo.IsActive),
		combo.ID,
	)
	if err != nil {
		return fmt.Errorf("update combo: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteCombo(id string) error {
	result, err := s.db.Exec("DELETE FROM combos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete combo: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}

	return nil
}

func scanCombo(scanner connectionScanner) (*Combo, error) {
	var combo Combo
	var steps string
	var strategy string
	var isActive int

	err := scanner.Scan(
		&combo.ID,
		&combo.Name,
		&steps,
		&strategy,
		&isActive,
		&combo.CreatedAt,
		&combo.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan combo: %w", err)
	}

	if err := json.Unmarshal([]byte(steps), &combo.Steps); err != nil {
		return nil, fmt.Errorf("decode combo steps: %w", err)
	}
	if strategy == "" {
		strategy = ComboStrategyFallback
	}
	combo.Strategy = strategy
	combo.IsActive = isActive != 0

	return &combo, nil
}

func comboSelectSQL() string {
	return `SELECT id, name, steps, strategy, is_active, created_at, updated_at FROM combos`
}

func encodeComboSteps(steps []ComboStep) (string, error) {
	data, err := json.Marshal(steps)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
