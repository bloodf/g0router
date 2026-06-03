# Phase 1: Core Types + SQLite Store

> **Depends on**: Phase 0  
> **Unlocks**: Phase 2, Phase 4, Phase 5, Phase 7, Phase 8  
> **Checkpoint**: `PHASE_1_COMPLETE`  
> **Key dependency**: `modernc.org/sqlite` (pure-Go, no CGO)

---

## Prerequisites

- [x] Phase 0 complete (`PHASE_0_COMPLETE`)
- [x] `go build ./cmd/g0router` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| SQLite library | `modernc.org/sqlite` | Pure Go, no CGO, cross-platform |
| ID generation | `lower(hex(randomblob(16)))` in SQL | 32-char hex, DB-native, no import |
| Time storage | RFC3339 strings | Human-readable, sortable, `time.RFC3339` |
| Nullable fields | Go `*T` pointers | Natural `omitempty`, distinguishes zero/absent |
| Auth types | `oauth`, `api_key`, `noauth` | Covers all provider auth patterns |
| API key hash | HMAC-SHA256 | Validate without storing plaintext |
| Settings | Key-value table | No migration needed for new settings |

---

## Task 1.1: Define Core Types

### TODO

- [x] Write `internal/providers/types_test.go` — **test file FIRST**
- [x] Run tests → see RED (types don't exist)
- [x] Write `internal/providers/types.go` — all types
- [x] Write `internal/providers/interface.go` — Provider interface
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-1/task-1: define core types with OpenAI-compatible JSON`

### Pre-conditions

- `internal/providers/` directory does not exist
- No dependencies beyond stdlib

### TDD Cycle

#### RED: Write Failing Tests First

Create `internal/providers/types_test.go`:

```go
package providers

import (
    "encoding/json"
    "testing"
)

func TestChatRequestJSONRoundTrip(t *testing.T) {
    stream := true
    temp := 0.7
    req := ChatRequest{
        Model: "gpt-4o",
        Messages: []Message{
            {Role: "user", Content: "hello"},
        },
        Stream:      &stream,
        Temperature: &temp,
        Tools: []Tool{
            {
                Type: "function",
                Function: ToolFunction{
                    Name:        "get_weather",
                    Description: "Get weather",
                    Parameters:  json.RawMessage(`{"type":"object"}`),
                },
            },
        },
    }

    data, err := json.Marshal(req)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }

    var got ChatRequest
    if err := json.Unmarshal(data, &got); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }

    if got.Model != "gpt-4o" {
        t.Errorf("model: %q", got.Model)
    }
    if len(got.Messages) != 1 || got.Messages[0].Role != "user" {
        t.Errorf("messages: %+v", got.Messages)
    }
    if got.Stream == nil || !*got.Stream {
        t.Error("stream should be true")
    }
    if got.Temperature == nil || *got.Temperature != 0.7 {
        t.Error("temperature should be 0.7")
    }
    if len(got.Tools) != 1 || got.Tools[0].Function.Name != "get_weather" {
        t.Errorf("tools: %+v", got.Tools)
    }
}

func TestChatRequestMinimal(t *testing.T) {
    input := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
    var req ChatRequest
    if err := json.Unmarshal([]byte(input), &req); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if req.Model != "gpt-4o" {
        t.Errorf("model: %q", req.Model)
    }
    if req.Stream != nil {
        t.Error("stream should be nil")
    }
    if req.Temperature != nil {
        t.Error("temperature should be nil")
    }
    if req.Tools != nil {
        t.Error("tools should be nil")
    }
}

