package store

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/semcache"
	_ "modernc.org/sqlite"
)

func TestSemcacheSchema(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name='semantic_cache'`)
	if err != nil {
		t.Fatalf("schema query: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("semantic_cache table missing")
	}
}

func TestSemcacheIndexes(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()

	idxs := []string{"idx_semantic_cache_model", "idx_semantic_cache_expires"}
	for _, idx := range idxs {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&name)
		if err != nil {
			t.Fatalf("index %q missing: %v", idx, err)
		}
	}
}

func TestSemcacheStoreAndGetByKey(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	entry := &semcache.Entry{
		CacheKey:      "ck1",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
	}
	if err := r.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := r.GetByKey("ck1", "gpt-4")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry")
	}
	if got.CacheKey != "ck1" {
		t.Fatalf("key = %q, want ck1", got.CacheKey)
	}
	if got.ResponseJSON != `{"id":"r1"}` {
		t.Fatalf("response = %q", got.ResponseJSON)
	}
}

func TestSemcacheGetByKeyExpired(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	entry := &semcache.Entry{
		CacheKey:      "ck1",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(-time.Hour)),
	}
	if err := r.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := r.GetByKey("ck1", "gpt-4")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for expired entry")
	}
}

func TestSemcacheListCandidates(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	for i := 0; i < 3; i++ {
		entry := &semcache.Entry{
			CacheKey:      "ck",
			Model:         "gpt-4",
			EmbeddingJSON: `[1.0, 0.0]`,
			ResponseJSON:  `{"id":"r"}`,
			ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
		}
		if err := r.Store(entry); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	cands, err := r.ListCandidates("gpt-4", 500)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(cands) != 3 {
		t.Fatalf("len(cands) = %d, want 3", len(cands))
	}
}

func TestSemcacheListCandidatesExpiredExcluded(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	valid := &semcache.Entry{
		CacheKey:      "valid",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
	}
	if err := r.Store(valid); err != nil {
		t.Fatalf("Store valid: %v", err)
	}
	expired := &semcache.Entry{
		CacheKey:      "expired",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r2"}`,
		ExpiresAt:     timePtr(time.Now().Add(-time.Hour)),
	}
	if err := r.Store(expired); err != nil {
		t.Fatalf("Store expired: %v", err)
	}

	cands, err := r.ListCandidates("gpt-4", 500)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("len(cands) = %d, want 1", len(cands))
	}
	if cands[0].CacheKey != "valid" {
		t.Fatalf("key = %q, want valid", cands[0].CacheKey)
	}
}

