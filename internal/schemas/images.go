package schemas

// ImageGenerationRequest is the payload for POST /v1/images/generations.
type ImageGenerationRequest struct {
	Prompt         string  `json:"prompt"`
	Model          string  `json:"model,omitempty"`
	N              *int    `json:"n,omitempty"`
	Quality        *string `json:"quality,omitempty"`
	ResponseFormat *string `json:"response_format,omitempty"`
	Size           *string `json:"size,omitempty"`
	Style          *string `json:"style,omitempty"`
	User           string  `json:"user,omitempty"`
}

// ImageGenerationResponse is the response for image generation.
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData is a single generated image result.
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageEditRequest is the payload for POST /v1/images/edits.
type ImageEditRequest struct {
	Image          []byte  `json:"-"`
	Mask           []byte  `json:"-"`
	Prompt         string  `json:"prompt"`
	Model          string  `json:"model,omitempty"`
	N              *int    `json:"n,omitempty"`
	Size           *string `json:"size,omitempty"`
	ResponseFormat *string `json:"response_format,omitempty"`
	User           string  `json:"user,omitempty"`
}

// ImageVariationRequest is the payload for POST /v1/images/variations.
type ImageVariationRequest struct {
	Image          []byte  `json:"-"`
	Model          string  `json:"model,omitempty"`
	N              *int    `json:"n,omitempty"`
	Size           *string `json:"size,omitempty"`
	ResponseFormat *string `json:"response_format,omitempty"`
	User           string  `json:"user,omitempty"`
}