func TestChatResponseJSONRoundTrip(t *testing.T) {
    finish := "stop"
    resp := ChatResponse{
        ID:      "chatcmpl-123",
        Object:  "chat.completion",
        Created: 1700000000,
        Model:   "gpt-4o",
        Choices: []Choice{
            {
                Index:        0,
                Message:      Message{Role: "assistant", Content: "Hello!"},
                FinishReason: &finish,
            },
        },
        Usage: &Usage{
            PromptTokens:     10,
            CompletionTokens: 5,
            TotalTokens:      15,
            PromptTokensDetails: &PromptTokensDetails{
                CachedTokens: 3,
            },
        },
    }

    data, err := json.Marshal(resp)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }

    var got ChatResponse
    if err := json.Unmarshal(data, &got); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }

    if got.ID != "chatcmpl-123" {
        t.Errorf("id: %q", got.ID)
    }
    if len(got.Choices) != 1 {
        t.Fatalf("choices len: %d", len(got.Choices))
    }
    if got.Usage == nil || got.Usage.TotalTokens != 15 {
        t.Errorf("usage: %+v", got.Usage)
    }
    if got.Usage.PromptTokensDetails == nil || got.Usage.PromptTokensDetails.CachedTokens != 3 {
        t.Error("cached tokens mismatch")
    }
}

func TestStreamChunkDelta(t *testing.T) {
    input := `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`
    var chunk StreamChunk
    if err := json.Unmarshal([]byte(input), &chunk); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if len(chunk.Choices) != 1 {
        t.Fatalf("choices: %d", len(chunk.Choices))
    }
    if chunk.Choices[0].Delta.Content == nil || *chunk.Choices[0].Delta.Content != "hello" {
        t.Error("delta content mismatch")
    }
    if chunk.Choices[0].FinishReason != nil {
        t.Error("finish_reason should be nil")
    }
}

