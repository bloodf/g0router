package store

import (
	"errors"
	"testing"
)

func TestModelAliasSetAndResolve(t *testing.T) {
	s := openTestStore(t)

	alias := ModelAlias{
		Alias:    "fast",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}
	if err := s.SetModelAlias(alias); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}

	got, err := s.ResolveModelAlias("fast")
	if err != nil {
		t.Fatalf("ResolveModelAlias: %v", err)
	}
	if got != alias {
		t.Fatalf("alias = %+v, want %+v", got, alias)
	}
}

func TestModelAliasSetReplacesExisting(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetModelAlias(ModelAlias{Alias: "fast", Provider: "groq", Model: "llama-3.3-70b-versatile"}); err != nil {
		t.Fatalf("first SetModelAlias: %v", err)
	}
	want := ModelAlias{Alias: "fast", Provider: "openai", Model: "gpt-4o-mini"}
	if err := s.SetModelAlias(want); err != nil {
		t.Fatalf("second SetModelAlias: %v", err)
	}

	got, err := s.ResolveModelAlias("fast")
	if err != nil {
		t.Fatalf("ResolveModelAlias: %v", err)
	}
	if got != want {
		t.Fatalf("alias = %+v, want %+v", got, want)
	}
}

func TestModelAliasListOrdersByAlias(t *testing.T) {
	s := openTestStore(t)

	for _, alias := range []ModelAlias{
		{Alias: "smart", Provider: "anthropic", Model: "claude-sonnet-4-20250514"},
		{Alias: "cheap", Provider: "openai", Model: "gpt-4o-mini"},
	} {
		if err := s.SetModelAlias(alias); err != nil {
			t.Fatalf("SetModelAlias %s: %v", alias.Alias, err)
		}
	}

	aliases, err := s.ListModelAliases()
	if err != nil {
		t.Fatalf("ListModelAliases: %v", err)
	}
	want := []ModelAlias{
		{Alias: "cheap", Provider: "openai", Model: "gpt-4o-mini"},
		{Alias: "smart", Provider: "anthropic", Model: "claude-sonnet-4-20250514"},
	}
	if len(aliases) != len(want) {
		t.Fatalf("len = %d, want %d", len(aliases), len(want))
	}
	for i := range want {
		if aliases[i] != want[i] {
			t.Fatalf("alias %d = %+v, want %+v", i, aliases[i], want[i])
		}
	}
}

func TestModelAliasDelete(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetModelAlias(ModelAlias{Alias: "fast", Provider: "groq", Model: "llama-3.3-70b-versatile"}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}
	if err := s.DeleteModelAlias("fast"); err != nil {
		t.Fatalf("DeleteModelAlias: %v", err)
	}

	_, err := s.ResolveModelAlias("fast")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestModelAliasNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.ResolveModelAlias("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	err = s.DeleteModelAlias("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound deleting missing alias, got %v", err)
	}
}