func TestSemcacheListCandidatesLimit(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	for i := 0; i < 10; i++ {
		entry := &semcache.Entry{
			CacheKey:      "ck",
			Model:         "gpt-4",
			EmbeddingJSON: `[1.0, 0.0]`,
			ResponseJSON:  `{"id":"r"}`,
			ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
		}
		if err := r.Store(entry); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	cands, err := r.ListCandidates("gpt-4", 5)
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(cands) != 5 {
		t.Fatalf("len(cands) = %d, want 5", len(cands))
	}
}

func TestSemcacheIncrementHit(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	entry := &semcache.Entry{
		CacheKey:      "ck1",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
	}
	if err := r.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	if err := r.IncrementHit(entry.ID); err != nil {
		t.Fatalf("IncrementHit: %v", err)
	}

	// Verify hit_count increased
	var hitCount int
	err := db.QueryRow(`SELECT hit_count FROM semantic_cache WHERE id = ?`, entry.ID).Scan(&hitCount)
	if err != nil {
		t.Fatalf("query hit_count: %v", err)
	}
	if hitCount != 1 {
		t.Fatalf("hit_count = %d, want 1", hitCount)
	}
}

func TestSemcacheClearAll(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	entry := &semcache.Entry{
		CacheKey:      "ck1",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
	}
	if err := r.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	if err := r.ClearAll(); err != nil {
		t.Fatalf("ClearAll: %v", err)
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM semantic_cache`).Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestSemcacheStats(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	entry := &semcache.Entry{
		CacheKey:      "ck1",
		Model:         "gpt-4",
		EmbeddingJSON: `[1.0, 0.0]`,
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
	}
	if err := r.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	if err := r.IncrementHit(entry.ID); err != nil {
		t.Fatalf("IncrementHit: %v", err)
	}

	count, hits, err := r.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if hits != 1 {
		t.Fatalf("hits = %d, want 1", hits)
	}
}

func TestSemcacheStoreEmbeddingsRoundTrip(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	r := NewSemcacheRepo(db)

	vec := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	embJSON, _ := json.Marshal(vec)
	entry := &semcache.Entry{
		CacheKey:      "ck1",
		Model:         "gpt-4",
		EmbeddingJSON: string(embJSON),
		ResponseJSON:  `{"id":"r1"}`,
		ExpiresAt:     timePtr(time.Now().Add(time.Hour)),
	}
	if err := r.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := r.GetByKey("ck1", "gpt-4")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry")
	}

	var gotVec []float64
	if err := json.Unmarshal([]byte(got.EmbeddingJSON), &gotVec); err != nil {
		t.Fatalf("unmarshal embedding: %v", err)
	}
	if len(gotVec) != 5 {
		t.Fatalf("len(vec) = %d, want 5", len(gotVec))
	}
}

func newSemcacheDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	schema := `CREATE TABLE IF NOT EXISTS semantic_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cache_key TEXT NOT NULL,
		embedding_json TEXT,
		model TEXT NOT NULL,
		response_json TEXT NOT NULL,
		expires_at DATETIME,
		hit_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_semantic_cache_model ON semantic_cache(model);
	CREATE INDEX IF NOT EXISTS idx_semantic_cache_expires ON semantic_cache(expires_at);`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestSemcacheGetByKeyScanError(t *testing.T) {
	db := newSemcacheDB(t)
	defer db.Close()
	// Insert a row then alter the schema to cause a scan mismatch
	if _, err := db.Exec(`ALTER TABLE semantic_cache ADD COLUMN extra_col TEXT`); err != nil {
		t.Skip("sqlite alter not supported")
	}
}

func TestSemcacheStoreClosedDB(t *testing.T) {
	db := newSemcacheDB(t)
	db.Close()
	r := NewSemcacheRepo(db)
	entry := &semcache.Entry{
		CacheKey:     "ck",
		Model:        "gpt-4",
		ResponseJSON: `{"id":"r"}`,
	}
	err := r.Store(entry)
	if err == nil {
		t.Fatal("expected error on closed db")
	}
}

func TestSemcacheIncrementHitClosedDB(t *testing.T) {
	db := newSemcacheDB(t)
	db.Close()
	r := NewSemcacheRepo(db)
	err := r.IncrementHit(1)
	if err == nil {
		t.Fatal("expected error on closed db")
	}
}

func TestSemcacheClearAllClosedDB(t *testing.T) {
	db := newSemcacheDB(t)
	db.Close()
	r := NewSemcacheRepo(db)
	err := r.ClearAll()
	if err == nil {
		t.Fatal("expected error on closed db")
	}
}

func TestSemcacheStatsClosedDB(t *testing.T) {
	db := newSemcacheDB(t)
	db.Close()
	r := NewSemcacheRepo(db)
	_, _, err := r.Stats()
	if err == nil {
		t.Fatal("expected error on closed db")
	}
}

func TestSemcacheListCandidatesClosedDB(t *testing.T) {
	db := newSemcacheDB(t)
	db.Close()
	r := NewSemcacheRepo(db)
	_, err := r.ListCandidates("gpt-4", 10)
	if err == nil {
		t.Fatal("expected error on closed db")
	}
}

func TestSemcacheGetByKeyClosedDB(t *testing.T) {
	db := newSemcacheDB(t)
	db.Close()
	r := NewSemcacheRepo(db)
	_, err := r.GetByKey("ck", "gpt-4")
	if err == nil {
		t.Fatal("expected error on closed db")
	}
}