func TestStreamChunkFinal(t *testing.T) {
    input := `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
    var chunk StreamChunk
    if err := json.Unmarshal([]byte(input), &chunk); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if chunk.Choices[0].FinishReason == nil || *chunk.Choices[0].FinishReason != "stop" {
        t.Error("finish_reason should be stop")
    }
    if chunk.Usage == nil || chunk.Usage.TotalTokens != 15 {
        t.Errorf("usage: %+v", chunk.Usage)
    }
}

func TestKeyJSONRoundTrip(t *testing.T) {
    key := Key{Value: "sk-test", Provider: ProviderOpenAI, ConnID: "abc", AuthType: "api_key"}
    data, err := json.Marshal(key)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var got Key
    if err := json.Unmarshal(data, &got); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if got.Value != "sk-test" || got.Provider != ProviderOpenAI {
        t.Errorf("key: %+v", got)
    }
}

func TestModelProviderString(t *testing.T) {
    if ProviderOpenAI.String() != "openai" {
        t.Errorf("got %q", ProviderOpenAI.String())
    }
    if ProviderAnthropic.String() != "anthropic" {
        t.Errorf("got %q", ProviderAnthropic.String())
    }
}
```

Run:
```bash
go test ./internal/providers/...
# Expected: FAIL — types don't exist yet
```

#### GREEN: Write Minimum Implementation

Create `internal/providers/types.go` with all types from the type definitions section below.

Create `internal/providers/interface.go`:

```go
package providers

import "context"

type Provider interface {
    Name() ModelProvider
    ChatCompletion(ctx context.Context, key Key, req *ChatRequest) (*ChatResponse, error)
    ChatCompletionStream(ctx context.Context, key Key, req *ChatRequest) (<-chan StreamChunk, error)
    ListModels(ctx context.Context, key Key) ([]Model, error)
}
```

Run:
```bash
go test ./internal/providers/...
# Expected: PASS — all 6 tests green
```

#### REFACTOR

- Check: no unused imports
- Check: all JSON tags match OpenAI spec field names
- Check: `go vet ./...` clean

### Type Definitions

*(Full Go struct definitions for ChatRequest, Message, ContentBlock, Tool, ToolFunction, ToolCall, ToolCallFunc, ChatResponse, Choice, Usage, PromptTokensDetails, CompletionTokensDetails, StreamChunk, StreamChoice, StreamDelta, Key, Model, ModelProvider — as specified in the previous version of this document)*

```go
package providers

import "encoding/json"

// ModelProvider identifies an upstream LLM provider.
type ModelProvider string

const (
    ProviderOpenAI        ModelProvider = "openai"
    ProviderAnthropic     ModelProvider = "anthropic"
    ProviderGemini        ModelProvider = "gemini"
    ProviderGroq          ModelProvider = "groq"
    ProviderCerebras      ModelProvider = "cerebras"
    ProviderMistral       ModelProvider = "mistral"
    ProviderOllama        ModelProvider = "ollama"
    ProviderBedrock       ModelProvider = "bedrock"
    ProviderAzure         ModelProvider = "azure"
    ProviderVertex        ModelProvider = "vertex"
    ProviderOpenRouter    ModelProvider = "openrouter"
    ProviderDeepSeek      ModelProvider = "deepseek"
    ProviderPerplexity    ModelProvider = "perplexity"
    ProviderFireworks     ModelProvider = "fireworks"
    ProviderTogether      ModelProvider = "together"
    ProviderNVIDIA        ModelProvider = "nvidia"
    ProviderHuggingFace   ModelProvider = "huggingface"
    ProviderCohere        ModelProvider = "cohere"
    ProviderReplicate     ModelProvider = "replicate"
    ProviderXAI           ModelProvider = "xai"
    ProviderNebius        ModelProvider = "nebius"
    ProviderGitHubCopilot ModelProvider = "github-copilot"
    ProviderCursor        ModelProvider = "cursor"
)

func (p ModelProvider) String() string { return string(p) }

// Key holds credentials for a single provider request.
type Key struct {
    Value    string        `json:"value"`
    Provider ModelProvider `json:"provider"`
    ConnID   string        `json:"conn_id"`
    AuthType string        `json:"auth_type"`
}

// Model represents an available model.
type Model struct {
    ID       string        `json:"id"`
    Object   string        `json:"object"`
    Created  int64         `json:"created"`
    OwnedBy  string        `json:"owned_by"`
    Provider ModelProvider `json:"-"`
}

// --- Chat Completions ---

type ChatRequest struct {
    Model               string   `json:"model"`
    Messages            []Message `json:"messages"`
    Stream              *bool    `json:"stream,omitempty"`
    Temperature         *float64 `json:"temperature,omitempty"`
    TopP                *float64 `json:"top_p,omitempty"`
    MaxTokens           *int     `json:"max_tokens,omitempty"`
    MaxCompletionTokens *int     `json:"max_completion_tokens,omitempty"`
    Stop                any      `json:"stop,omitempty"`
    Tools               []Tool   `json:"tools,omitempty"`
    ToolChoice          any      `json:"tool_choice,omitempty"`
    ResponseFormat      any      `json:"response_format,omitempty"`
    Seed                *int     `json:"seed,omitempty"`
    FrequencyPenalty    *float64 `json:"frequency_penalty,omitempty"`
    PresencePenalty     *float64 `json:"presence_penalty,omitempty"`
    N                   *int     `json:"n,omitempty"`
    User                *string  `json:"user,omitempty"`
    StreamOptions       any      `json:"stream_options,omitempty"`
    System              any      `json:"system,omitempty"`
    Thinking            any      `json:"thinking,omitempty"`
}

type Message struct {
    Role       string     `json:"role"`
    Content    any        `json:"content"`
    Name       *string    `json:"name,omitempty"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID *string    `json:"tool_call_id,omitempty"`
}

type Tool struct {
    Type     string       `json:"type"`
    Function ToolFunction `json:"function"`
}

type ToolFunction struct {
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ToolCall struct {
    ID       string       `json:"id"`
    Type     string       `json:"type"`
    Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"`
}

type ChatResponse struct {
    ID                string   `json:"id"`
    Object            string   `json:"object"`
    Created           int64    `json:"created"`
    Model             string   `json:"model"`
    Choices           []Choice `json:"choices"`
    Usage             *Usage   `json:"usage,omitempty"`
    SystemFingerprint *string  `json:"system_fingerprint,omitempty"`
}

type Choice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason *string `json:"finish_reason"`
}

type Usage struct {
    PromptTokens            int                      `json:"prompt_tokens"`
    CompletionTokens        int                      `json:"completion_tokens"`
    TotalTokens             int                      `json:"total_tokens"`
    PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
    CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
    CachedTokens int `json:"cached_tokens"`
}

