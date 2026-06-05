package openaicompat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

// Embeddings proxies POST /v1/embeddings to the configured upstream. Many
// OpenAI-compatible providers expose this endpoint; images and audio are left
// unimplemented so the engine returns ErrCapabilityUnsupported for them.
func (p *Provider) Embeddings(ctx context.Context, key providers.Key, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%s embeddings: nil request", p.provider)
	}

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/embeddings", key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s embeddings: %w", p.provider, err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(p.provider, resp)
	}

	var decoded providers.EmbeddingsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse %s embeddings response: %w", p.provider, err)
	}
	return &decoded, nil
}
