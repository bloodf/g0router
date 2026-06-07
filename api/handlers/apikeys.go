package handlers

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type createAPIKeyRequest struct {
	Name             string   `json:"name"`
	ExpiresAt        *int64   `json:"expires_at"`
	Scopes           []string `json:"scopes"`
	RateLimitRPM     *int     `json:"rate_limit_rpm"`
	RateLimitTPM     *int     `json:"rate_limit_tpm"`
	DailySpendCapUSD *float64 `json:"daily_spend_cap_usd"`
}

// apiKeyView is the public representation of a key. It never exposes the secret
// or key hash.
type apiKeyView struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Prefix           string   `json:"prefix"`
	IsActive         bool     `json:"is_active"`
	LastUsedAt       *string  `json:"last_used_at"`
	CreatedAt        string   `json:"created_at"`
	ExpiresAt        *int64   `json:"expires_at"`
	Scopes           []string `json:"scopes"`
	RateLimitRPM     *int     `json:"rate_limit_rpm"`
	RateLimitTPM     *int     `json:"rate_limit_tpm"`
	DailySpendCapUSD *float64 `json:"daily_spend_cap_usd"`
}

func newAPIKeyView(key store.APIKey) apiKeyView {
	return apiKeyView{
		ID:               key.ID,
		Name:             key.Name,
		Prefix:           key.Prefix,
		IsActive:         key.IsActive,
		LastUsedAt:       key.LastUsedAt,
		CreatedAt:        key.CreatedAt,
		ExpiresAt:        key.ExpiresAt,
		Scopes:           key.Scopes,
		RateLimitRPM:     key.RateLimitRPM,
		RateLimitTPM:     key.RateLimitTPM,
		DailySpendCapUSD: key.DailySpendCapUSD,
	}
}

type createAPIKeyResponse struct {
	Key apiKeyView `json:"key"`
	Raw string     `json:"raw"`
}

type updateAPIKeyPolicyRequest struct {
	Name             string   `json:"name"`
	ExpiresAt        *int64   `json:"expires_at"`
	Scopes           []string `json:"scopes"`
	RateLimitRPM     *int     `json:"rate_limit_rpm"`
	RateLimitTPM     *int     `json:"rate_limit_tpm"`
	DailySpendCapUSD *float64 `json:"daily_spend_cap_usd"`
}

func (r updateAPIKeyPolicyRequest) toPolicy() store.APIKeyPolicy {
	return store.APIKeyPolicy{
		ExpiresAt:        r.ExpiresAt,
		Scopes:           r.Scopes,
		RateLimitRPM:     r.RateLimitRPM,
		RateLimitTPM:     r.RateLimitTPM,
		DailySpendCapUSD: r.DailySpendCapUSD,
	}
}

type apiKeyStore interface {
	ListAPIKeys() ([]store.APIKey, error)
	CreateAPIKey(string, string) (*store.APIKey, string, error)
	UpdateAPIKeyPolicy(string, store.APIKeyPolicy) error
	RenameAPIKey(id string, name string) error
	GetAPIKey(string) (*store.APIKey, error)
	DeleteAPIKey(string) error
	RegenerateAPIKey(id string, secret string) (*store.APIKey, string, error)
}

func APIKeys(ctx *fasthttp.RequestCtx, s apiKeyStore, secret, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		keys, err := s.ListAPIKeys()
		if err != nil {
			log.Printf("list api keys: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to list api keys")
			return
		}
		views := make([]apiKeyView, 0, len(keys))
		for _, key := range keys {
			views = append(views, newAPIKeyView(key))
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[apiKeyView]{Data: views})
	case fasthttp.MethodPost:
		var req createAPIKeyRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		key, raw, err := s.CreateAPIKey(req.Name, secret)
		if err != nil {
			log.Printf("create api key: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create api key")
			return
		}
		policy := store.APIKeyPolicy{
			ExpiresAt:        req.ExpiresAt,
			Scopes:           req.Scopes,
			RateLimitRPM:     req.RateLimitRPM,
			RateLimitTPM:     req.RateLimitTPM,
			DailySpendCapUSD: req.DailySpendCapUSD,
		}
		if hasPolicy(policy) {
			if err := s.UpdateAPIKeyPolicy(key.ID, policy); err != nil {
				if errors.Is(err, store.ErrInvalidPolicy) {
					writeError(ctx, fasthttp.StatusBadRequest, err.Error())
					return
				}
				log.Printf("update api key policy: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to update api key policy")
				return
			}
			updated, err := s.GetAPIKey(key.ID)
			if err != nil {
				log.Printf("reload api key: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to load api key")
				return
			}
			key = updated
		}
		writeJSON(ctx, fasthttp.StatusCreated, createAPIKeyResponse{Key: newAPIKeyView(*key), Raw: raw})
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "api key id required")
			return
		}
		var req updateAPIKeyPolicyRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name != "" {
			if err := s.RenameAPIKey(id, req.Name); err != nil {
				log.Printf("rename api key: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to rename api key")
				return
			}
		}
		if err := s.UpdateAPIKeyPolicy(id, req.toPolicy()); err != nil {
			if errors.Is(err, store.ErrInvalidPolicy) {
				writeError(ctx, fasthttp.StatusBadRequest, err.Error())
				return
			}
			log.Printf("update api key policy: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update api key policy")
			return
		}
		updated, err := s.GetAPIKey(id)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "api key not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, createAPIKeyResponse{Key: newAPIKeyView(*updated)})
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "api key id required")
			return
		}
		if err := s.DeleteAPIKey(id); err != nil {
			log.Printf("delete api key: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete api key")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func hasPolicy(policy store.APIKeyPolicy) bool {
	return policy.ExpiresAt != nil || len(policy.Scopes) > 0 ||
		policy.RateLimitRPM != nil || policy.RateLimitTPM != nil ||
		policy.DailySpendCapUSD != nil
}

func RegenerateAPIKey(ctx *fasthttp.RequestCtx, s apiKeyStore, secret, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "api key id required")
		return
	}
	key, raw, err := s.RegenerateAPIKey(id, secret)
	if err != nil {
		log.Printf("regenerate api key: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to regenerate api key")
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, createAPIKeyResponse{Key: newAPIKeyView(*key), Raw: raw})
}