type CompletionTokensDetails struct {
    ReasoningTokens int `json:"reasoning_tokens"`
}

// --- Streaming ---

type StreamChunk struct {
    ID                string         `json:"id"`
    Object            string         `json:"object"`
    Created           int64          `json:"created"`
    Model             string         `json:"model"`
    Choices           []StreamChoice `json:"choices"`
    Usage             *Usage         `json:"usage,omitempty"`
    SystemFingerprint *string        `json:"system_fingerprint,omitempty"`
}

type StreamChoice struct {
    Index        int         `json:"index"`
    Delta        StreamDelta `json:"delta"`
    FinishReason *string     `json:"finish_reason"`
}

type StreamDelta struct {
    Role      *string    `json:"role,omitempty"`
    Content   *string    `json:"content,omitempty"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
```

### Verification

```bash
go test ./internal/providers/... -v    # All 6 tests pass
go vet ./...                            # Clean
```

### Post-conditions

- [x] `internal/providers/types.go` — all types compile
- [x] `internal/providers/interface.go` — Provider interface defined
- [x] `internal/providers/types_test.go` — 6+ tests, all green
- [x] JSON round-trip works for ChatRequest, ChatResponse, StreamChunk, Key
- [x] `go vet ./...` clean

### Commit

```
phase-1/task-1: define core types with OpenAI-compatible JSON
```

---

## Task 1.2: SQLite Store Foundation

### TODO

- [x] `go get modernc.org/sqlite`
- [x] Write `internal/store/sqlite_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/store/sqlite.go` — NewStore, Close, migrate
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-1/task-2: sqlite store foundation with migration`

### Pre-conditions

- `modernc.org/sqlite` in `go.mod`
- No existing `internal/store/` directory

### TDD Cycle

#### RED: Write Failing Tests First

Create `internal/store/sqlite_test.go`:

```go
package store

import (
    "database/sql"
    "os"
    "path/filepath"
    "testing"
)

// openTestStore is a helper used by ALL store tests.
func openTestStore(t *testing.T) *Store {
    t.Helper()
    dir := t.TempDir()
    path := filepath.Join(dir, "test.db")
    s, err := NewStore(path)
    if err != nil {
        t.Fatalf("NewStore: %v", err)
    }
    t.Cleanup(func() { s.Close() })
    return s
}

func TestNewStoreCreatesDatabase(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.db")

    s, err := NewStore(path)
    if err != nil {
        t.Fatalf("NewStore: %v", err)
    }
    defer s.Close()

    if _, err := os.Stat(path); err != nil {
        t.Fatalf("db file missing: %v", err)
    }
}

func TestNewStoreCreatesParentDirs(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "sub", "deep", "test.db")

    s, err := NewStore(path)
    if err != nil {
        t.Fatalf("NewStore: %v", err)
    }
    defer s.Close()

    if _, err := os.Stat(path); err != nil {
        t.Fatalf("db file missing: %v", err)
    }
}

func TestMigrateCreatesTables(t *testing.T) {
    s := openTestStore(t)

    expected := []string{
        "connections", "settings", "api_keys", "combos",
        "model_aliases", "pricing_overrides", "request_log", "mcp_clients",
    }

    for _, table := range expected {
        var name string
        err := s.db.QueryRow(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
        ).Scan(&name)
        if err == sql.ErrNoRows {
            t.Errorf("table %q not created", table)
        } else if err != nil {
            t.Errorf("table %q query error: %v", table, err)
        }
    }
}

func TestMigrateIsIdempotent(t *testing.T) {
    s := openTestStore(t)

    // Migrate again — should not error
    if err := s.migrate(); err != nil {
        t.Fatalf("second migrate: %v", err)
    }

    // Count tables — should be same
    var count int
    s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
    if count < 8 {
        t.Errorf("expected >= 8 tables, got %d", count)
    }
}

