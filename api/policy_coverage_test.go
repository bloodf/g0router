package api

import (
	"testing"

	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/ratelimit"
)

func TestRecordVirtualKeyUsageNilID(t *testing.T) {
	s := &Server{config: ServerConfig{}}
	s.recordVirtualKeyUsage(nil, nil, "gpt-4o", nil, &providers.ChatResponse{}, nil)
}

func TestRecordVirtualKeyUsageEmptyID(t *testing.T) {
	empty := ""
	s := &Server{config: ServerConfig{}}
	s.recordVirtualKeyUsage(&empty, nil, "gpt-4o", nil, &providers.ChatResponse{}, nil)
}

func TestRecordVirtualKeyUsageNilGovernance(t *testing.T) {
	id := "1"
	s := &Server{config: ServerConfig{Governance: nil}}
	s.recordVirtualKeyUsage(&id, nil, "gpt-4o", nil, &providers.ChatResponse{}, nil)
}

func TestRecordVirtualKeyUsageInvalidKeyID(t *testing.T) {
	id := "abc"
	s := &Server{config: ServerConfig{Governance: governance.New(nil, nil)}}
	s.recordVirtualKeyUsage(&id, nil, "gpt-4o", nil, &providers.ChatResponse{}, nil)
}

func TestRecordVirtualKeyUsageInvalidTeamID(t *testing.T) {
	id := "1"
	teamID := "abc"
	s := newAPITestStore(t)
	key, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	_ = key

	srv := &Server{config: ServerConfig{Store: s, Governance: governance.New(s, ratelimit.NewLimiter())}}
	srv.recordVirtualKeyUsage(&id, &teamID, "gpt-4o", nil, &providers.ChatResponse{Usage: &providers.Usage{TotalTokens: 10}}, nil)
}

func TestRecordVirtualKeyUsageModelFromRequest(t *testing.T) {
	id := "1"
	s := newAPITestStore(t)
	key, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	_ = key

	srv := &Server{config: ServerConfig{Store: s, Governance: governance.New(s, ratelimit.NewLimiter())}}
	req := &providers.ChatRequest{Model: "gpt-4o-mini"}
	srv.recordVirtualKeyUsage(&id, nil, "", req, &providers.ChatResponse{Usage: &providers.Usage{TotalTokens: 10}}, nil)
}

func TestRecordVirtualKeyUsageProviderFromResponse(t *testing.T) {
	id := "1"
	s := newAPITestStore(t)
	key, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	_ = key

	srv := &Server{config: ServerConfig{Store: s, Governance: governance.New(s, ratelimit.NewLimiter())}}
	srv.recordVirtualKeyUsage(&id, nil, "gpt-4o", nil, &providers.ChatResponse{Provider: providers.ProviderAnthropic, Usage: &providers.Usage{TotalTokens: 10}}, nil)
}

func TestRecordVirtualKeyUsageStreamUsage(t *testing.T) {
	id := "1"
	s := newAPITestStore(t)
	key, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	_ = key

	srv := &Server{config: ServerConfig{Store: s, Governance: governance.New(s, ratelimit.NewLimiter())}}
	streamUsage := &providers.Usage{TotalTokens: 20}
	srv.recordVirtualKeyUsage(&id, nil, "gpt-4o", nil, nil, streamUsage)
}

func TestRecordVirtualKeyUsageNoUsage(t *testing.T) {
	id := "1"
	s := newAPITestStore(t)
	key, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	_ = key

	srv := &Server{config: ServerConfig{Store: s, Governance: governance.New(s, ratelimit.NewLimiter())}}
	srv.recordVirtualKeyUsage(&id, nil, "gpt-4o", nil, nil, nil)
}

func TestRecordVirtualKeyUsageWithTeam(t *testing.T) {
	s := newAPITestStore(t)
	team, err := s.CreateTeam("eng", floatPtr(1000.0), "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	_, _, err = s.CreateVirtualKey("vk", &team.ID, floatPtr(100.0), "monthly", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	srv := &Server{config: ServerConfig{Store: s, Governance: governance.New(s, ratelimit.NewLimiter())}}
	vid := "1"
	tid := "1"
	srv.recordVirtualKeyUsage(&vid, &tid, "gpt-4o", nil, &providers.ChatResponse{Usage: &providers.Usage{TotalTokens: 10}}, nil)
}
