package schemas

// TextCompletionRequest is the payload for POST /v1/completions (legacy).
type TextCompletionRequest struct {
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt"`
	Suffix           *string        `json:"suffix,omitempty"`
	MaxTokens        *int           `json:"max_tokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	N                *int           `json:"n,omitempty"`
	Stream           bool           `json:"stream"`
	Logprobs         *int           `json:"logprobs,omitempty"`
	Echo             *bool          `json:"echo,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  *float64       `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64       `json:"frequency_penalty,omitempty"`
	BestOf           *int           `json:"best_of,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
}

// TextCompletionResponse is the non-streaming response for legacy completions.
type TextCompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []TextCompletionChoice   `json:"choices"`
	Usage   *Usage                   `json:"usage,omitempty"`
}

// TextCompletionChoice is a single completion choice in the legacy endpoint.
type TextCompletionChoice struct {
	Text         string    `json:"text"`
	Index        int       `json:"index"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"`
	FinishReason string    `json:"finish_reason,omitempty"`
}
