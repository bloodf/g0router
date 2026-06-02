package streaming

type Response struct {
	ID         string           `json:"id"`
	Object     string           `json:"object"`
	CreatedAt  int64            `json:"created_at"`
	Model      string           `json:"model"`
	Status     string           `json:"status"`
	OutputText string           `json:"output_text,omitempty"`
	Output     []ResponseOutput `json:"output,omitempty"`
	Usage      *ResponseUsage   `json:"usage,omitempty"`
}

type ResponseOutput struct {
	Type    string            `json:"type"`
	Role    string            `json:"role,omitempty"`
	Content []ResponseContent `json:"content,omitempty"`
}

type ResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type ResponseEvent struct {
	Type           string    `json:"type"`
	Delta          string    `json:"delta,omitempty"`
	Text           string    `json:"text,omitempty"`
	SequenceNumber int       `json:"sequence_number,omitempty"`
	Response       *Response `json:"response,omitempty"`
}

type ResponsesAccumulator struct {
	response Response
	text     string
}

func NewResponsesAccumulator() *ResponsesAccumulator {
	return &ResponsesAccumulator{}
}

func (a *ResponsesAccumulator) AddEvent(event ResponseEvent) {
	switch event.Type {
	case "response.output_text.delta":
		a.text += event.Delta
	case "response.output_text.done":
		a.text = event.Text
	case "response.completed":
		if event.Response != nil {
			a.response = *event.Response
		}
	}
}

func (a *ResponsesAccumulator) Response() Response {
	resp := a.response
	if a.text != "" {
		resp.OutputText = a.text
	}
	return resp
}
