package semcache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	byKey            map[string]*Entry
	candidates       []Entry
	storeErr         error
	hitErr           error
	getByKeyErr      error
	listCandidatesErr error
}

func (f *fakeRepo) GetByKey(key, model string) (*Entry, error) {
	if f.getByKeyErr != nil {
		return nil, f.getByKeyErr
	}
	if e, ok := f.byKey[key+"#"+model]; ok {
		return e, nil
	}
	return nil, nil
}

func (f *fakeRepo) ListCandidates(model string, limit int) ([]Entry, error) {
	if f.listCandidatesErr != nil {
		return nil, f.listCandidatesErr
	}
	return f.candidates, nil
}

func (f *fakeRepo) Store(entry *Entry) error {
	if f.storeErr != nil {
		return f.storeErr
	}
	if f.byKey == nil {
		f.byKey = make(map[string]*Entry)
	}
	f.byKey[entry.CacheKey+"#"+entry.Model] = entry
	return nil
}

func (f *fakeRepo) IncrementHit(id int64) error {
	return f.hitErr
}

func (f *fakeRepo) ClearAll() error {
	f.byKey = nil
	f.candidates = nil
	return nil
}

func (f *fakeRepo) Stats() (int, int64, error) {
	return len(f.byKey), 0, nil
}

type fakeEmbedder struct {
	vec []float64
	err error
}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return f.vec, f.err
}

func TestLookupExactKeyHit(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{
			"key#model": {
				ID:           1,
				CacheKey:     "key",
				Model:        "model",
				ResponseJSON: `{"id":"resp1"}`,
				ExpiresAt:    timePtr(time.Now().Add(time.Hour)),
			},
		},
	}
	c := NewCache(repo, nil, 0.95)

	resp, ok, err := c.Lookup(context.Background(), "key", "model", nil)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if resp == nil || resp.ID != "resp1" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestLookupExactKeyExpired(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{
			"key#model": {
				ID:           1,
				CacheKey:     "key",
				Model:        "model",
				ResponseJSON: `{"id":"resp1"}`,
				ExpiresAt:    timePtr(time.Now().Add(-time.Hour)),
			},
		},
	}
	c := NewCache(repo, nil, 0.95)

	_, ok, err := c.Lookup(context.Background(), "key", "model", nil)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss (expired)")
	}
}

func TestLookupSemanticMatch(t *testing.T) {
	repo := &fakeRepo{
		candidates: []Entry{
			{
				ID:            1,
				CacheKey:      "other",
				Model:         "model",
				EmbeddingJSON: `[1.0, 0.0]`,
				ResponseJSON:  `{"id":"resp1"}`,
				ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
			},
		},
	}
	embedder := &fakeEmbedder{vec: []float64{1.0, 0.0}}
	c := NewCache(repo, embedder, 0.95)

	resp, ok, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !ok {
		t.Fatal("expected semantic cache hit")
	}
	if resp == nil || resp.ID != "resp1" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestLookupSemanticNoMatch(t *testing.T) {
	repo := &fakeRepo{
		candidates: []Entry{
			{
				ID:            1,
				CacheKey:      "other",
				Model:         "model",
				EmbeddingJSON: `[0.0, 1.0]`,
				ResponseJSON:  `{"id":"resp1"}`,
				ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
			},
		},
	}
	embedder := &fakeEmbedder{vec: []float64{1.0, 0.0}}
	c := NewCache(repo, embedder, 0.95)

	_, ok, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss (orthogonal)")
	}
}

func TestLookupNoEmbedder(t *testing.T) {
	repo := &fakeRepo{}
	c := NewCache(repo, nil, 0.95)

	_, ok, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss (no embedder)")
	}
}

func TestLookupEmbedderError(t *testing.T) {
	repo := &fakeRepo{}
	embedder := &fakeEmbedder{err: errors.New("embed fail")}
	c := NewCache(repo, embedder, 0.95)

	_, ok, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss (embedder error)")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	repo := &fakeRepo{byKey: make(map[string]*Entry)}
	embedder := &fakeEmbedder{vec: []float64{1.0, 2.0}}
	c := NewCache(repo, embedder, 0.95)

	resp := &CachedResponse{ID: "resp1", Object: "chat.completion", Model: "model"}
	if err := c.Store(context.Background(), "key", "model", "prompt", resp, time.Hour); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, ok, err := c.Lookup(context.Background(), "key", "model", nil)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit after store")
	}
	if got.ID != "resp1" {
		t.Fatalf("id = %q, want resp1", got.ID)
	}
}

func TestStoreNoEmbedder(t *testing.T) {
	repo := &fakeRepo{byKey: make(map[string]*Entry)}
	c := NewCache(repo, nil, 0.95)

	resp := &CachedResponse{ID: "resp1"}
	err := c.Store(context.Background(), "key", "model", "prompt", resp, time.Hour)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
}

func TestStoreEmbedderError(t *testing.T) {
	repo := &fakeRepo{byKey: make(map[string]*Entry)}
	embedder := &fakeEmbedder{err: errors.New("embed fail")}
	c := NewCache(repo, embedder, 0.95)

	resp := &CachedResponse{ID: "resp1"}
	err := c.Store(context.Background(), "key", "model", "prompt", resp, time.Hour)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
}

