package bedrock

import (
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func TestWithProxyPool(t *testing.T) {
	p := New("")
	pool := &store.ProxyPool{Protocol: "http", Host: "proxy", Port: 8080}
	result := p.WithProxyPool(pool)
	if result == nil {
		t.Fatal("expected non-nil provider")
	}
	if result == p {
		t.Fatal("expected a new provider instance")
	}
}
