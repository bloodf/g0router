package schemas

// RequestType categorizes the kind of inference request.
type RequestType string

// Supported request type constants.
const (
	RequestTypeChat        RequestType = "chat"
	RequestTypeCompletion  RequestType = "completion"
	RequestTypeEmbedding   RequestType = "embedding"
	RequestTypeImage       RequestType = "image"
	RequestTypeAudio       RequestType = "audio"
	RequestTypeResponses   RequestType = "responses"
)

// PricingEntry holds per-provider, per-model pricing data.
type PricingEntry struct {
	Provider         string      `json:"provider"`
	Model            string      `json:"model"`
	Mode             RequestType `json:"mode"`
	InputPrice       float64     `json:"input_price"`
	OutputPrice      float64     `json:"output_price"`
	ImagePrice       float64     `json:"image_price,omitempty"`
	AudioInputPrice  float64     `json:"audio_input_price,omitempty"`
	AudioOutputPrice float64     `json:"audio_output_price,omitempty"`
	TieredPricing    []Tier      `json:"tiered_pricing,omitempty"`
}

// Tier defines a pricing tier based on token threshold.
type Tier struct {
	Threshold   int     `json:"threshold"`
	InputPrice  float64 `json:"input_price"`
	OutputPrice float64 `json:"output_price"`
}

// ModelCapability describes what a model can do.
type ModelCapability struct {
	ContextWindow int      `json:"context_window"`
	Modalities    []string `json:"modalities"`
	Tools         bool     `json:"tools"`
	Reasoning     bool     `json:"reasoning"`
	NoTemperature bool     `json:"no_temperature"`
}

// Cost is the computed cost for a request.
type Cost struct {
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	TotalCost  float64 `json:"total_cost"`
}
