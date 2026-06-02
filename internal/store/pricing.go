package store

import (
	"database/sql"
	"fmt"
)

type PricingOverride struct {
	Provider           string
	Model              string
	InputCostPerToken  float64
	OutputCostPerToken float64
}

func (s *Store) SetPricingOverride(override PricingOverride) error {
	_, err := s.db.Exec(
		`INSERT INTO pricing_overrides (
			provider, model, input_cost_per_token, output_cost_per_token
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(provider, model) DO UPDATE SET
			input_cost_per_token = excluded.input_cost_per_token,
			output_cost_per_token = excluded.output_cost_per_token`,
		override.Provider,
		override.Model,
		override.InputCostPerToken,
		override.OutputCostPerToken,
	)
	if err != nil {
		return fmt.Errorf("upsert pricing override: %w", err)
	}
	return nil
}

func (s *Store) GetPricingOverride(provider, model string) (PricingOverride, error) {
	override, err := scanPricingOverride(s.db.QueryRow(
		`SELECT provider, model, input_cost_per_token, output_cost_per_token
		FROM pricing_overrides
		WHERE provider = ? AND model = ?`,
		provider,
		model,
	))
	if err != nil {
		return PricingOverride{}, err
	}
	return override, nil
}

func (s *Store) ListPricingOverrides() ([]PricingOverride, error) {
	rows, err := s.db.Query(
		`SELECT provider, model, input_cost_per_token, output_cost_per_token
		FROM pricing_overrides
		ORDER BY provider, model`,
	)
	if err != nil {
		return nil, fmt.Errorf("query pricing overrides: %w", err)
	}
	defer rows.Close()

	var overrides []PricingOverride
	for rows.Next() {
		override, err := scanPricingOverride(rows)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pricing overrides: %w", err)
	}

	return overrides, nil
}

func (s *Store) DeletePricingOverride(provider, model string) error {
	result, err := s.db.Exec(
		"DELETE FROM pricing_overrides WHERE provider = ? AND model = ?",
		provider,
		model,
	)
	if err != nil {
		return fmt.Errorf("delete pricing override: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func scanPricingOverride(scanner connectionScanner) (PricingOverride, error) {
	var override PricingOverride
	var inputCost sql.NullFloat64
	var outputCost sql.NullFloat64

	err := scanner.Scan(
		&override.Provider,
		&override.Model,
		&inputCost,
		&outputCost,
	)
	if err == sql.ErrNoRows {
		return PricingOverride{}, ErrNotFound
	}
	if err != nil {
		return PricingOverride{}, fmt.Errorf("scan pricing override: %w", err)
	}

	override.InputCostPerToken = inputCost.Float64
	override.OutputCostPerToken = outputCost.Float64

	return override, nil
}
