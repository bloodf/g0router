package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// SemanticCacheEntry is one row of the exact-key-hash semantic cache. ExpiresAt
// is an ISO-8601 (RFC3339) string; an empty value means the row never expires
// (stored as SQL NULL). EmbeddingJSON is retained for forward-compatibility
// with the deferred semantic-similarity half — the hash-only cache writes "[]".
type SemanticCacheEntry struct {
	ID            int64
	CacheKey      string
	EmbeddingJSON string
	Model         string
	ResponseJSON  string
	ExpiresAt     string // ISO-8601; "" == no expiry (NULL)
	HitCount      int64
	CreatedAt     string
}

// SemanticCacheStatsResult summarizes the cache for the admin GET (counts only,
// never full responses).
type SemanticCacheStatsResult struct {
	Entries   int64
	TotalHits int64
}

// expiresArg maps an ISO expiry string to a nullable SQL argument: "" → NULL.
func expiresArg(iso string) any {
	if iso == "" {
		return nil
	}
	return iso
}

// GetSemanticCacheByKey returns the non-expired entry for cacheKey. nowISO is
// the injected clock value (D7/D9): a row is served only when expires_at IS NULL
// or expires_at > nowISO. Returns ErrNotFound on a miss or an expired-only row.
func (s *Store) GetSemanticCacheByKey(cacheKey, nowISO string) (*SemanticCacheEntry, error) {
	row := s.db.QueryRow(
		`SELECT id, cache_key, embedding_json, model, response_json,
		        COALESCE(expires_at, ''), COALESCE(hit_count, 0), COALESCE(created_at, '')
		 FROM semantic_cache
		 WHERE cache_key = ? AND (expires_at IS NULL OR expires_at > ?)
		 ORDER BY id DESC LIMIT 1`,
		cacheKey, nowISO,
	)
	return scanSemanticCacheEntry(row)
}

// InsertSemanticCacheEntry writes a cache row (write-through on a miss).
func (s *Store) InsertSemanticCacheEntry(e SemanticCacheEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO semantic_cache (cache_key, embedding_json, model, response_json, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		e.CacheKey, e.EmbeddingJSON, e.Model, e.ResponseJSON, expiresArg(e.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("insert semantic cache entry: %w", err)
	}
	return nil
}

// IncrementSemanticCacheHit bumps hit_count for the row id on a cache hit.
func (s *Store) IncrementSemanticCacheHit(id int64) error {
	if _, err := s.db.Exec(
		"UPDATE semantic_cache SET hit_count = hit_count + 1 WHERE id = ?", id,
	); err != nil {
		return fmt.Errorf("increment semantic cache hit %d: %w", id, err)
	}
	return nil
}

// PurgeExpiredSemanticCache deletes rows whose expires_at is non-NULL and at or
// before nowISO. It is called opportunistically on the read path (lazy purge,
// D7) — there is no background reaper. Returns the number of rows removed.
func (s *Store) PurgeExpiredSemanticCache(nowISO string) (int64, error) {
	res, err := s.db.Exec(
		"DELETE FROM semantic_cache WHERE expires_at IS NOT NULL AND expires_at <= ?", nowISO,
	)
	if err != nil {
		return 0, fmt.Errorf("purge expired semantic cache: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return n, nil
}

// ListSemanticCacheEntries returns all rows ordered by id ascending (admin GET).
func (s *Store) ListSemanticCacheEntries() ([]*SemanticCacheEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, cache_key, embedding_json, model, response_json,
		        COALESCE(expires_at, ''), COALESCE(hit_count, 0), COALESCE(created_at, '')
		 FROM semantic_cache ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query semantic cache entries: %w", err)
	}
	defer rows.Close()

	var out []*SemanticCacheEntry
	for rows.Next() {
		e, err := scanSemanticCacheEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semantic cache entries: %w", err)
	}
	return out, nil
}

// SemanticCacheStats returns the row count and total hit count (admin GET).
func (s *Store) SemanticCacheStats() (SemanticCacheStatsResult, error) {
	var res SemanticCacheStatsResult
	err := s.db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(hit_count), 0) FROM semantic_cache",
	).Scan(&res.Entries, &res.TotalHits)
	if err != nil {
		return SemanticCacheStatsResult{}, fmt.Errorf("semantic cache stats: %w", err)
	}
	return res, nil
}

// ClearSemanticCache removes all cache rows (admin DELETE).
func (s *Store) ClearSemanticCache() error {
	if _, err := s.db.Exec("DELETE FROM semantic_cache"); err != nil {
		return fmt.Errorf("clear semantic cache: %w", err)
	}
	return nil
}

func scanSemanticCacheEntry(row interface {
	Scan(dest ...any) error
}) (*SemanticCacheEntry, error) {
	var e SemanticCacheEntry
	err := row.Scan(&e.ID, &e.CacheKey, &e.EmbeddingJSON, &e.Model, &e.ResponseJSON, &e.ExpiresAt, &e.HitCount, &e.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan semantic cache entry: %w", err)
	}
	return &e, nil
}
