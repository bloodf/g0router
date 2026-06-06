package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/semcache"
)

// SemcacheRepo is a SQLite-backed repository for semantic cache entries.
type SemcacheRepo struct {
	db *sql.DB
}

// NewSemcacheRepo creates a new SemcacheRepo. It ensures the table and indexes exist.
func NewSemcacheRepo(db *sql.DB) *SemcacheRepo {
	r := &SemcacheRepo{db: db}
	_ = r.migrate()
	return r
}

func (r *SemcacheRepo) migrate() error {
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
	if _, err := r.db.Exec(schema); err != nil {
		return fmt.Errorf("semcache migrate: %w", err)
	}
	return nil
}

// GetByKey returns the non-expired entry matching key and model, or nil.
func (r *SemcacheRepo) GetByKey(key, model string) (*semcache.Entry, error) {
	row := r.db.QueryRow(`
		SELECT id, cache_key, embedding_json, model, response_json, expires_at, hit_count, created_at
		FROM semantic_cache
		WHERE cache_key = ? AND model = ?
		ORDER BY id DESC
		LIMIT 1`, key, model)

	var e semcache.Entry
	var expires sql.NullTime
	err := row.Scan(&e.ID, &e.CacheKey, &e.EmbeddingJSON, &e.Model, &e.ResponseJSON, &expires, &e.HitCount, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("semcache get by key: %w", err)
	}
	if expires.Valid {
		e.ExpiresAt = &expires.Time
	}
	if e.ExpiresAt != nil && time.Now().After(*e.ExpiresAt) {
		return nil, nil
	}
	return &e, nil
}

// ListCandidates returns up to limit non-expired entries for the given model,
// ordered by most recently created first.
func (r *SemcacheRepo) ListCandidates(model string, limit int) ([]semcache.Entry, error) {
	rows, err := r.db.Query(`
		SELECT id, cache_key, embedding_json, model, response_json, expires_at, hit_count, created_at
		FROM semantic_cache
		WHERE model = ?
		ORDER BY id DESC
		LIMIT ?`, model, limit)
	if err != nil {
		return nil, fmt.Errorf("semcache list candidates: %w", err)
	}
	defer rows.Close()

	var out []semcache.Entry
	for rows.Next() {
		var e semcache.Entry
		var expires sql.NullTime
		if err := rows.Scan(&e.ID, &e.CacheKey, &e.EmbeddingJSON, &e.Model, &e.ResponseJSON, &expires, &e.HitCount, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("semcache scan candidate: %w", err)
		}
		if expires.Valid {
			e.ExpiresAt = &expires.Time
		}
		if e.ExpiresAt != nil && time.Now().After(*e.ExpiresAt) {
			continue
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Store inserts a new cache entry.
func (r *SemcacheRepo) Store(entry *semcache.Entry) error {
	res, err := r.db.Exec(`
		INSERT INTO semantic_cache (cache_key, embedding_json, model, response_json, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		entry.CacheKey, entry.EmbeddingJSON, entry.Model, entry.ResponseJSON, entry.ExpiresAt)
	if err != nil {
		return fmt.Errorf("semcache store: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("semcache last insert id: %w", err)
	}
	entry.ID = id
	return nil
}

// IncrementHit increases hit_count by 1 for the given entry.
func (r *SemcacheRepo) IncrementHit(id int64) error {
	_, err := r.db.Exec(`UPDATE semantic_cache SET hit_count = hit_count + 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("semcache increment hit: %w", err)
	}
	return nil
}

// ClearAll deletes all cache entries.
func (r *SemcacheRepo) ClearAll() error {
	_, err := r.db.Exec(`DELETE FROM semantic_cache`)
	if err != nil {
		return fmt.Errorf("semcache clear all: %w", err)
	}
	return nil
}

// Stats returns the total number of entries and cumulative hit_count.
func (r *SemcacheRepo) Stats() (int, int64, error) {
	var count int
	var totalHits sql.NullInt64
	err := r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(hit_count), 0)
		FROM semantic_cache`).Scan(&count, &totalHits)
	if err != nil {
		return 0, 0, fmt.Errorf("semcache stats: %w", err)
	}
	return count, totalHits.Int64, nil
}

// SemcacheRepo returns a repository backed by this store's database.
func (s *Store) SemcacheRepo() *SemcacheRepo {
	return NewSemcacheRepo(s.db)
}
