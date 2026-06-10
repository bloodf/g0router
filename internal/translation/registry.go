package translation

import "fmt"

// RequestTranslator converts a source-format request body into an OpenAI-
// shaped request body (or the target format in a two-step pipeline).
type RequestTranslator func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error)

// ResponseTranslator converts an OpenAI-shaped stream chunk into the target
// format. It may return zero, one, or many chunks per input chunk.
type ResponseTranslator func(chunk map[string]any, state *StreamState) ([]map[string]any, error)

// StreamState holds mutable state for a streaming response translator across
// chunks (message IDs, block indices, tool-call buffers, etc.).
type StreamState struct {
	MessageStartSent     bool
	MessageID            string
	Model                string
	NextBlockIndex       int
	TextBlockStarted     bool
	TextBlockIndex       int
	TextBlockClosed      bool
	ThinkingBlockStarted bool
	ThinkingBlockIndex   int
	ToolCalls            map[int]ToolCallInfo
	ToolArgBuffers       map[int]string
	FinishReason         string
	FinishReasonSent     bool
	Usage                map[string]any
	ContentBlockIndex    int
	// Fields for claude→openai response translation.
	FunctionIndex        int
	ToolCallIndex        int
	ServerToolBlockIndex int
	InThinkingBlock      bool
	CurrentBlockIndex    int
	ClaudeBlockTools     map[int]claudeOpenAIToolCall
	ToolNameMap          map[string]string
	// Fields for openai→antigravity response translation.
	AntigravityToolCallAccum map[int]map[string]any
	AntigravityResponseID    string
	AntigravityModelVersion  string
	AntigravityUsage         map[string]any
	// Fields for openai→responses response translation.
	ResponsesSeq             int
	ResponsesStarted         bool
	ResponsesID              string
	ResponsesCreated         int64
	ResponsesReasoningID     string
	ResponsesReasoningIndex  int
	ResponsesReasoningBuf    string
	ResponsesReasoningDone   bool
	ResponsesInThinking      bool
	ResponsesMsgItemAdded    map[int]bool
	ResponsesContentAdded    map[int]bool
	ResponsesItemDone        map[int]bool
	ResponsesMsgTextBuf      map[int]string
	ResponsesFuncCallIDs     map[int]string
	ResponsesFuncNames       map[int]string
	ResponsesFuncArgsBuf     map[int]string
	ResponsesFuncItemDone    map[int]bool
	ResponsesFuncArgsDone    map[int]bool
	ResponsesCompletedSent   bool
	// Fields for responses→openai response translation.
	ResponsesChatID          string
	ResponsesToolCallIndex   int
	ResponsesCurrentToolCallID string
}

// claudeOpenAIToolCall tracks an in-flight Claude tool_use block during
// claude→openai response translation.
type claudeOpenAIToolCall struct {
	Index     int
	ID        string
	Name      string
	Arguments string
}

// NewStreamState creates a zero-valued StreamState with initialized maps.
func NewStreamState() *StreamState {
	return &StreamState{
		ToolCalls:                make(map[int]ToolCallInfo),
		ToolArgBuffers:           make(map[int]string),
		ClaudeBlockTools:         make(map[int]claudeOpenAIToolCall),
		ServerToolBlockIndex:     -1,
		AntigravityToolCallAccum: make(map[int]map[string]any),
		ResponsesMsgItemAdded:    make(map[int]bool),
		ResponsesContentAdded:    make(map[int]bool),
		ResponsesItemDone:        make(map[int]bool),
		ResponsesMsgTextBuf:      make(map[int]string),
		ResponsesFuncCallIDs:     make(map[int]string),
		ResponsesFuncNames:       make(map[int]string),
		ResponsesFuncArgsBuf:     make(map[int]string),
		ResponsesFuncItemDone:    make(map[int]bool),
		ResponsesFuncArgsDone:    make(map[int]bool),
	}
}

