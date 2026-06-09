package schemas

// ResponsesRequest is the payload for POST /v1/responses.
type ResponsesRequest struct {
	Model                string            `json:"model"`
	Input                any               `json:"input"`
	Include              []string          `json:"include,omitempty"`
	Instructions         *string           `json:"instructions,omitempty"`
	MaxOutputTokens      *int              `json:"max_output_tokens,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	ParallelToolCalls    *bool             `json:"parallel_tool_calls,omitempty"`
	PreviousResponseID   *string           `json:"previous_response_id,omitempty"`
	Reasoning            *ReasoningConfig  `json:"reasoning,omitempty"`
	Store                *bool             `json:"store,omitempty"`
	Stream               *bool             `json:"stream,omitempty"`
	Temperature          *float64          `json:"temperature,omitempty"`
	Text                 *TextConfig       `json:"text,omitempty"`
	ToolChoice           *ToolChoice       `json:"tool_choice,omitempty"`
	Tools                []Tool            `json:"tools,omitempty"`
	TopP                 *float64          `json:"top_p,omitempty"`
	Truncation           *string           `json:"truncation,omitempty"`
	User                 string            `json:"user,omitempty"`
}

// ReasoningConfig controls reasoning behavior in the Responses API.
type ReasoningConfig struct {
	Effort          *string `json:"effort,omitempty"`
	GenerateSummary *string `json:"generate_summary,omitempty"`
}

// TextConfig configures text output in the Responses API.
type TextConfig struct {
	Format *ResponseFormat `json:"format,omitempty"`
}

// ResponsesResponse is the response for the Responses API.
type ResponsesResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	CreatedAt         int64              `json:"created_at"`
	Model             string             `json:"model"`
	Output            []ResponseOutputItem `json:"output"`
	Status            string             `json:"status"`
	Usage             *Usage             `json:"usage,omitempty"`
	Error             *APIError           `json:"error,omitempty"`
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
	Instructions      *string            `json:"instructions,omitempty"`
	MaxOutputTokens   *int               `json:"max_output_tokens,omitempty"`
	ParallelToolCalls *bool              `json:"parallel_tool_calls,omitempty"`
	PreviousResponseID *string            `json:"previous_response_id,omitempty"`
	Reasoning         *ReasoningConfig   `json:"reasoning,omitempty"`
	Store             *bool              `json:"store,omitempty"`
	Temperature       *float64           `json:"temperature,omitempty"`
	Text              *TextConfig        `json:"text,omitempty"`
	ToolChoice        *ToolChoice        `json:"tool_choice,omitempty"`
	Tools             []Tool             `json:"tools,omitempty"`
	TopP              *float64           `json:"top_p,omitempty"`
	Truncation        *string            `json:"truncation,omitempty"`
	User              string             `json:"user,omitempty"`
}

// ResponseOutputItem is a single item in the Responses API output array.
type ResponseOutputItem struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Status  string            `json:"status,omitempty"`
	Role    string            `json:"role,omitempty"`
	Content []ResponseContent `json:"content,omitempty"`
}

// ResponseContent is content inside a ResponseOutputItem.
type ResponseContent struct {
	Type        string               `json:"type"`
	Text        string               `json:"text,omitempty"`
	Annotations []ResponseAnnotation `json:"annotations,omitempty"`
}

// ResponseAnnotation is an annotation on response content.
type ResponseAnnotation struct {
	Type         string        `json:"type"`
	Text         string        `json:"text,omitempty"`
	FileCitation *FileCitation `json:"file_citation,omitempty"`
	URLCitation  *URLCitation  `json:"url_citation,omitempty"`
}

// FileCitation references a file in an annotation.
type FileCitation struct {
	FileID string `json:"file_id"`
	Quote  string `json:"quote"`
}

// URLCitation references a URL in an annotation.
type URLCitation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// IncompleteDetails explains why a response was incomplete.
type IncompleteDetails struct {
	Reason string `json:"reason"`
}
