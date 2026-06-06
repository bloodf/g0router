package handlers

import (
	"encoding/json"
	"log"

	"github.com/bloodf/g0router/internal/guardrails"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type guardrailsStore interface {
	GetGuardrailsConfig() (store.GuardrailsConfig, error)
	UpdateGuardrailsConfig(store.GuardrailsConfig) error
}

type guardrailsConfigView struct {
	GuardrailsEnabled   bool     `json:"guardrails_enabled"`
	GuardrailsBlocklist []string `json:"guardrails_blocklist"`
	PIIRedactionEnabled bool     `json:"pii_redaction_enabled"`
	PIIRedactionTypes   []string `json:"pii_redaction_types"`
}

func newGuardrailsConfigView(cfg store.GuardrailsConfig) guardrailsConfigView {
	return guardrailsConfigView{
		GuardrailsEnabled:   cfg.GuardrailsEnabled,
		GuardrailsBlocklist: cfg.GuardrailsBlocklist,
		PIIRedactionEnabled: cfg.PIIRedactionEnabled,
		PIIRedactionTypes:   cfg.PIIRedactionTypes,
	}
}

type guardrailsTestRequest struct {
	Prompt string `json:"prompt"`
}

type guardrailsTestResponse struct {
	Blocked        bool     `json:"blocked"`
	RedactedPrompt string   `json:"redacted_prompt"`
	Matches        []string `json:"matches"`
}

type updateGuardrailsRequest struct {
	GuardrailsEnabled   bool     `json:"guardrails_enabled"`
	GuardrailsBlocklist []string `json:"guardrails_blocklist"`
	PIIRedactionEnabled bool     `json:"pii_redaction_enabled"`
	PIIRedactionTypes   []string `json:"pii_redaction_types"`
}

func Guardrails(ctx *fasthttp.RequestCtx, s guardrailsStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		cfg, err := s.GetGuardrailsConfig()
		if err != nil {
			log.Printf("get guardrails config: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to get guardrails config")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newGuardrailsConfigView(cfg))
	case fasthttp.MethodPut:
		var req updateGuardrailsRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		cfg := store.GuardrailsConfig{
			GuardrailsEnabled:   req.GuardrailsEnabled,
			GuardrailsBlocklist: req.GuardrailsBlocklist,
			PIIRedactionEnabled: req.PIIRedactionEnabled,
			PIIRedactionTypes:   req.PIIRedactionTypes,
		}
		if err := s.UpdateGuardrailsConfig(cfg); err != nil {
			log.Printf("update guardrails config: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update guardrails config")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newGuardrailsConfigView(cfg))
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func GuardrailsTest(ctx *fasthttp.RequestCtx, s guardrailsStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	var req guardrailsTestRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	cfg, err := s.GetGuardrailsConfig()
	if err != nil {
		log.Printf("get guardrails config: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get guardrails config")
		return
	}

	grCfg := guardrails.Config{
		Enabled:   cfg.GuardrailsEnabled,
		Blocklist: cfg.GuardrailsBlocklist,
	}
	piiCfg := guardrails.PIIConfig{
		Enabled: cfg.PIIRedactionEnabled,
		Types:   cfg.PIIRedactionTypes,
	}

	blocked, matches := grCfg.CheckBlocklist(req.Prompt)
	redacted := piiCfg.Redact(req.Prompt)

	writeJSON(ctx, fasthttp.StatusOK, guardrailsTestResponse{
		Blocked:        blocked,
		RedactedPrompt: redacted,
		Matches:        matches,
	})
}
