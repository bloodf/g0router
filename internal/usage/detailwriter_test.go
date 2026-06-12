package usage

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

type fakeRequestDetailStore struct {
	mu     sync.Mutex
	items  []*store.RequestDetailRow
	err    error
	maxRec int
}

func (f *fakeRequestDetailStore) SaveRequestDetails(items []*store.RequestDetailRow, maxRecords int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.items = append(f.items, items...)
	f.maxRec = maxRecords
	return nil
}

func (f *fakeRequestDetailStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

type writerTestTimer struct {
	fn      func()
	stopped bool
}

func newTestWriterDeps() (*fakeRequestDetailStore, *ObsConfigLoader, func() time.Time, func(time.Duration, func()) func(), *[]writerTestTimer, func()) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	s := &fakeSettingsReader{values: map[string]string{}}
	loader := NewObsConfigLoader(s, func(string) string { return "" }, clock)

	var timers []writerTestTimer
	var mu sync.Mutex
	timerFactory := func(d time.Duration, fn func()) func() {
		mu.Lock()
		defer mu.Unlock()
		timers = append(timers, writerTestTimer{fn: fn})
		idx := len(timers) - 1
		return func() {
			mu.Lock()
			defer mu.Unlock()
			timers[idx].stopped = true
		}
	}
	fireLast := func() {
		mu.Lock()
		fn := timers[len(timers)-1].fn
		mu.Unlock()
		fn()
	}
	return &fakeRequestDetailStore{}, loader, clock, timerFactory, &timers, fireLast
}

func newRealWriterStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "g0router.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestWriterFlushAtBatchSize(t *testing.T) {
	st, loader, clock, tf, timers, _ := newTestWriterDeps()
	loader.settings = &fakeSettingsReader{values: map[string]string{"observabilityBatchSize": "2"}}
	w := NewDetailWriter(st, loader, clock, tf, rand.Read)

	if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	if st.count() != 0 {
		t.Fatalf("count after first save = %d, want 0", st.count())
	}
	if len(*timers) != 1 {
		t.Fatalf("timers = %d, want 1", len(*timers))
	}

	if err := w.Save(&RequestDetail{Model: "claude-3"}); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	if st.count() != 2 {
		t.Fatalf("count after second save = %d, want 2", st.count())
	}
	if !(*timers)[0].stopped {
		t.Error("timer not cancelled on batch flush")
	}
}

func TestWriterTimerFlush(t *testing.T) {
	st, loader, clock, tf, _, fireLast := newTestWriterDeps()
	w := NewDetailWriter(st, loader, clock, tf, rand.Read)

	if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if st.count() != 0 {
		t.Fatalf("count before timer = %d, want 0", st.count())
	}

	fireLast()
	if st.count() != 1 {
		t.Fatalf("count after timer = %d, want 1", st.count())
	}
}

func TestWriterRetention(t *testing.T) {
	st := newRealWriterStore(t)
	s := &fakeSettingsReader{values: map[string]string{
		"observabilityBatchSize":       "1",
		"observabilityMaxRecords":      "3",
		"observabilityFlushIntervalMs": "1000",
	}}
	loader := NewObsConfigLoader(s, func(string) string { return "" }, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) })
	w := NewDetailWriter(st, loader, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }, nil, rand.Read)

	for i := 0; i < 5; i++ {
		if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	rows, _, err := st.QueryRequestDetails(store.RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
}

func TestWriterDisabledDrops(t *testing.T) {
	st := newRealWriterStore(t)
	s := &fakeSettingsReader{values: map[string]string{"enableObservability": "false"}}
	loader := NewObsConfigLoader(s, func(string) string { return "" }, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) })
	w := NewDetailWriter(st, loader, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }, nil, rand.Read)

	if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	rows, _, err := st.QueryRequestDetails(store.RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("len(rows) = %d, want 0", len(rows))
	}
}

func TestWriterCloseFlushes(t *testing.T) {
	st := newRealWriterStore(t)
	s := &fakeSettingsReader{values: map[string]string{"observabilityBatchSize": "5"}}
	loader := NewObsConfigLoader(s, func(string) string { return "" }, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) })
	w := NewDetailWriter(st, loader, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }, nil, rand.Read)

	if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	rows, _, err := st.QueryRequestDetails(store.RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
}

func TestWriterConcurrent(t *testing.T) {
	st := newRealWriterStore(t)
	s := &fakeSettingsReader{values: map[string]string{"observabilityBatchSize": "10"}}
	loader := NewObsConfigLoader(s, func(string) string { return "" }, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) })
	w := NewDetailWriter(st, loader, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }, nil, rand.Read)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err != nil {
				t.Errorf("Save: %v", err)
			}
		}()
	}
	wg.Wait()
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	rows, _, err := st.QueryRequestDetails(store.RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 50 {
		t.Fatalf("len(rows) = %d, want 50", len(rows))
	}
}

func TestWriterSanitizesAndTruncates(t *testing.T) {
	st := newRealWriterStore(t)
	s := &fakeSettingsReader{values: map[string]string{
		"observabilityBatchSize":   "1",
		"observabilityMaxJsonSize": "1",
	}}
	loader := NewObsConfigLoader(s, func(string) string { return "" }, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) })
	w := NewDetailWriter(st, loader, func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }, nil, rand.Read)

	big := strings.Repeat("a", 2000)
	if err := w.Save(&RequestDetail{
		Model:    "gpt-4o",
		Response: map[string]any{"body": big},
		Request:  map[string]any{"headers": map[string]string{"Authorization": "secret", "Content-Type": "json"}},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rows, _, err := st.QueryRequestDetails(store.RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	var data map[string]any
	if err := json.Unmarshal(rows[0], &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req := data["request"].(map[string]any)
	headers := req["headers"].(map[string]any)
	if headers["Authorization"] != nil {
		t.Errorf("authorization not sanitized")
	}
	if headers["Content-Type"] != "json" {
		t.Errorf("content-type missing")
	}
	resp := data["response"].(map[string]any)
	if resp["_truncated"] != true {
		t.Error("response not truncated")
	}
}

func TestWriterFlushErrorPropagates(t *testing.T) {
	st, loader, clock, tf, _, _ := newTestWriterDeps()
	loader.settings = &fakeSettingsReader{values: map[string]string{"observabilityBatchSize": "1"}}
	st.err = errors.New("flush failed")
	w := NewDetailWriter(st, loader, clock, tf, rand.Read)

	if err := w.Save(&RequestDetail{Model: "gpt-4o"}); err == nil {
		t.Fatal("expected flush error")
	}
}
