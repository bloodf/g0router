package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/guardrails"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestPipelineApplyGuardrailsBlocked(t *testing.T) {
	p := NewPipeline(nil, &fakeSettingsProvider{
		guardrailsEnabled:   true,
		guardrailsBlocklist: []string{"badword"},
	}, nil)
	req := providers.ChatRequest{Model: "gpt-4", Messages: []providers.Message{{Role: "user", Content: "hello badword"}}}
	_, err := p.Process(context.Background(), &req)
	if !errors.Is(err, guardrails.ErrBlocklistMatch) {
		t.Fatalf("expected ErrBlocklistMatch, got %v", err)
	}
}

func TestPipelineApplyGuardrailsPIINoRedaction(t *testing.T) {
	p := NewPipeline(nil, &fakeSettingsProvider{
		guardrailsEnabled:   true,
		guardrailsBlocklist: []string{"zzzzz"},
		piiRedactionEnabled: true,
		piiRedactionTypes:   []string{"email"},
	}, nil)
	req := providers.ChatRequest{Model: "gpt-4", Messages: []providers.Message{{Role: "user", Content: "hello world"}}}
	processed, err := p.Process(context.Background(), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed == nil {
		t.Fatal("expected processed request")
	}
}

func TestPipelineInjectPromptTemplatesError(t *testing.T) {
	p := NewPipelineWithTemplates(nil, &fakeSettingsProvider{}, nil, &fakeTemplateProvider{err: errors.New("boom")})
	req := providers.ChatRequest{Model: "gpt-4", Messages: []providers.Message{{Role: "user", Content: "hi"}}}
	processed, err := p.Process(context.Background(), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed == nil {
		t.Fatal("expected processed request")
	}
}

type fakeTemplateProvider struct {
	templates []store.PromptTemplate
	err       error
}

func (f *fakeTemplateProvider) ListPromptTemplates() ([]store.PromptTemplate, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.templates, nil
}
