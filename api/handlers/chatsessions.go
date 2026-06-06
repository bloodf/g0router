package handlers

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type chatSessionListItem struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Model     string `json:"model"`
	Provider  string `json:"provider"`
	UpdatedAt string `json:"updated_at"`
}

type chatSessionResponse struct {
	ID        string          `json:"id"`
	Title     string          `json:"title,omitempty"`
	Model     string          `json:"model"`
	Provider  string          `json:"provider"`
	Messages  json.RawMessage `json:"messages"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

func newChatSessionResponse(session store.ChatSession) chatSessionResponse {
	return chatSessionResponse{
		ID:        session.ID,
		Title:     session.Title,
		Model:     session.Model,
		Provider:  session.Provider,
		Messages:  json.RawMessage(session.MessagesJSON),
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

type createChatSessionRequest struct {
	Title    string `json:"title"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

type updateChatSessionRequest struct {
	Title    *string         `json:"title"`
	Messages json.RawMessage `json:"messages"`
}

type chatSessionStore interface {
	ListChatSessions() ([]store.ChatSession, error)
	GetChatSession(id string) (*store.ChatSession, error)
	CreateChatSession(title, model, provider, messagesJSON string) (*store.ChatSession, error)
	UpdateChatSession(id string, title, messagesJSON *string) error
	DeleteChatSession(id string) error
}

// ChatSessionList returns all chat sessions.
func ChatSessionList(ctx *fasthttp.RequestCtx, s chatSessionStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	sessions, err := s.ListChatSessions()
	if err != nil {
		log.Printf("list chat sessions: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list chat sessions")
		return
	}
	items := make([]chatSessionListItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, chatSessionListItem{
			ID:        session.ID,
			Title:     session.Title,
			Model:     session.Model,
			Provider:  session.Provider,
			UpdatedAt: session.UpdatedAt,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[chatSessionListItem]{Data: items})
}

// ChatSessionGet returns a single chat session by id.
func ChatSessionGet(ctx *fasthttp.RequestCtx, s chatSessionStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "chat session id required")
		return
	}
	session, err := s.GetChatSession(id)
	if err != nil {
		writeStoreError(ctx, "get chat session", err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": newChatSessionResponse(*session)})
}

// ChatSessionCreate creates a new chat session.
func ChatSessionCreate(ctx *fasthttp.RequestCtx, s chatSessionStore, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	var req createChatSessionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "model required")
		return
	}
	if req.Provider == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider required")
		return
	}
	session, err := s.CreateChatSession(req.Title, req.Model, req.Provider, "[]")
	if err != nil {
		log.Printf("create chat session: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create chat session")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "chat_session.create",
		Target: session.ID,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusCreated, map[string]any{"data": newChatSessionResponse(*session)})
}

// ChatSessionUpdate updates an existing chat session.
func ChatSessionUpdate(ctx *fasthttp.RequestCtx, s chatSessionStore, audit auditWriter, id string) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "chat session id required")
		return
	}
	var req updateChatSessionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	var titlePtr, messagesPtr *string
	if req.Title != nil {
		titlePtr = req.Title
	}
	if len(req.Messages) > 0 && string(req.Messages) != "null" {
		s := string(req.Messages)
		messagesPtr = &s
	}
	if titlePtr == nil && messagesPtr == nil {
		writeError(ctx, fasthttp.StatusBadRequest, "nothing to update")
		return
	}
	if err := s.UpdateChatSession(id, titlePtr, messagesPtr); err != nil {
		if errors.Is(err, store.ErrInvalidMessagesJSON) {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		writeStoreError(ctx, "update chat session", err)
		return
	}
	session, err := s.GetChatSession(id)
	if err != nil {
		writeStoreError(ctx, "get chat session", err)
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "chat_session.update",
		Target: id,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": newChatSessionResponse(*session)})
}

// ChatSessionDelete deletes a chat session.
func ChatSessionDelete(ctx *fasthttp.RequestCtx, s chatSessionStore, audit auditWriter, id string) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "chat session id required")
		return
	}
	if err := s.DeleteChatSession(id); err != nil {
		writeStoreError(ctx, "delete chat session", err)
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "chat_session.delete",
		Target: id,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}
