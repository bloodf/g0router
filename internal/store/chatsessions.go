package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrInvalidMessagesJSON = errors.New("invalid messages_json: must be valid JSON, ≤2MB, images ≤5MB each and ≤4 per message, mime must be image/png|jpeg|webp|gif")

var dataImageRegex = regexp.MustCompile(`^data:image/(png|jpeg|webp|gif);base64,([A-Za-z0-9+/=]+)$`)

// ChatSession represents a stored chat conversation.
type ChatSession struct {
	ID           string
	Title        string
	Model        string
	Provider     string
	MessagesJSON string
	CreatedAt    string
	UpdatedAt    string
}

func validateMessagesJSON(raw string) error {
	if len(raw) > 2*1024*1024 {
		return ErrInvalidMessagesJSON
	}

	var messages []map[string]any
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return ErrInvalidMessagesJSON
	}

	for _, msg := range messages {
		count, err := countImagesInValue(msg)
		if err != nil {
			return err
		}
		if count > 4 {
			return ErrInvalidMessagesJSON
		}
	}

	return nil
}

func countImagesInValue(v any) (int, error) {
	switch val := v.(type) {
	case string:
		if !strings.HasPrefix(val, "data:image/") {
			return 0, nil
		}
		m := dataImageRegex.FindStringSubmatch(val)
		if m == nil {
			return 0, ErrInvalidMessagesJSON
		}
		data, err := base64.StdEncoding.DecodeString(m[2])
		if err != nil {
			return 0, ErrInvalidMessagesJSON
		}
		if len(data) > 5*1024*1024 {
			return 0, ErrInvalidMessagesJSON
		}
		return 1, nil
	case map[string]any:
		count := 0
		for _, v := range val {
			c, err := countImagesInValue(v)
			if err != nil {
				return 0, err
			}
			count += c
		}
		return count, nil
	case []any:
		count := 0
		for _, v := range val {
			c, err := countImagesInValue(v)
			if err != nil {
				return 0, err
			}
			count += c
		}
		return count, nil
	default:
		return 0, nil
	}
}

// ListChatSessions returns all chat sessions without MessagesJSON populated.
func (s *Store) ListChatSessions() ([]ChatSession, error) {
	rows, err := s.db.Query(`SELECT id, title, model, provider, updated_at FROM chat_sessions ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("query chat sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var session ChatSession
		if err := rows.Scan(&session.ID, &session.Title, &session.Model, &session.Provider, &session.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan chat session: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat sessions: %w", err)
	}
	return sessions, nil
}

// GetChatSession returns a single chat session by ID including MessagesJSON.
func (s *Store) GetChatSession(id string) (*ChatSession, error) {
	var session ChatSession
	var messagesJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT id, title, model, provider, messages_json, created_at, updated_at FROM chat_sessions WHERE id = ?`,
		id,
	).Scan(&session.ID, &session.Title, &session.Model, &session.Provider, &messagesJSON, &session.CreatedAt, &session.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get chat session: %w", err)
	}
	session.MessagesJSON = stringValueFromNull(messagesJSON)
	return &session, nil
}

// CreateChatSession inserts a new chat session after validating messagesJSON.
func (s *Store) CreateChatSession(title, model, provider, messagesJSON string) (*ChatSession, error) {
	if err := validateMessagesJSON(messagesJSON); err != nil {
		return nil, err
	}

	var session ChatSession
	var messagesNull sql.NullString
	err := s.db.QueryRow(
		`INSERT INTO chat_sessions (title, model, provider, messages_json) VALUES (?, ?, ?, ?) RETURNING id, title, model, provider, messages_json, created_at, updated_at`,
		emptyStringNil(title),
		emptyStringNil(model),
		emptyStringNil(provider),
		emptyStringNil(messagesJSON),
	).Scan(&session.ID, &session.Title, &session.Model, &session.Provider, &messagesNull, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create chat session: %w", err)
	}
	session.MessagesJSON = stringValueFromNull(messagesNull)
	return &session, nil
}

// UpdateChatSession applies a partial update to a chat session. updated_at is always refreshed.
func (s *Store) UpdateChatSession(id string, title, messagesJSON *string) error {
	if messagesJSON != nil {
		if err := validateMessagesJSON(*messagesJSON); err != nil {
			return err
		}
	}

	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}

	if title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *title)
	}
	if messagesJSON != nil {
		setClauses = append(setClauses, "messages_json = ?")
		args = append(args, *messagesJSON)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE chat_sessions SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update chat session: %w", err)
	}
	return requireRowsAffected(result)
}

// DeleteChatSession removes a chat session by ID.
func (s *Store) DeleteChatSession(id string) error {
	result, err := s.db.Exec(`DELETE FROM chat_sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete chat session: %w", err)
	}
	return requireRowsAffected(result)
}