func TestStoreImplementsCloser(t *testing.T) {
    // Compile-time check
    var _ interface{ Close() error } = (*Store)(nil)
}
```

Run:
```bash
go test ./internal/store/...
# Expected: FAIL — Store type doesn't exist
```

#### GREEN: Write Minimum Implementation

Create `internal/store/sqlite.go`:

```go
package store

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"

    _ "modernc.org/sqlite"
)

type Store struct {
    path string
    db   *sql.DB
}

func NewStore(path string) (*Store, error) {
    if dir := filepath.Dir(path); dir != "" && dir != "." {
        if err := os.MkdirAll(dir, 0o755); err != nil {
            return nil, fmt.Errorf("create data dir: %w", err)
        }
    }

    db, err := sql.Open("sqlite", path)
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }

    // Pragmas
    for _, pragma := range []string{
        "PRAGMA journal_mode = WAL",
        "PRAGMA busy_timeout = 5000",
        "PRAGMA foreign_keys = ON",
        "PRAGMA synchronous = NORMAL",
    } {
        if _, err := db.Exec(pragma); err != nil {
            db.Close()
            return nil, fmt.Errorf("pragma %q: %w", pragma, err)
        }
    }

    s := &Store{path: path, db: db}
    if err := s.migrate(); err != nil {
        db.Close()
        return nil, fmt.Errorf("migrate: %w", err)
    }
    return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
    // All DDL from docs/SCHEMA.md
    ddl := []string{
        `CREATE TABLE IF NOT EXISTS connections (
            id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
            provider TEXT NOT NULL,
            name TEXT NOT NULL DEFAULT '',
            auth_type TEXT NOT NULL CHECK (auth_type IN ('oauth', 'api_key', 'noauth')),
            access_token TEXT,
            refresh_token TEXT,
            expires_at INTEGER,
            api_key TEXT,
            is_active INTEGER NOT NULL DEFAULT 1,
            provider_specific_data TEXT,
            account_id TEXT,
            email TEXT,
            unavailable_until INTEGER,
            backoff_level INTEGER NOT NULL DEFAULT 0,
            model_locks TEXT,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
        `CREATE INDEX IF NOT EXISTS idx_connections_provider ON connections(provider)`,
        `CREATE INDEX IF NOT EXISTS idx_connections_active ON connections(provider, is_active)`,
        // ... settings, api_keys, combos, model_aliases, pricing_overrides, request_log, mcp_clients
        // (full DDL from SCHEMA.md — each table as a separate string)
    }
    for _, stmt := range ddl {
        if _, err := s.db.Exec(stmt); err != nil {
            return fmt.Errorf("exec %q: %w", stmt[:40], err)
        }
    }
    return nil
}
```

Run:
```bash
go test ./internal/store/... -v
# Expected: PASS — all 5 tests green
```

#### REFACTOR

- Verify: all 8 tables from SCHEMA.md are in the DDL
- Verify: indexes created
- Verify: pragmas set correctly

### Verification

```bash
go test ./internal/store/... -v -count=1
go vet ./...
```

### Post-conditions

- [x] `NewStore(path)` creates SQLite DB file
- [x] `NewStore` creates parent directories
- [x] All 8 tables exist after migration
- [x] Second `migrate()` is idempotent
- [x] `Store` implements `io.Closer`
- [x] WAL mode enabled

### Commit

```
phase-1/task-2: sqlite store foundation with migration
```

---

## Task 1.3: Connection CRUD

### TODO

- [x] Write `internal/store/connections_test.go` — **test file FIRST**
- [x] Write `internal/store/errors.go` — `ErrNotFound`
- [x] Run tests → see RED
- [x] Write `internal/store/connections.go`
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-1/task-3: connection CRUD with provider filtering`

### Pre-conditions

- Task 1.2 complete (Store struct, migration)
- `openTestStore(t)` helper available

### TDD Cycle

#### RED: Write Failing Tests First

Create `internal/store/errors.go`:
```go
package store

import "errors"

var ErrNotFound = errors.New("store: not found")
```

Create `internal/store/connections_test.go`:

```go
package store

import (
    "testing"
    "time"
)

func TestConnectionCreateAndGetByID(t *testing.T) {
    s := openTestStore(t)

    conn := &Connection{
        Provider:    "anthropic",
        Name:        "work",
        AuthType:    AuthTypeOAuth,
        AccessToken: strPtr("tok-123"),
        RefreshToken: strPtr("ref-456"),
        ExpiresAt:   int64Ptr(time.Now().Add(time.Hour).Unix()),
        IsActive:    true,
        Email:       strPtr("user@example.com"),
    }

    if err := s.CreateConnection(conn); err != nil {
        t.Fatalf("CreateConnection: %v", err)
    }
    if conn.ID == "" {
        t.Fatal("ID should be set after create")
    }

    got, err := s.GetConnection(conn.ID)
    if err != nil {
        t.Fatalf("GetConnection: %v", err)
    }
    if got.Provider != "anthropic" || got.Name != "work" {
        t.Errorf("got: %+v", got)
    }
    if got.AccessToken == nil || *got.AccessToken != "tok-123" {
        t.Error("access_token mismatch")
    }
    if got.Email == nil || *got.Email != "user@example.com" {
        t.Error("email mismatch")
    }
}

func TestConnectionGetNotFound(t *testing.T) {
    s := openTestStore(t)
    _, err := s.GetConnection("nonexistent")
    if err != ErrNotFound {
        t.Fatalf("expected ErrNotFound, got: %v", err)
    }
}

func TestConnectionGetByProvider(t *testing.T) {
    s := openTestStore(t)

    for _, p := range []string{"anthropic", "anthropic", "openai"} {
        s.CreateConnection(&Connection{Provider: p, AuthType: AuthTypeAPIKey, IsActive: true})
    }

    conns, err := s.GetConnections("anthropic")
    if err != nil {
        t.Fatalf("GetConnections: %v", err)
    }
    if len(conns) != 2 {
        t.Errorf("expected 2, got %d", len(conns))
    }
}

func TestConnectionGetActive(t *testing.T) {
    s := openTestStore(t)

    s.CreateConnection(&Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true})
    s.CreateConnection(&Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: false})

    active, err := s.GetActiveConnections("openai")
    if err != nil {
        t.Fatalf("GetActiveConnections: %v", err)
    }
    if len(active) != 1 {
        t.Errorf("expected 1 active, got %d", len(active))
    }
}

