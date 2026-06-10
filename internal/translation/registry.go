package translation

import "fmt"

// RequestTranslator converts a source-format request body into an OpenAI-
// shaped request body (or the target format in a two-step pipeline).
type RequestTranslator func(model string, body map[string]any, stream bool) (map[string]any, error)

// ResponseTranslator converts an OpenAI-shaped stream chunk into the target
// format. It may return zero, one, or many chunks per input chunk.
type ResponseTranslator func(chunk map[string]any, state *StreamState) ([]map[string]any, error)

// StreamState holds mutable state for a streaming response translator across
// chunks (message IDs, block indices, tool-call buffers, etc.).
type StreamState struct {
	MessageStartSent      bool
	MessageID             string
	Model                 string
	NextBlockIndex        int
	TextBlockStarted      bool
	TextBlockIndex        int
	TextBlockClosed       bool
	ThinkingBlockStarted  bool
	ThinkingBlockIndex    int
	ToolCalls             map[int]toolCallInfo
	ToolArgBuffers        map[int]string
	FinishReason          string
	FinishReasonSent      bool
	Usage                 map[string]any
	ContentBlockIndex     int
}

type toolCallInfo struct {
	ID         string
	Name       string
	BlockIndex int
}

// Registry maps from:to format pairs to their translators.
type Registry struct {
	request  map[string]RequestTranslator
	response map[string]ResponseTranslator
}

// NewRegistry creates a registry with all Wave-1 translators wired.
func NewRegistry() *Registry {
	r := &Registry{
		request:  make(map[string]RequestTranslator),
		response: make(map[string]ResponseTranslator),
	}
	return r
}

// Register adds a request translator, response translator, or both for the
// given from:to format pair.
func (r *Registry) Register(from, to Format, req RequestTranslator, resp ResponseTranslator) {
	key := fmt.Sprintf("%s:%s", from, to)
	if req != nil {
		r.request[key] = req
	}
	if resp != nil {
		r.response[key] = resp
	}
}

// RequestTranslatorFor returns the registered request translator for from:to,
// or nil if none is registered.
func (r *Registry) RequestTranslatorFor(from, to Format) RequestTranslator {
	return r.request[fmt.Sprintf("%s:%s", from, to)]
}

// ResponseTranslatorFor returns the registered response translator for from:to,
// or nil if none is registered.
func (r *Registry) ResponseTranslatorFor(from, to Format) ResponseTranslator {
	return r.response[fmt.Sprintf("%s:%s", from, to)]
}

// NeedsTranslation reports whether the two formats differ.
func (r *Registry) NeedsTranslation(from, to Format) bool {
	return from != to
}
