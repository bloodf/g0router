package api

import (
	"testing"

	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/valyala/fasthttp"
)

func TestValidVirtualKeyStoreNil(t *testing.T) {
	srv := NewServer(ServerConfig{
		Port:    0,
		Version: "test",
		Store:   nil,
	})
	ctx := &fasthttp.RequestCtx{}
	ok, err := srv.validVirtualKey(ctx, "gvk-test")
	if err != nil {
		t.Fatalf("validVirtualKey: %v", err)
	}
	if ok {
		t.Fatal("expected false when store is nil")
	}
}

func TestValidVirtualKeyGovernanceNil(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: nil,
	})
	ctx := &fasthttp.RequestCtx{}
	ok, err := srv.validVirtualKey(ctx, "gvk-test")
	if err != nil {
		t.Fatalf("validVirtualKey: %v", err)
	}
	if ok {
		t.Fatal("expected false when governance is nil")
	}
}

func TestValidVirtualKeyValidateError(t *testing.T) {
	s := newAPITestStore(t)
	s.Close()
	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	_, err := srv.validVirtualKey(ctx, "gvk-test")
	if err == nil {
		t.Fatal("expected error from validate")
	}
}