func TestConnectionUpdate(t *testing.T) {
    s := openTestStore(t)

    conn := &Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true}
    s.CreateConnection(conn)

    conn.Name = "renamed"
    conn.IsActive = false
    if err := s.UpdateConnection(conn); err != nil {
        t.Fatalf("UpdateConnection: %v", err)
    }

    got, _ := s.GetConnection(conn.ID)
    if got.Name != "renamed" || got.IsActive {
        t.Errorf("update failed: %+v", got)
    }
}

func TestConnectionDelete(t *testing.T) {
    s := openTestStore(t)

    conn := &Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true}
    s.CreateConnection(conn)

    if err := s.DeleteConnection(conn.ID); err != nil {
        t.Fatalf("DeleteConnection: %v", err)
    }

    _, err := s.GetConnection(conn.ID)
    if err != ErrNotFound {
        t.Fatalf("expected ErrNotFound after delete, got: %v", err)
    }
}

func TestConnectionProviderSpecificDataRoundTrip(t *testing.T) {
    s := openTestStore(t)

    conn := &Connection{
        Provider:             "anthropic",
        AuthType:             AuthTypeOAuth,
        IsActive:             true,
        ProviderSpecificData: map[string]any{"tier": "pro", "org_id": "org-123"},
    }
    s.CreateConnection(conn)

    got, _ := s.GetConnection(conn.ID)
    if got.ProviderSpecificData == nil {
        t.Fatal("ProviderSpecificData is nil")
    }
    if got.ProviderSpecificData["tier"] != "pro" {
        t.Errorf("tier: %v", got.ProviderSpecificData["tier"])
    }
}

