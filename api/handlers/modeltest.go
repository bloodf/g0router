package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/modelcatalog"
	providerinfo "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type modelTestStore interface {
	GetActiveConnections(provider string) ([]*store.Connection, error)
	ListConnections() ([]*store.Connection, error)
	AppendAudit(entry store.AuditEntry) error
}

type modelTestResult struct {
	OK        bool    `json:"ok"`
	LatencyMS int64   `json:"latency_ms"`
	Error     *string `json:"error"`
}

type modelTestBatchResult struct {
	Provider     string  `json:"provider"`
	ConnectionID string  `json:"connection_id"`
	OK           bool    `json:"ok"`
	LatencyMS    int64   `json:"latency_ms"`
	Error        *string `json:"error"`
}

func ModelTest(ctx *fasthttp.RequestCtx, s modelTestStore, adapterSource ProviderAdapterSource, providerID, model string) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	providerID = providerinfo.CanonicalProviderID(providerID)
	_, ok := providerinfo.ProviderMatrix().Provider(providerID)
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}

	connections, err := s.GetActiveConnections(providerID)
	if err != nil {
		log.Printf("get active connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get connections")
		return
	}
	if len(connections) == 0 {
		writeError(ctx, fasthttp.StatusBadRequest, "no active connections for provider")
		return
	}

	if adapterSource == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "adapter source unavailable")
		return
	}

	adapter, ok := adapterSource.GetProvider(providers.ModelProvider(providerID))
	if !ok {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "provider adapter unavailable")
		return
	}

	var reqBody struct {
		Messages []providers.Message `json:"messages"`
	}
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &reqBody); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid request body")
			return
		}
	}

	messages := reqBody.Messages
	if len(messages) == 0 {
		messages = []providers.Message{{Role: "user", Content: "Say hello"}}
	}

	conn := connections[0]
	key := providers.Key{
		Provider: providers.ModelProvider(providerID),
		ConnID:   conn.ID,
		AuthType: string(conn.AuthType),
	}
	if conn.APIKey != nil {
		key.Value = *conn.APIKey
	} else if conn.AccessToken != nil {
		key.Value = *conn.AccessToken
	}
	if conn.AccountID != nil {
		key.AccountID = *conn.AccountID
	}

	chatReq := &providers.ChatRequest{
		Model:    model,
		Messages: messages,
	}

	start := time.Now()
	_, err = adapter.ChatCompletion(requestContext(ctx), key, chatReq)
	latency := time.Since(start).Milliseconds()

	result := modelTestResult{OK: true, LatencyMS: latency}
	if err != nil {
		result.OK = false
		result.LatencyMS = 0
		msg := err.Error()
		result.Error = &msg
	}

	if result.OK {
		_ = s.AppendAudit(store.AuditEntry{
			Action: "model.test",
			Target: providerID + "/" + model,
		})
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": result})
}

func ModelTestBatch(ctx *fasthttp.RequestCtx, s modelTestStore, adapterSource ProviderAdapterSource) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if adapterSource == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "adapter source unavailable")
		return
	}

	var reqBody struct {
		Providers []string `json:"providers"`
	}
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &reqBody); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid request body")
			return
		}
	}

	providerSet := make(map[string]bool, len(reqBody.Providers))
	for _, p := range reqBody.Providers {
		providerSet[providerinfo.CanonicalProviderID(p)] = true
	}

	connections, err := s.ListConnections()
	if err != nil {
		log.Printf("list connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list connections")
		return
	}

	var toTest []*store.Connection
	for _, conn := range connections {
		if !conn.IsActive {
			continue
		}
		if len(providerSet) > 0 && !providerSet[conn.Provider] {
			continue
		}
		toTest = append(toTest, conn)
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)

	baseCtx := requestContext(ctx)
	defaultMessages := []providers.Message{{Role: "user", Content: "Say hello"}}

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		results := make(chan modelTestBatchResult, len(toTest))
		var wg sync.WaitGroup

		for _, conn := range toTest {
			wg.Add(1)
			go func(conn *store.Connection) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						msg := fmt.Sprintf("panic: %v", r)
						results <- modelTestBatchResult{
							Provider:     conn.Provider,
							ConnectionID: conn.ID,
							OK:           false,
							LatencyMS:    0,
							Error:        &msg,
						}
					}
				}()
				results <- testConnection(baseCtx, adapterSource, conn, defaultMessages)
			}(conn)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		for res := range results {
			data, err := json.Marshal(res)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: result\ndata: %s\n\n", data); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}

		done, _ := json.Marshal(map[string]any{})
		if _, err := fmt.Fprintf(w, "event: done\ndata: %s\n\n", done); err != nil {
			return
		}
		_ = w.Flush()
	})
}

func testConnection(ctx context.Context, adapterSource ProviderAdapterSource, conn *store.Connection, messages []providers.Message) modelTestBatchResult {
	provider := providers.ModelProvider(conn.Provider)
	adapter, ok := adapterSource.GetProvider(provider)
	if !ok {
		msg := "provider adapter unavailable"
		return modelTestBatchResult{
			Provider:     conn.Provider,
			ConnectionID: conn.ID,
			OK:           false,
			LatencyMS:    0,
			Error:        &msg,
		}
	}

	model := pickTestModel(provider, conn)
	key := providers.Key{
		Provider: provider,
		ConnID:   conn.ID,
		AuthType: string(conn.AuthType),
	}
	if conn.APIKey != nil {
		key.Value = *conn.APIKey
	} else if conn.AccessToken != nil {
		key.Value = *conn.AccessToken
	}
	if conn.AccountID != nil {
		key.AccountID = *conn.AccountID
	}

	chatReq := &providers.ChatRequest{
		Model:    model,
		Messages: messages,
	}

	start := time.Now()
	_, err := adapter.ChatCompletion(ctx, key, chatReq)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		msg := err.Error()
		return modelTestBatchResult{
			Provider:     conn.Provider,
			ConnectionID: conn.ID,
			OK:           false,
			LatencyMS:    0,
			Error:        &msg,
		}
	}

	return modelTestBatchResult{
		Provider:     conn.Provider,
		ConnectionID: conn.ID,
		OK:           true,
		LatencyMS:    latency,
		Error:        nil,
	}
}

func pickTestModel(provider providers.ModelProvider, conn *store.Connection) string {
	if len(conn.ModelLocks) > 0 {
		for model := range conn.ModelLocks {
			return model
		}
	}
	models := modelcatalog.NewCatalog().Models(provider)
	for model := range models {
		return model
	}
	return string(provider)
}
