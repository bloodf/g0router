package handlers

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var wsUpgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

type wsFeatureFlagStore interface {
	GetFeatureFlagByKey(key string) (*store.FeatureFlag, error)
}

// WSChat handles GET /api/ws WebSocket upgrades.
// Auth is enforced by applyMiddleware before the handler is reached.
func WSChat(ctx *fasthttp.RequestCtx, engine InferenceEngine, flagStore wsFeatureFlagStore) {
	if engine == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}

	if flagStore != nil {
		flag, err := flagStore.GetFeatureFlagByKey("websocket_chat")
		if err != nil || flag == nil || !flag.Enabled {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}
	}

	if err := wsUpgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		wsHandleConnection(conn, engine)
	}); err != nil {
		log.Printf("websocket upgrade failed: %v", err)
	}
}

type wsClientMessage struct {
	Type      string              `json:"type"`
	SessionID string              `json:"session_id"`
	Model     string              `json:"model"`
	Messages  []providers.Message `json:"messages"`
}

type wsDeltaMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type wsDoneMessage struct {
	Type  string           `json:"type"`
	Usage *providers.Usage `json:"usage"`
}

type wsErrorMessage struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

func wsHandleConnection(conn *websocket.Conn, engine InferenceEngine) {
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	var msg wsClientMessage
	if err := conn.ReadJSON(&msg); err != nil {
		_ = conn.WriteJSON(wsErrorMessage{Type: "error", Error: "invalid message"})
		return
	}
	if msg.Type != "chat" {
		_ = conn.WriteJSON(wsErrorMessage{Type: "error", Error: "expected type chat"})
		return
	}
	if msg.Model == "" {
		_ = conn.WriteJSON(wsErrorMessage{Type: "error", Error: "model required"})
		return
	}

	req := &providers.ChatRequest{
		Model:    msg.Model,
		Messages: msg.Messages,
	}
	stream := true
	req.Stream = &stream

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var closeOnce sync.Once
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				closeOnce.Do(func() {
					cancel()
				})
				return
			}
		}
	}()

	streamCh, err := engine.DispatchStream(streamCtx, req)
	if err != nil {
		_ = conn.WriteJSON(wsErrorMessage{Type: "error", Error: "dispatch error"})
		_ = conn.Close()
		select {
		case <-readDone:
		case <-time.After(2 * time.Second):
		}
		return
	}

	_ = conn.SetReadDeadline(time.Time{})

	var lastUsage *providers.Usage
streamLoop:
	for {
		select {
		case <-streamCtx.Done():
			_ = conn.Close()
			select {
			case <-readDone:
			case <-time.After(2 * time.Second):
			}
			return
		case chunk, ok := <-streamCh:
			if !ok {
				break streamLoop
			}
			if chunk.Error != nil {
				_ = conn.WriteJSON(wsErrorMessage{Type: "error", Error: chunk.Error.Message})
				_ = conn.Close()
				select {
			case <-readDone:
			case <-time.After(2 * time.Second):
			}
			return
			}
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != nil && *choice.Delta.Content != "" {
					if err := conn.WriteJSON(wsDeltaMessage{Type: "delta", Content: *choice.Delta.Content}); err != nil {
						_ = conn.Close()
						select {
						case <-readDone:
						case <-time.After(2 * time.Second):
						}
						return
					}
				}
			}
			if chunk.Usage != nil {
				lastUsage = chunk.Usage
			}
		}
	}

	done := wsDoneMessage{Type: "done"}
	if lastUsage != nil {
		done.Usage = lastUsage
	}
	_ = conn.WriteJSON(done)

	// Close and wait for reader goroutine before returning so fasthttp doesn't
	// reclaim the underlying hijackConn while the goroutine is still reading.
	_ = conn.Close()
	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
	}
}