// Helpers
func strPtr(s string) *string    { return &s }
func int64Ptr(i int64) *int64    { return &i }
func intPtr(i int) *int          { return &i }
func floatPtr(f float64) *float64 { return &f }
```

Run:
```bash
go test ./internal/store/...
# Expected: FAIL — Connection type and methods don't exist
```

#### GREEN: Implement

Create `internal/store/connections.go` with:
- `AuthType` enum (`oauth`, `api_key`, `noauth`)
- `Connection` struct (all fields per SCHEMA.md)
- `CreateConnection` — INSERT, sets ID from DB default
- `GetConnection` — SELECT by ID, returns `ErrNotFound` if missing
- `GetConnections` — SELECT by provider
- `GetActiveConnections` — SELECT by provider WHERE is_active=1
- `UpdateConnection` — UPDATE by ID, sets updated_at
- `DeleteConnection` — DELETE by ID, returns `ErrNotFound` if 0 rows affected
- JSON marshal/unmarshal for `ProviderSpecificData` and `ModelLocks` map fields

Run:
```bash
go test ./internal/store/... -v
# Expected: PASS — all 7 tests green
```

#### REFACTOR

- Verify: no SQL injection (all `?` params)
- Verify: `updated_at` set on update
- Verify: `created_at` set on create

### Verification

```bash
go test ./internal/store/... -v -count=1
go vet ./...
```

### Post-conditions

- [x] Connection CRUD works (create, get by ID, get by provider, update, delete)
- [x] `ErrNotFound` returned for missing connections
- [x] `ProviderSpecificData` JSON round-trips
- [x] Active filtering works

### Commit

```
phase-1/task-3: connection CRUD with provider filtering
```

---

## Task 1.4: Settings + API Keys Store

### TODO

- [x] Write `internal/store/settings_test.go` — **test file FIRST**
- [x] Write `internal/store/apikeys_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/store/settings.go`
- [x] Write `internal/store/apikeys.go`
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-1/task-4: settings and API key store with HMAC validation`

### Pre-conditions

- Task 1.2 complete (Store struct, migration)
- `openTestStore(t)` helper available

### TDD Cycle

#### RED: Write Failing Tests First

**Settings tests** — `internal/store/settings_test.go`:
- `TestGetSettingsDefaults` — empty DB → returns default Settings (require_api_key=true, rtk=true, caveman=false, level="full")
- `TestUpdateAndGetSettings` — update → get → values match
- `TestUpdateSettingsIdempotent` — update twice, no error

**API Key tests** — `internal/store/apikeys_test.go`:
- `TestCreateAPIKeyReturnsRawKey` — raw key starts with `g0r_`
- `TestValidateAPIKeyCorrect` — correct key → `(meta, true)`
- `TestValidateAPIKeyWrong` — wrong key → `(nil, false)`
- `TestValidateAPIKeyAfterDelete` — deleted key → `(nil, false)`
- `TestListAPIKeys` — returns records without hashes
- `TestCreateAPIKeyDuplicateName` — duplicate name → error

#### GREEN: Implement

**Settings**: Key-value store — each field is a row in `settings` table.
- `GetSettings()` reads all rows, maps to struct, fills defaults for missing keys
- `UpdateSettings()` upserts each field as `INSERT OR REPLACE`

**API Keys**: HMAC-SHA256 hashing.
1. `CreateAPIKey(name, secret)` → generate 32 random bytes → hex → prefix `g0r_` → HMAC-SHA256(raw, secret) → store hash
2. `ValidateAPIKey(raw, secret)` → compute HMAC → query by hash + is_active=1
3. `ListAPIKeys()` → SELECT id, name, prefix, is_active, last_used_at, created_at
4. `DeleteAPIKey(id)` → UPDATE is_active=0

### Verification

```bash
go test ./internal/store/... -v -count=1  # All store tests pass
go vet ./...
```

### Post-conditions

- [x] Default settings returned on empty DB
- [x] Settings round-trip through update/get
- [x] API key HMAC validation works
- [x] Deleted keys fail validation
- [x] Duplicate names rejected

