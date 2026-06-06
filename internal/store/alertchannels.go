package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// AlertChannel represents a notification endpoint.
type AlertChannel struct {
	ID          int64
	Name        string
	ChannelType string
	Config      string
	Events      []string
	IsActive    bool
	CreatedAt   string
}

// ListAlertChannels returns all alert channels with decrypted configs.
func (s *Store) ListAlertChannels() ([]AlertChannel, error) {
	rows, err := s.db.Query(`SELECT id, name, channel_type, config_enc, events_json, is_active, created_at FROM alert_channels ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list alert channels: %w", err)
	}
	defer rows.Close()

	var channels []AlertChannel
	for rows.Next() {
		ch, err := s.scanAlertChannel(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert channels: %w", err)
	}
	return channels, nil
}

// GetAlertChannel returns a single alert channel by id with decrypted config.
func (s *Store) GetAlertChannel(id int64) (*AlertChannel, error) {
	ch, err := s.scanAlertChannel(s.db.QueryRow(
		`SELECT id, name, channel_type, config_enc, events_json, is_active, created_at FROM alert_channels WHERE id = ?`,
		id,
	))
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// CreateAlertChannel inserts a new alert channel, encrypting the Config field.
func (s *Store) CreateAlertChannel(name, channelType, config string, events []string, isActive bool) (*AlertChannel, error) {
	configEnc, err := s.encryptString(config)
	if err != nil {
		return nil, fmt.Errorf("encrypt config: %w", err)
	}

	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}

	res, err := s.db.Exec(
		`INSERT INTO alert_channels (name, channel_type, config_enc, events_json, is_active) VALUES (?, ?, ?, ?, ?)`,
		name, channelType, emptyStringNil(configEnc), string(eventsJSON), boolInt(isActive),
	)
	if err != nil {
		return nil, fmt.Errorf("create alert channel: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create alert channel id: %w", err)
	}
	return s.GetAlertChannel(id)
}

// UpdateAlertChannel updates an existing alert channel. If config is non-empty it
// is encrypted and stored; otherwise the existing config_enc is preserved.
func (s *Store) UpdateAlertChannel(id int64, name, channelType, config string, events []string, isActive bool) error {
	var configEnc any
	if config != "" {
		enc, err := s.encryptString(config)
		if err != nil {
			return fmt.Errorf("encrypt config: %w", err)
		}
		configEnc = enc
	} else {
		configEnc = nil
	}

	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE alert_channels SET name = ?, channel_type = ?, config_enc = COALESCE(?, config_enc), events_json = ?, is_active = ? WHERE id = ?`,
		name, channelType, configEnc, string(eventsJSON), boolInt(isActive), id,
	)
	if err != nil {
		return fmt.Errorf("update alert channel: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

// DeleteAlertChannel removes an alert channel by id.
func (s *Store) DeleteAlertChannel(id int64) error {
	result, err := s.db.Exec(`DELETE FROM alert_channels WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete alert channel: %w", err)
	}
	if err := requireRowsAffected(result); err != nil {
		return err
	}
	return nil
}

func (s *Store) scanAlertChannel(scanner interface{ Scan(dest ...any) error }) (AlertChannel, error) {
	var ch AlertChannel
	var configEnc sql.NullString
	var eventsRaw string
	var isActive int

	err := scanner.Scan(&ch.ID, &ch.Name, &ch.ChannelType, &configEnc, &eventsRaw, &isActive, &ch.CreatedAt)
	if err == sql.ErrNoRows {
		return AlertChannel{}, ErrNotFound
	}
	if err != nil {
		return AlertChannel{}, fmt.Errorf("scan alert channel: %w", err)
	}

	ch.IsActive = isActive == 1
	_ = json.Unmarshal([]byte(eventsRaw), &ch.Events)

	if configEnc.Valid && configEnc.String != "" {
		decrypted, err := s.decryptString(configEnc.String)
		if err != nil {
			return AlertChannel{}, fmt.Errorf("decrypt config: %w", err)
		}
		ch.Config = decrypted
	}

	return ch, nil
}
