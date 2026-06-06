package semcache

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// Entry is a single cached response with optional semantic embedding.
type Entry struct {
	ID            int64
	CacheKey      string
	EmbeddingJSON string
	Model         string
	ResponseJSON  string
	ExpiresAt     *time.Time
	HitCount      int
	CreatedAt     time.Time
}

// CachedResponse is the shape of an OpenAI chat completion response.
type CachedResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Embedder generates a vector embedding for a given prompt text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// Repository persists semantic cache entries.
type Repository interface {
	GetByKey(key, model string) (*Entry, error)
	ListCandidates(model string, limit int) ([]Entry, error)
	Store(entry *Entry) error
	IncrementHit(id int64) error
	ClearAll() error
	Stats() (count int, totalHits int64, err error)
}

// Cache provides exact-key and semantic similarity lookup.
type Cache struct {
	repo      Repository
	embedder  Embedder
	threshold float64
}

// NewCache creates a Cache with the given repository and threshold.
func NewCache(repo Repository, embedder Embedder, threshold float64) *Cache {
	if threshold <= 0 {
		threshold = 0.95
	}
	return &Cache{
		repo:      repo,
		embedder:  embedder,
		threshold: threshold,
	}
}

// Lookup attempts an exact key match first, then falls back to semantic similarity.
func (c *Cache) Lookup(ctx context.Context, key, model string, promptFn func() string) (*CachedResponse, bool, error) {
	entry, err := c.repo.GetByKey(key, model)
	if err != nil {
		return nil, false, fmt.Errorf("semcache get by key: %w", err)
	}
	if entry != nil && !isExpired(entry.ExpiresAt) {
		if err := c.repo.IncrementHit(entry.ID); err != nil {
			return nil, false, fmt.Errorf("semcache increment hit: %w", err)
		}
		resp, err := unmarshalResponse(entry.ResponseJSON)
		if err != nil {
			return nil, false, err
		}
		return resp, true, nil
	}

	if c.embedder == nil || promptFn == nil {
		return nil, false, nil
	}

	vec, err := c.embedder.Embed(ctx, promptFn())
	if err != nil {
		return nil, false, nil // silently miss on embedder error
	}

	candidates, err := c.repo.ListCandidates(model, 500)
	if err != nil {
		return nil, false, fmt.Errorf("semcache list candidates: %w", err)
	}

	var best *Entry
	bestSim := -1.0
	for i := range candidates {
		cand := &candidates[i]
		if isExpired(cand.ExpiresAt) {
			continue
		}
		candVec, err := unmarshalEmbedding(cand.EmbeddingJSON)
		if err != nil || len(candVec) == 0 {
			continue
		}
		sim := cosineSimilarity(vec, candVec)
		if sim > bestSim {
			bestSim = sim
			best = cand
		}
	}

	if best == nil || !meetsThreshold(bestSim, c.threshold) {
		return nil, false, nil
	}

	if err := c.repo.IncrementHit(best.ID); err != nil {
		return nil, false, fmt.Errorf("semcache increment hit: %w", err)
	}
	resp, err := unmarshalResponse(best.ResponseJSON)
	if err != nil {
		return nil, false, err
	}
	return resp, true, nil
}

// Store persists a response. If an embedder is available, it also stores the embedding.
func (c *Cache) Store(ctx context.Context, key, model, prompt string, resp *CachedResponse, ttl time.Duration) error {
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	entry := &Entry{
		CacheKey:     key,
		Model:        model,
		ResponseJSON: string(respJSON),
	}
	if ttl > 0 {
		tm := time.Now().Add(ttl)
		entry.ExpiresAt = &tm
	}

	if c.embedder != nil {
		vec, err := c.embedder.Embed(ctx, prompt)
		if err == nil && len(vec) > 0 {
			embJSON, _ := json.Marshal(vec)
			entry.EmbeddingJSON = string(embJSON)
		}
	}

	if err := c.repo.Store(entry); err != nil {
		return fmt.Errorf("semcache store: %w", err)
	}
	return nil
}

// Clear removes all cached entries.
func (c *Cache) Clear() error {
	return c.repo.ClearAll()
}

// Stats returns the number of entries and total hits.
func (c *Cache) Stats() (int, int64, error) {
	return c.repo.Stats()
}

func isExpired(t *time.Time) bool {
	if t == nil {
		return false
	}
	return time.Now().After(*t)
}

func meetsThreshold(sim, threshold float64) bool {
	return sim >= threshold-1e-9
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func unmarshalResponse(data string) (*CachedResponse, error) {
	var resp CachedResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal cached response: %w", err)
	}
	return &resp, nil
}

func unmarshalEmbedding(data string) ([]float64, error) {
	if data == "" {
		return nil, nil
	}
	var vec []float64
	if err := json.Unmarshal([]byte(data), &vec); err != nil {
		return nil, err
	}
	return vec, nil
}