### Commit

```
phase-1/task-4: settings and API key store with HMAC validation
```

---

## Task 1.5: Usage Log Store

### TODO

- [x] Write `internal/store/usage_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/store/usage.go`
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-1/task-5: usage log store with filtering and aggregation`

### Pre-conditions

- Task 1.2 complete

### TDD Cycle

#### RED: Write Failing Tests First

`internal/store/usage_test.go`:
- `TestLogRequestAndGetUsage` — insert entry → get returns it
- `TestLogRequestNullableFields` — insert with nil optional fields → no scan errors
- `TestGetUsageFilterByProvider` — 3 entries (2 openai, 1 anthropic) → filter → 2
- `TestGetUsageFilterByDateRange` — entries at different times → range filter works
- `TestGetUsagePagination` — 10 entries → limit=3,offset=2 → returns 3
- `TestGetUsageSummary` — 3 entries → correct totals
- `TestGetUsageSummaryEmpty` — no entries → all zeros

#### GREEN: Implement

Key implementation details:
- `LogRequest` — INSERT with all nullable columns using `?` placeholders
- `GetUsage` — Dynamic WHERE clause built from non-nil filter fields
- `GetUsageSummary` — `SELECT COUNT(*), COALESCE(SUM(...), 0)` with same WHERE
- All nullable INTEGER/REAL scanned via `sql.NullInt64`/`sql.NullFloat64` → converted to `*int`/`*float64`
- Boolean columns (rtk_enabled, caveman_enabled) stored as INTEGER, scanned as NullInt64

### Verification

```bash
go test ./internal/store/... -v -count=1
go vet ./...
```

### Commit

```
phase-1/task-5: usage log store with filtering and aggregation
```

---

## Task 1.6: Config Loading

### TODO

- [x] Write `internal/config/config_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/config/config.go`
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-1/task-6: config loading from env with validation`

### Pre-conditions

- No external dependencies needed

### TDD Cycle

#### RED: Write Failing Tests First

`internal/config/config_test.go`:
- `TestLoadDefaults` — clear env, set API_KEY_SECRET → port=20128, rtk=true, caveman=false
- `TestLoadFromEnv` — set all env vars → config matches
- `TestLoadValidatesAPIKeySecret` — REQUIRE_API_KEY=true, no secret → error
- `TestLoadRequireAPIKeyFalse` — no secret needed → ok
- `TestLoadInvalidPort` — PORT=99999 → error
- `TestLoadInvalidCavemanLevel` — CAVEMAN_LEVEL=mega → error
- `TestLoadBooleanParsing` — "yes", "1", "TRUE" → all true
- `TestLoadHomeDirExpansion` — DATA_DIR not set → contains home dir

**Important**: Each test must call `t.Setenv()` (not `os.Setenv`) to avoid test pollution.

#### GREEN: Implement

```go
func Load() (*Config, error)
```

1. `os.Getenv` for each variable
2. Apply defaults for missing
3. Expand `~` in DataDir via `os.UserHomeDir()`
4. Parse booleans: accept true/1/yes/false/0/no (case-insensitive)
5. Validate: port range, required secrets, caveman level enum
6. Return error (not panic) on validation failure

### Verification

```bash
go test ./internal/config/... -v -count=1
go test ./... -count=1   # ALL tests pass (types + store + config)
go vet ./...
```

### Commit

```
phase-1/task-6: config loading from env with validation
```

---

## Phase Gate

```bash
go test ./... -count=1   # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary still builds
```

**Expected test count**: ~25+ tests across 6 test files.

## Phase Checklist

- [x] Task 1.1 complete (core types)
- [x] Task 1.2 complete (SQLite store)
- [x] Task 1.3 complete (connection CRUD)
- [x] Task 1.4 complete (settings + API keys)
- [x] Task 1.5 complete (usage log)
- [x] Task 1.6 complete (config loading)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-1/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_1.status → `DONE`
- [x] Update `docs/WORKFLOW.md`: current_phase → `2`
- [x] **PHASE_1_COMPLETE**
