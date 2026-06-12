package usage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// DetailStore persists request detail rows.
type DetailStore interface {
	SaveRequestDetails(items []*store.RequestDetailRow, maxRecords int) error
}

// RequestDetail is a single observed request to be buffered and persisted.
type RequestDetail struct {
	ID               string
	Timestamp        string
	Provider         string
	Model            string
	ConnectionID     string
	Status           string
	Latency          any
	Tokens           any
	Request          any
	ProviderRequest  any
	ProviderResponse any
	Response         any
}

// DetailWriter buffers request details and flushes them to the store in batches.
type DetailWriter struct {
	store        DetailStore
	config       *ObsConfigLoader
	clock        func() time.Time
	timerFactory func(time.Duration, func()) func()
	randRead     func([]byte) (int, error)

	mu         sync.Mutex
	buffer     []RequestDetail
	stopTimer  func()
	closed     bool
}

// NewDetailWriter creates a writer with injected dependencies.
func NewDetailWriter(store DetailStore, config *ObsConfigLoader, clock func() time.Time, timerFactory func(time.Duration, func()) func(), randRead func([]byte) (int, error)) *DetailWriter {
	if timerFactory == nil {
		timerFactory = func(d time.Duration, fn func()) func() {
			t := time.AfterFunc(d, fn)
			return func() { t.Stop() }
		}
	}
	if randRead == nil {
		randRead = rand.Read
	}
	return &DetailWriter{
		store:        store,
		config:       config,
		clock:        clock,
		timerFactory: timerFactory,
		randRead:     randRead,
	}
}

// Save buffers a detail. It flushes immediately when the batch threshold is reached.
func (w *DetailWriter) Save(detail RequestDetail) error {
	cfg := w.config.Load()
	if !cfg.Enabled {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("detail writer is closed")
	}

	w.buffer = append(w.buffer, detail)
	if len(w.buffer) >= cfg.BatchSize {
		if w.stopTimer != nil {
			w.stopTimer()
			w.stopTimer = nil
		}
		return w.flushLocked(cfg)
	}

	if w.stopTimer == nil {
		w.stopTimer = w.timerFactory(time.Duration(cfg.FlushIntervalMs)*time.Millisecond, func() {
			w.mu.Lock()
			cfg := w.config.Load()
			w.stopTimer = nil
			_ = w.flushLocked(cfg)
			w.mu.Unlock()
		})
	}
	return nil
}

// Close flushes any buffered items and stops the timer.
func (w *DetailWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopTimer != nil {
		w.stopTimer()
		w.stopTimer = nil
	}
	if w.closed {
		return nil
	}
	w.closed = true
	cfg := w.config.Load()
	return w.flushLocked(cfg)
}

func (w *DetailWriter) flushLocked(cfg ObsConfig) error {
	if len(w.buffer) == 0 {
		return nil
	}

	items := w.buffer
	w.buffer = nil

	var rows []*store.RequestDetailRow
	for _, d := range items {
		row, err := w.prepareRow(d, cfg)
		if err != nil {
			return fmt.Errorf("prepare detail row: %w", err)
		}
		rows = append(rows, row)
	}

	if err := w.store.SaveRequestDetails(rows, cfg.MaxRecords); err != nil {
		return fmt.Errorf("save request details: %w", err)
	}
	return nil
}

func (w *DetailWriter) prepareRow(d RequestDetail, cfg ObsConfig) (*store.RequestDetailRow, error) {
	timestamp := d.Timestamp
	if timestamp == "" {
		timestamp = w.clock().UTC().Format("2006-01-02T15:04:05.000Z07:00")
	}

	id := d.ID
	if id == "" {
		var err error
		id, err = generateDetailID(timestamp, d.Model, w.randRead)
		if err != nil {
			return nil, fmt.Errorf("generate detail id: %w", err)
		}
	}

	request := d.Request
	if reqMap, ok := request.(map[string]any); ok {
		switch h := reqMap["headers"].(type) {
		case map[string]string:
			reqMap["headers"] = SanitizeHeaders(h)
		case map[string]any:
			strHeaders := make(map[string]string)
			for k, v := range h {
				if s, ok := v.(string); ok {
					strHeaders[k] = s
				}
			}
			reqMap["headers"] = SanitizeHeaders(strHeaders)
		}
	}

	record := map[string]any{
		"id":               id,
		"provider":         nilIfEmpty(d.Provider),
		"model":            nilIfEmpty(d.Model),
		"connectionId":     nilIfEmpty(d.ConnectionID),
		"timestamp":        timestamp,
		"status":           nilIfEmpty(d.Status),
		"latency":          TruncateField(d.Latency, cfg.MaxJSONSize),
		"tokens":           TruncateField(d.Tokens, cfg.MaxJSONSize),
		"request":          TruncateField(request, cfg.MaxJSONSize),
		"providerRequest":  TruncateField(d.ProviderRequest, cfg.MaxJSONSize),
		"providerResponse": TruncateField(d.ProviderResponse, cfg.MaxJSONSize),
		"response":         TruncateField(d.Response, cfg.MaxJSONSize),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal detail record: %w", err)
	}

	return &store.RequestDetailRow{
		ID:           id,
		Timestamp:    timestamp,
		Provider:     d.Provider,
		Model:        d.Model,
		ConnectionID: d.ConnectionID,
		Status:       d.Status,
		Data:         data,
	}, nil
}

var modelSlugRe = regexp.MustCompile(`[^a-zA-Z0-9-]`)

func generateDetailID(timestamp, model string, randRead func([]byte) (int, error)) (string, error) {
	b := make([]byte, 6)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	var sb strings.Builder
	for _, x := range b {
		sb.WriteByte(chars[int(x)%len(chars)])
	}

	modelPart := model
	if modelPart == "" {
		modelPart = "unknown"
	}
	modelPart = modelSlugRe.ReplaceAllString(modelPart, "-")
	return fmt.Sprintf("%s-%s-%s", timestamp, sb.String(), modelPart), nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
