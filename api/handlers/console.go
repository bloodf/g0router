package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bloodf/g0router/internal/console"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type consoleBroker interface {
	Subscribe() (uint64, <-chan console.Entry)
	Unsubscribe(id uint64)
	Recent() []console.Entry
	Clear()
}

// ConsoleLogsStream serves GET /api/console-logs/stream as a Server-Sent Events
// feed. It replays the ring buffer for initial hydration, then delivers live
// entries as they are published.
func ConsoleLogsStream(ctx *fasthttp.RequestCtx, broker consoleBroker, stopCh <-chan struct{}) {
	if broker == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		subID, ch := broker.Subscribe()
		defer broker.Unsubscribe(subID)

		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		for _, ent := range broker.Recent() {
			if !writeConsoleLogEvent(w, ent) {
				return
			}
		}
		if err := w.Flush(); err != nil {
			return
		}

		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-stopCh:
				return
			case ent, ok := <-ch:
				if !ok {
					return
				}
				if !writeConsoleLogEvent(w, ent) {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}

func writeConsoleLogEvent(w *bufio.Writer, ent console.Entry) bool {
	data, err := json.Marshal(ent)
	if err != nil {
		return true
	}
	if _, err := fmt.Fprintf(w, "event: log\ndata: %s\n\n", data); err != nil {
		return false
	}
	return true
}

// ConsoleLogsClear serves DELETE /api/console-logs. It clears the console
// broker ring buffer and records an audit entry.
func ConsoleLogsClear(ctx *fasthttp.RequestCtx, broker consoleBroker, audit auditWriter) {
	if broker == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}
	broker.Clear()
	if audit != nil {
		if err := audit.AppendAudit(store.AuditEntry{
			Action: "console_logs.clear",
		}); err != nil {
			log.Printf("append audit: %v", err)
		}
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}
