package schemas

// EmbeddingRequest is the payload for POST /v1/embeddings.
type EmbeddingRequest struct {
	Input          any     `json:"input"`
	Model          string  `json:"model"`
	EncodingFormat *string `json:"encoding_format,omitempty"`
	Dimensions     *int    `json:"dimensions,omitempty"`
	User           string  `json:"user,omitempty"`
}

// EmbeddingResponse is the response for the embeddings endpoint.
type EmbeddingResponse struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  *Usage      `json:"usage,omitempty"`
}

// Embedding is a single embedding vector.
type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}
