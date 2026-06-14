package voyageai

import (
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestNewVoyageAI(t *testing.T) {
	p, err := New("voyage-ai")
	if err != nil {
		t.Fatalf("New(\"voyage-ai\") error: %v", err)
	}
	if p == nil {
		t.Fatal("New(\"voyage-ai\") returned nil provider")
	}
	if got, want := p.GetProvider(), schemas.ModelProvider("voyage-ai"); got != want {
		t.Errorf("GetProvider() = %q, want %q", got, want)
	}
}

func TestNewVoyageAIRejectsOther(t *testing.T) {
	for _, id := range []string{"bogus", "openai", "deepseek"} {
		if _, err := New(id); err == nil {
			t.Errorf("New(%q) error = nil, want error (not a voyage provider)", id)
		}
	}
}
