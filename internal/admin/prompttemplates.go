package admin

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type promptDTO struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	SystemPrompt string   `json:"system_prompt"`
	Models       []string `json:"models"`
	IsActive     bool     `json:"is_active"`
	CreatedAt    string   `json:"created_at"`
}

func toPromptDTO(p *store.PromptTemplate) promptDTO {
	models := p.Models
	if models == nil {
		models = []string{}
	}
	return promptDTO{
		ID:           p.ID,
		Name:         p.Name,
		SystemPrompt: p.SystemPrompt,
		Models:       models,
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt,
	}
}

type promptRequest struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"system_prompt"`
	Models       []string `json:"models"`
	IsActive     *bool    `json:"is_active"`
}

func (r *promptRequest) isActive() bool {
	if r.IsActive == nil {
		return true
	}
	return *r.IsActive
}

// ListPromptTemplates handles GET /api/prompt-templates. Data is a bare array.
func (h *Handlers) ListPromptTemplates(ctx *fasthttp.RequestCtx) {
	templates, err := h.store.ListPromptTemplates()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list prompt templates")
		return
	}
	out := make([]promptDTO, 0, len(templates))
	for _, p := range templates {
		out = append(out, toPromptDTO(p))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreatePromptTemplate handles POST /api/prompt-templates.
func (h *Handlers) CreatePromptTemplate(ctx *fasthttp.RequestCtx) {
	var req promptRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}

	created, err := h.store.CreatePromptTemplate(&store.PromptTemplate{
		Name:         req.Name,
		SystemPrompt: req.SystemPrompt,
		Models:       req.Models,
		IsActive:     req.isActive(),
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create prompt template")
		return
	}
	h.recordAudit(ctx, "prompt_template.create", created.Name, "Created prompt template "+created.Name)
	writeData(ctx, fasthttp.StatusCreated, toPromptDTO(created))
}

// GetPromptTemplate handles GET /api/prompt-templates/{id}.
func (h *Handlers) GetPromptTemplate(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	p, err := h.store.GetPromptTemplateByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "prompt template not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load prompt template")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toPromptDTO(p))
}

// UpdatePromptTemplate handles PUT /api/prompt-templates/{id}.
func (h *Handlers) UpdatePromptTemplate(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req promptRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}

	updated, err := h.store.UpdatePromptTemplate(id, &store.PromptTemplate{
		Name:         req.Name,
		SystemPrompt: req.SystemPrompt,
		Models:       req.Models,
		IsActive:     req.isActive(),
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "prompt template not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update prompt template")
		return
	}
	h.recordAudit(ctx, "prompt_template.update", updated.Name, "Updated prompt template "+updated.Name)
	writeData(ctx, fasthttp.StatusOK, toPromptDTO(updated))
}

// DeletePromptTemplate handles DELETE /api/prompt-templates/{id}.
func (h *Handlers) DeletePromptTemplate(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetPromptTemplateByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "prompt template not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load prompt template")
		return
	}
	if err := h.store.DeletePromptTemplate(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "prompt template not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete prompt template")
		return
	}
	h.recordAudit(ctx, "prompt_template.delete", existing.Name, "Deleted prompt template "+existing.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Prompt template deleted successfully"})
}

type testPromptRequest struct {
	PromptID     int64  `json:"prompt_id"`
	SystemPrompt string `json:"system_prompt"`
	Sample       string `json:"sample"`
}

// TestPromptTemplate handles POST /api/prompt-templates/test. It renders a
// resolved system prompt against a sample input without calling any LLM: when
// prompt_id is set the stored template's system_prompt is used, otherwise the
// inline system_prompt. The result is a deterministic preview string.
func (h *Handlers) TestPromptTemplate(ctx *fasthttp.RequestCtx) {
	var req testPromptRequest
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	systemPrompt := req.SystemPrompt
	if req.PromptID > 0 {
		p, err := h.store.GetPromptTemplateByID(req.PromptID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(ctx, fasthttp.StatusNotFound, "prompt template not found")
			return
		}
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "load prompt template")
			return
		}
		systemPrompt = p.SystemPrompt
	}

	rendered := renderPrompt(systemPrompt, req.Sample)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"rendered": rendered})
}

// renderPrompt produces a deterministic preview by composing the resolved
// system prompt with the sample input. No external dependency or LLM call.
func renderPrompt(systemPrompt, sample string) string {
	var b strings.Builder
	b.WriteString("System: ")
	b.WriteString(systemPrompt)
	if sample != "" {
		b.WriteString("\n\nUser: ")
		b.WriteString(sample)
	}
	return b.String()
}
