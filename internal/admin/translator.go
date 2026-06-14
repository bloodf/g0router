package admin

import (
	"encoding/json"

	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// translatorSamplePayload is the in-handler sample client request the translator
// page loads into its first textarea. It mirrors the e2e mock's
// SAMPLE_CLIENT_REQUEST (translator.ts:13-18) and MUST contain "gpt-4o" so the
// frozen translator.spec assertion holds against real Go.
var translatorSamplePayload = map[string]any{
	"model":    "gpt-4o",
	"provider": "openai",
	"messages": []any{
		map[string]any{"role": "user", "content": "Translate this request"},
	},
	"stream": false,
}

// knownFormats maps the wire-format strings the translator accepts to their
// translation.Format. Only formats with registered converters are useful;
// unknown strings fall back to the empty Format (handled as a 400).
var knownFormats = map[string]translation.Format{
	"openai":           translation.FormatOpenAI,
	"openai-responses": translation.FormatOpenAIResponses,
	"openai-response":  translation.FormatOpenAIResponse,
	"claude":           translation.FormatClaude,
	"gemini":           translation.FormatGemini,
	"gemini-cli":       translation.FormatGeminiCLI,
	"vertex":           translation.FormatVertex,
	"codex":            translation.FormatCodex,
	"antigravity":      translation.FormatAntigravity,
	"kiro":             translation.FormatKiro,
	"cursor":           translation.FormatCursor,
	"ollama":           translation.FormatOllama,
	"commandcode":      translation.FormatCommandCode,
}

// parseTranslatorFormat resolves a format string, defaulting to def when empty.
func parseTranslatorFormat(s string, def translation.Format) (translation.Format, bool) {
	if s == "" {
		return def, true
	}
	f, ok := knownFormats[s]
	return f, ok
}

// TranslatorLoad handles GET /api/translator/load?file=<name>. It returns a
// sample client request payload (JSON-serialized) for the requested file. The
// payloads are self-contained constants (no store), mirroring the e2e mock.
func (h *Handlers) TranslatorLoad(ctx *fasthttp.RequestCtx) {
	file := string(ctx.QueryArgs().Peek("file"))
	if file == "" {
		file = "sample"
	}
	payload, err := json.MarshalIndent(translatorSamplePayload, "", "  ")
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "encode sample payload")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"file":    file,
		"payload": string(payload),
	})
}

// TranslatorTranslate handles POST /api/translator/translate. It parses the
// {from,to,model,payload} body, runs the translation registry's request
// pipeline, and returns the transformed payload marked translated:true. It is a
// pure body transform: no credentials, no network.
func (h *Handlers) TranslatorTranslate(ctx *fasthttp.RequestCtx) {
	var req struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Model   string `json:"model"`
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	from, ok := parseTranslatorFormat(req.From, translation.FormatOpenAI)
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "unknown source format")
		return
	}
	to, ok := parseTranslatorFormat(req.To, translation.FormatClaude)
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "unknown target format")
		return
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(req.Payload), &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "payload is not valid JSON")
		return
	}

	registry := translation.NewRegistry()
	transformed, err := registry.TranslateRequest(from, to, req.Model, body, false, nil)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "translate request failed")
		return
	}

	out, err := json.MarshalIndent(transformed, "", "  ")
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "encode translated payload")
		return
	}

	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"payload":    string(out),
		"translated": true,
		"from":       string(from),
		"to":         string(to),
	})
}
