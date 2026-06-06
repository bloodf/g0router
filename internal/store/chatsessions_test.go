package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestChatSessionCreateListGetRoundTrip(t *testing.T) {
	s := openTestStore(t)

	msgs := `[{"role":"user","content":"hello"}]`
	session, err := s.CreateChatSession("Test Session", "gpt-4", "openai", msgs)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if session.Title != "Test Session" {
		t.Fatalf("title = %q, want Test Session", session.Title)
	}
	if session.Model != "gpt-4" {
		t.Fatalf("model = %q, want gpt-4", session.Model)
	}
	if session.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", session.Provider)
	}
	if session.MessagesJSON != msgs {
		t.Fatalf("messagesJSON = %q, want %q", session.MessagesJSON, msgs)
	}

	list, err := s.ListChatSessions()
	if err != nil {
		t.Fatalf("ListChatSessions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if list[0].ID != session.ID {
		t.Fatalf("id = %q, want %q", list[0].ID, session.ID)
	}
	if list[0].Title != "Test Session" {
		t.Fatalf("title = %q, want Test Session", list[0].Title)
	}
	if list[0].MessagesJSON != "" {
		t.Fatalf("ListChatSessions should not include MessagesJSON, got %q", list[0].MessagesJSON)
	}

	got, err := s.GetChatSession(session.ID)
	if err != nil {
		t.Fatalf("GetChatSession: %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("id = %q, want %q", got.ID, session.ID)
	}
	if got.MessagesJSON != msgs {
		t.Fatalf("messagesJSON = %q, want %q", got.MessagesJSON, msgs)
	}
}

func TestChatSessionListExcludesMessagesJSON(t *testing.T) {
	s := openTestStore(t)

	_, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"secret"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	list, err := s.ListChatSessions()
	if err != nil {
		t.Fatalf("ListChatSessions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if list[0].MessagesJSON != "" {
		t.Fatalf("MessagesJSON should be excluded from list, got %q", list[0].MessagesJSON)
	}
}

func TestChatSessionUpdateTitleOnly(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("Original", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	newTitle := "Updated"
	if err := s.UpdateChatSession(session.ID, &newTitle, nil); err != nil {
		t.Fatalf("UpdateChatSession: %v", err)
	}

	got, err := s.GetChatSession(session.ID)
	if err != nil {
		t.Fatalf("GetChatSession: %v", err)
	}
	if got.Title != "Updated" {
		t.Fatalf("title = %q, want Updated", got.Title)
	}
	if got.MessagesJSON != `[{"role":"user","content":"hi"}]` {
		t.Fatalf("messagesJSON = %q, want unchanged", got.MessagesJSON)
	}
}

func TestChatSessionUpdateMessagesJSONValid(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	newMsgs := `[{"role":"user","content":"updated"},{"role":"assistant","content":"ok"}]`
	if err := s.UpdateChatSession(session.ID, nil, &newMsgs); err != nil {
		t.Fatalf("UpdateChatSession: %v", err)
	}

	got, err := s.GetChatSession(session.ID)
	if err != nil {
		t.Fatalf("GetChatSession: %v", err)
	}
	if got.MessagesJSON != newMsgs {
		t.Fatalf("messagesJSON = %q, want %q", got.MessagesJSON, newMsgs)
	}
}

func TestChatSessionUpdateMessagesJSONInvalidJSON(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	badMsgs := `not json`
	if err := s.UpdateChatSession(session.ID, nil, &badMsgs); !errors.Is(err, ErrInvalidMessagesJSON) {
		t.Fatalf("expected ErrInvalidMessagesJSON, got %v", err)
	}
}

func TestChatSessionUpdateMessagesJSONOversized(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	largeContent := strings.Repeat("x", 2*1024*1024+100)
	msgs, _ := json.Marshal([]map[string]any{{"role": "user", "content": largeContent}})
	oversized := string(msgs)
	if err := s.UpdateChatSession(session.ID, nil, &oversized); !errors.Is(err, ErrInvalidMessagesJSON) {
		t.Fatalf("expected ErrInvalidMessagesJSON, got %v", err)
	}
}

func TestChatSessionUpdateMessagesJSONBadImageMime(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	badMsgs := `[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/bmp;base64,AAAA"}}]}]`
	if err := s.UpdateChatSession(session.ID, nil, &badMsgs); !errors.Is(err, ErrInvalidMessagesJSON) {
		t.Fatalf("expected ErrInvalidMessagesJSON, got %v", err)
	}
}

func TestChatSessionUpdateMessagesJSONImageTooLarge(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	bigData := make([]byte, 5*1024*1024+100)
	encoded := base64.StdEncoding.EncodeToString(bigData)
	imageURL := `data:image/png;base64,` + encoded
	msgs, _ := json.Marshal([]map[string]any{{"role": "user", "content": []map[string]any{{"type": "image_url", "image_url": map[string]any{"url": imageURL}}}}})
	badMsgs := string(msgs)

	if err := s.UpdateChatSession(session.ID, nil, &badMsgs); !errors.Is(err, ErrInvalidMessagesJSON) {
		t.Fatalf("expected ErrInvalidMessagesJSON, got %v", err)
	}
}

func TestChatSessionUpdateMessagesJSONTooManyImages(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	images := make([]map[string]any, 5)
	for i := 0; i < 5; i++ {
		images[i] = map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,iVBORw0KGgo="}}
	}
	msgs, _ := json.Marshal([]map[string]any{{"role": "user", "content": images}})
	badMsgs := string(msgs)

	if err := s.UpdateChatSession(session.ID, nil, &badMsgs); !errors.Is(err, ErrInvalidMessagesJSON) {
		t.Fatalf("expected ErrInvalidMessagesJSON, got %v", err)
	}
}

func TestChatSessionDeleteGetError(t *testing.T) {
	s := openTestStore(t)

	session, err := s.CreateChatSession("S1", "m1", "p1", `[{"role":"user","content":"hi"}]`)
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	if err := s.DeleteChatSession(session.ID); err != nil {
		t.Fatalf("DeleteChatSession: %v", err)
	}

	_, err = s.GetChatSession(session.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
