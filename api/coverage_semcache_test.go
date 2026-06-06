package api

import (
	"testing"

	"github.com/bloodf/g0router/internal/proxy"
)

func TestInitSemanticCacheNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	srv.initSemanticCache()
}

func TestInitSemanticCacheFlagDisabled(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	srv.initSemanticCache()
}

func TestInitSemanticCacheNonEngine(t *testing.T) {
	s := newAPITestStore(t)
	flag, err := s.GetFeatureFlagByKey("semantic_cache")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}
	if err := s.ToggleFeatureFlag(flag.ID, true); err != nil {
		t.Fatalf("ToggleFeatureFlag: %v", err)
	}
	srv := NewServer(ServerConfig{Store: s, InferenceEngine: routeInferenceEngine{}})
	srv.initSemanticCache()
}

func TestInitSemanticCacheWired(t *testing.T) {
	s := newAPITestStore(t)
	flag, err := s.GetFeatureFlagByKey("semantic_cache")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}
	if err := s.ToggleFeatureFlag(flag.ID, true); err != nil {
		t.Fatalf("ToggleFeatureFlag: %v", err)
	}

	engine := proxy.NewEngine(s)
	srv := NewServer(ServerConfig{Store: s, InferenceEngine: engine})
	srv.initSemanticCache()
}
