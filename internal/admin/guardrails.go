package admin

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type guardrailsConfigDTO struct {
	Enabled             bool     `json:"guardrails_enabled"`
	Blocklist           []string `json:"guardrails_blocklist"`
	PIIRedactionEnabled bool     `json:"pii_redaction_enabled"`
	PIIRedactionTypes   []string `json:"pii_redaction_types"`
}

func toGuardrailsDTO(g *store.Guardrails) guardrailsConfigDTO {
	blocklist := g.Blocklist
	if blocklist == nil {
		blocklist = []string{}
	}
	types := g.PIIRedactionTypes
	if types == nil {
		types = []string{}
	}
	return guardrailsConfigDTO{
		Enabled:             g.Enabled,
		Blocklist:           blocklist,
		PIIRedactionEnabled: g.PIIRedactionEnabled,
		PIIRedactionTypes:   types,
	}
}

// guardrailEngine builds the guardrails domain engine over the store. No New()
// signature change and no new global state (the auditService accessor precedent).
func (h *Handlers) guardrailEngine() *governance.GuardrailEngine {
	return governance.NewGuardrailEngine(h.store)
}

// GetGuardrails handles GET /api/guardrails.
func (h *Handlers) GetGuardrails(ctx *fasthttp.RequestCtx) {
	cfg, err := h.guardrailEngine().Config()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load guardrails")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toGuardrailsDTO(cfg))
}

// UpdateGuardrails handles PUT /api/guardrails. The body may be partial; fields
// merge over the current config (mirroring the mock's spread semantics).
func (h *Handlers) UpdateGuardrails(ctx *fasthttp.RequestCtx) {
	engine := h.guardrailEngine()
	cfg, err := engine.Config()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load guardrails")
		return
	}

	var patch struct {
		Enabled             *bool     `json:"guardrails_enabled"`
		Blocklist           *[]string `json:"guardrails_blocklist"`
		PIIRedactionEnabled *bool     `json:"pii_redaction_enabled"`
		PIIRedactionTypes   *[]string `json:"pii_redaction_types"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &patch); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if patch.Enabled != nil {
		cfg.Enabled = *patch.Enabled
	}
	if patch.Blocklist != nil {
		cfg.Blocklist = *patch.Blocklist
	}
	if patch.PIIRedactionEnabled != nil {
		cfg.PIIRedactionEnabled = *patch.PIIRedactionEnabled
	}
	if patch.PIIRedactionTypes != nil {
		cfg.PIIRedactionTypes = *patch.PIIRedactionTypes
	}

	if err := engine.Save(cfg); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "save guardrails")
		return
	}
	h.recordAudit(ctx, "guardrails.update", "guardrails",
		fmt.Sprintf("enabled=%v blocklist=%d", cfg.Enabled, len(cfg.Blocklist)))
	writeData(ctx, fasthttp.StatusOK, toGuardrailsDTO(cfg))
}

type testGuardrailsRequest struct {
	Prompt string `json:"prompt"`
}

// TestGuardrails handles POST /api/guardrails/test. It evaluates the prompt
// against the stored config without any LLM call and returns the deterministic
// {blocked, redacted_prompt, matches} result.
func (h *Handlers) TestGuardrails(ctx *fasthttp.RequestCtx) {
	var req testGuardrailsRequest
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	engine := h.guardrailEngine()
	cfg, err := engine.Config()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load guardrails")
		return
	}
	blocked, redacted, matches := engine.Evaluate(cfg, req.Prompt)
	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"blocked":         blocked,
		"redacted_prompt": redacted,
		"matches":         matches,
	})
}
