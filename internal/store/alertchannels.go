package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// AlertChannel is a configured alert destination. Config may carry secrets
// (webhook URLs / tokens) and is encrypted at rest in the config_enc column.
type AlertChannel struct {
	ID          int64
	Name        string
	ChannelType string
	Config      map[string]any
	Events      []string
	IsActive    bool
	CreatedAt   string // ISO-8601 (RFC3339)
}

// CreateAlertChannel inserts a channel, encrypting its config blob.
func (s *Store) CreateAlertChannel(in *AlertChannel) (*AlertChannel, error) {
	configEnc, err := s.encryptConfig(in.Config)
	if err != nil {
		return nil, err
	}
	eventsJSON, err := marshalStrings(in.Events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		`INSERT INTO alert_channels (name, channel_type, config_enc, events_json, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		in.Name, in.ChannelType, configEnc, eventsJSON, boolToInt(in.IsActive), now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert alert channel %s: %w", in.Name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return s.GetAlertChannelByID(id)
}

// ListAlertChannels returns all channels ordered by id ascending, config decrypted.
func (s *Store) ListAlertChannels() ([]*AlertChannel, error) {
	rows, err := s.db.Query(
		`SELECT id, name, channel_type, config_enc, events_json, is_active, created_at
		 FROM alert_channels ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query alert channels: %w", err)
	}
	defer rows.Close()

	var out []*AlertChannel
	for rows.Next() {
		c, err := s.scanAlertChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert channels: %w", err)
	}
	return out, nil
}

// GetAlertChannelByID returns the channel with the given id, config decrypted.
func (s *Store) GetAlertChannelByID(id int64) (*AlertChannel, error) {
	return s.scanAlertChannel(s.db.QueryRow(
		`SELECT id, name, channel_type, config_enc, events_json, is_active, created_at
		 FROM alert_channels WHERE id = ?`, id))
}

// UpdateAlertChannel updates mutable fields, re-encrypting the config blob.
func (s *Store) UpdateAlertChannel(id int64, in *AlertChannel) (*AlertChannel, error) {
	configEnc, err := s.encryptConfig(in.Config)
	if err != nil {
		return nil, err
	}
	eventsJSON, err := marshalStrings(in.Events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}
	res, err := s.db.Exec(
		`UPDATE alert_channels SET name = ?, channel_type = ?, config_enc = ?, events_json = ?, is_active = ?
		 WHERE id = ?`,
		in.Name, in.ChannelType, configEnc, eventsJSON, boolToInt(in.IsActive), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update alert channel %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.GetAlertChannelByID(id)
}

// DeleteAlertChannel removes the channel with the given id.
func (s *Store) DeleteAlertChannel(id int64) error {
	res, err := s.db.Exec("DELETE FROM alert_channels WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete alert channel %d: %w", id, err)
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

func (s *Store) encryptConfig(config map[string]any) (string, error) {
	if config == nil {
		config = map[string]any{}
	}
	b, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	enc, err := s.cipher.Encrypt(string(b))
	if err != nil {
		return "", fmt.Errorf("encrypt config: %w", err)
	}
	return enc, nil
}

func (s *Store) scanAlertChannel(row interface {
	Scan(dest ...any) error
}) (*AlertChannel, error) {
	var c AlertChannel
	var configEnc, eventsJSON string
	var isActive int
	err := row.Scan(&c.ID, &c.Name, &c.ChannelType, &configEnc, &eventsJSON, &isActive, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan alert channel: %w", err)
	}
	c.IsActive = isActive != 0

	plain, err := s.cipher.Decrypt(configEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}
	c.Config = map[string]any{}
	if plain != "" {
		if err := json.Unmarshal([]byte(plain), &c.Config); err != nil {
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}
	}
	if c.Events, err = unmarshalStrings(eventsJSON); err != nil {
		return nil, fmt.Errorf("unmarshal events: %w", err)
	}
	return &c, nil
}