func TestClear(t *testing.T) {
	repo := &fakeRepo{byKey: map[string]*Entry{"k": {}}}
	c := NewCache(repo, nil, 0.95)
	if err := c.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
}

func TestStats(t *testing.T) {
	repo := &fakeRepo{byKey: map[string]*Entry{"k": {}}}
	c := NewCache(repo, nil, 0.95)
	count, hits, err := c.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	_ = hits
}

func TestNewCacheDefaultThreshold(t *testing.T) {
	c := NewCache(&fakeRepo{}, nil, 0)
	if c.threshold != 0.95 {
		t.Fatalf("threshold = %v, want 0.95", c.threshold)
	}
}

func TestLookupGetByKeyError(t *testing.T) {
	repo := &fakeRepo{getByKeyErr: errors.New("db fail")}
	c := NewCache(repo, nil, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLookupListCandidatesError(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{}, // no exact hit
		candidates: []Entry{},
	}
	c := NewCache(repo, &fakeEmbedder{vec: []float64{1, 0}}, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
}

func TestLookupIncrementHitError(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{
			"key#model": {
				ID:           1,
				CacheKey:     "key",
				Model:        "model",
				ResponseJSON: `{"id":"resp1"}`,
				ExpiresAt:    timePtr(time.Now().Add(time.Hour)),
			},
		},
		hitErr: errors.New("hit fail"),
	}
	c := NewCache(repo, nil, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", nil)
	if err == nil {
		t.Fatal("expected error from increment hit")
	}
}

func TestStoreRepoError(t *testing.T) {
	repo := &fakeRepo{storeErr: errors.New("store fail")}
	c := NewCache(repo, nil, 0.95)
	resp := &CachedResponse{ID: "resp1"}
	err := c.Store(context.Background(), "key", "model", "prompt", resp, time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsExpiredNil(t *testing.T) {
	if isExpired(nil) {
		t.Fatal("nil should not be expired")
	}
}

func TestUnmarshalResponseBadJSON(t *testing.T) {
	_, err := unmarshalResponse("not json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnmarshalEmbeddingEmpty(t *testing.T) {
	v, err := unmarshalEmbedding("")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if v != nil {
		t.Fatal("expected nil")
	}
}

func TestUnmarshalEmbeddingBadJSON(t *testing.T) {
	_, err := unmarshalEmbedding("not json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLookupExactKeyBadResponseJSON(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{
			"key#model": {
				ID:           1,
				CacheKey:     "key",
				Model:        "model",
				ResponseJSON: `not json`,
				ExpiresAt:    timePtr(time.Now().Add(time.Hour)),
			},
		},
	}
	c := NewCache(repo, nil, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", nil)
	if err == nil {
		t.Fatal("expected error from bad response JSON")
	}
}

func TestLookupListCandidatesReturnsError(t *testing.T) {
	repo := &fakeRepo{
		byKey:             map[string]*Entry{},
		listCandidatesErr: errors.New("db fail"),
	}
	c := NewCache(repo, &fakeEmbedder{vec: []float64{1, 0}}, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLookupSemanticCandidateExpired(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{},
		candidates: []Entry{
			{
				ID:            1,
				CacheKey:      "other",
				Model:         "model",
				EmbeddingJSON: `[1.0, 0.0]`,
				ResponseJSON:  `{"id":"resp1"}`,
				ExpiresAt:     timePtr(time.Now().Add(-time.Hour)),
			},
		},
	}
	embedder := &fakeEmbedder{vec: []float64{1.0, 0.0}}
	c := NewCache(repo, embedder, 0.95)
	_, ok, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ok {
		t.Fatal("expected miss (expired candidate)")
	}
}

func TestLookupSemanticCandidateBadEmbedding(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{},
		candidates: []Entry{
			{
				ID:            1,
				CacheKey:      "other",
				Model:         "model",
				EmbeddingJSON: `not json`,
				ResponseJSON:  `{"id":"resp1"}`,
				ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
			},
		},
	}
	embedder := &fakeEmbedder{vec: []float64{1.0, 0.0}}
	c := NewCache(repo, embedder, 0.95)
	_, ok, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ok {
		t.Fatal("expected miss (bad embedding)")
	}
}

func TestLookupSemanticIncrementHitError(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{},
		candidates: []Entry{
			{
				ID:            1,
				CacheKey:      "other",
				Model:         "model",
				EmbeddingJSON: `[1.0, 0.0]`,
				ResponseJSON:  `{"id":"resp1"}`,
				ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
			},
		},
		hitErr: errors.New("hit fail"),
	}
	embedder := &fakeEmbedder{vec: []float64{1.0, 0.0}}
	c := NewCache(repo, embedder, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err == nil {
		t.Fatal("expected error from increment hit")
	}
}

func TestLookupSemanticBadResponseJSON(t *testing.T) {
	repo := &fakeRepo{
		byKey: map[string]*Entry{},
		candidates: []Entry{
			{
				ID:            1,
				CacheKey:      "other",
				Model:         "model",
				EmbeddingJSON: `[1.0, 0.0]`,
				ResponseJSON:  `not json`,
				ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
			},
		},
	}
	embedder := &fakeEmbedder{vec: []float64{1.0, 0.0}}
	c := NewCache(repo, embedder, 0.95)
	_, _, err := c.Lookup(context.Background(), "key", "model", func() string { return "prompt" })
	if err == nil {
		t.Fatal("expected error from bad response JSON")
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