// ToolCallInfo tracks a tool call in flight during response translation.
type ToolCallInfo struct {
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
	r.Register(FormatClaude, FormatOpenAI, claudeToOpenAIRequest, claudeToOpenAIResponse)
	r.Register(FormatOpenAI, FormatClaude, openaiToClaudeRequest, openaiToClaudeResponse)
	r.Register(FormatOpenAI, FormatGemini, openaiToGeminiRequest, nil)
	r.Register(FormatGemini, FormatOpenAI, nil, geminiToOpenAIResponse)
	r.Register(FormatOpenAI, FormatGeminiCLI, func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
		gemini, err := openaiToGeminiCLIRequest(model, body, stream, credentials)
		if err != nil {
			return nil, fmt.Errorf("openai->gemini-cli request: %w", err)
		}
		env, err := wrapInCloudCodeEnvelope(model, gemini, credentials, false)
		if err != nil {
			return nil, fmt.Errorf("openai->gemini-cli envelope: %w", err)
		}
		return env, nil
	}, nil)
	r.Register(FormatOpenAI, FormatAntigravity, openaiToAntigravityRequest, nil)
	r.Register(FormatAntigravity, FormatOpenAI, antigravityToOpenAIRequest, nil)
	r.Register(FormatOpenAI, FormatVertex, openaiToVertexRequest, nil)
	r.Register(FormatOpenAI, FormatAntigravity, nil, openaiToAntigravityResponse)
	r.Register(FormatGeminiCLI, FormatOpenAI, nil, geminiToOpenAIResponse)
	r.Register(FormatVertex, FormatOpenAI, nil, geminiToOpenAIResponse)
	r.Register(FormatAntigravity, FormatOpenAI, nil, geminiToOpenAIResponse)
	r.Register(FormatOpenAIResponses, FormatOpenAI, responsesToOpenAIRequest, responsesToOpenAIResponse)
	r.Register(FormatOpenAI, FormatOpenAIResponses, openaiToResponsesRequest, openaiToResponsesResponse)
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

// TranslateRequest runs the source -> openai -> target pipeline.
func (r *Registry) TranslateRequest(from, to Format, model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	result := body
	if !r.NeedsTranslation(from, to) {
		return result, nil
	}

	// Step 1: source -> openai (if source is not openai).
	if from != FormatOpenAI {
		fn := r.RequestTranslatorFor(from, FormatOpenAI)
		if fn != nil {
			var err error
			result, err = fn(model, result, stream, credentials)
			if err != nil {
				return nil, fmt.Errorf("translate %s->openai: %w", from, err)
			}
		}
	}

	// Step 2: openai -> target (if target is not openai).
	if to != FormatOpenAI {
		fn := r.RequestTranslatorFor(FormatOpenAI, to)
		if fn != nil {
			var err error
			result, err = fn(model, result, stream, credentials)
			if err != nil {
				return nil, fmt.Errorf("translate openai->%s: %w", to, err)
			}
		}
	}

	return result, nil
}

// TranslateResponse runs the target -> openai -> source pipeline, preserving
// fan-out: each step may return zero, one, or many chunks.
func (r *Registry) TranslateResponse(to, from Format, chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if !r.NeedsTranslation(from, to) {
		return []map[string]any{chunk}, nil
	}

	results := []map[string]any{chunk}

	// Step 1: target -> openai (if target is not openai).
	if to != FormatOpenAI {
		fn := r.ResponseTranslatorFor(to, FormatOpenAI)
		if fn != nil {
			converted, err := fn(chunk, state)
			if err != nil {
				return nil, fmt.Errorf("translate %s->openai response: %w", to, err)
			}
			results = converted
		}
	}

	// Step 2: openai -> source (if source is not openai).
	if from != FormatOpenAI {
		fn := r.ResponseTranslatorFor(FormatOpenAI, from)
		if fn != nil {
			var finalResults []map[string]any
			for _, c := range results {
				converted, err := fn(c, state)
				if err != nil {
					return nil, fmt.Errorf("translate openai->%s response: %w", from, err)
				}
				finalResults = append(finalResults, converted...)
			}
			results = finalResults
		}
	}

	return results, nil
}
