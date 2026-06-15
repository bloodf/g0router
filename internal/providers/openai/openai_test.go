package openai

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
}

func TestProviderSetNetworkConfig(t *testing.T) {
	p := NewProvider()
	p.SetNetworkConfig(schemas.NetworkConfig{Timeout: 30, ProxyURL: "http://proxy"})
	if p.networkConfig.Timeout != 30 {
		t.Errorf("timeout = %d, want 30", p.networkConfig.Timeout)
	}
	if p.networkConfig.ProxyURL != "http://proxy" {
		t.Errorf("proxy = %q, want http://proxy", p.networkConfig.ProxyURL)
	}
}

func TestNotImplementedStubs(t *testing.T) {
	p := NewProvider()
	ctx := &schemas.GatewayContext{RequestID: "test-1"}
	key := schemas.Key{ID: "key-1", Provider: "openai", Value: "sk-test"}

	tests := []struct {
		name string
		err  *schemas.ProviderError
	}{
		{"Responses", func() *schemas.ProviderError { _, e := p.Responses(ctx, key, nil); return e }()},
		{"ResponsesStream", func() *schemas.ProviderError { _, e := p.ResponsesStream(ctx, nil, key, nil); return e }()},
		{"CountTokens", func() *schemas.ProviderError { _, e := p.CountTokens(ctx, key, nil); return e }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tt.err.Type != "not_implemented" {
				t.Errorf("type = %q, want not_implemented", tt.err.Type)
			}
			if tt.err.StatusCode != 501 {
				t.Errorf("status = %d, want 501", tt.err.StatusCode)
			}
		})
	}
}

func TestErrorConverter(t *testing.T) {
	ec := NewErrorConverter()
	body := []byte(`{"error":{"message":"bad request","type":"invalid_request_error","code":"400"}}`)
	meta := schemas.ErrorMeta{Provider: "openai", StatusCode: 400}
	err := ec.Convert(400, body, meta)

	if err.Message != "bad request" {
		t.Errorf("message = %q, want bad request", err.Message)
	}
	if err.Type != "invalid_request_error" {
		t.Errorf("type = %q, want invalid_request_error", err.Type)
	}
	if err.Code == nil || *err.Code != "400" {
		t.Errorf("code = %v, want 400", err.Code)
	}
	if err.StatusCode != 400 {
		t.Errorf("status = %d, want 400", err.StatusCode)
	}
}

func TestErrorConverterMalformed(t *testing.T) {
	ec := NewErrorConverter()
	body := []byte(`not json`)
	meta := schemas.ErrorMeta{Provider: "openai", StatusCode: 500}
	err := ec.Convert(500, body, meta)

	if err.Type != "api_error" {
		t.Errorf("type = %q, want api_error", err.Type)
	}
	if err.StatusCode != 500 {
		t.Errorf("status = %d, want 500", err.StatusCode)
	}
}
