package inference

import (
	"strings"
	"testing"
)

// fakeAliasStore is an in-memory AliasStore implementation for tests.
type fakeAliasStore struct {
	aliases map[string]string
}

func (f *fakeAliasStore) CreateAlias(name, target string) error {
	f.aliases[name] = target
	return nil
}

func (f *fakeAliasStore) ResolveChain(name string) (string, error) {
	seen := make(map[string]bool)
	cur := name
	for i := 0; i < 10; i++ {
		if seen[cur] {
			return "", nil
		}
		seen[cur] = true
		next, ok := f.aliases[cur]
		if !ok {
			return cur, nil
		}
		cur = next
	}
	return cur, nil
}

func TestResolveModelAliasChain(t *testing.T) {
	st := &fakeAliasStore{aliases: map[string]string{
		"A": "B",
		"B": "C",
		"C": "claude-3-5-sonnet",
	}}

	if got := ResolveModelAlias(st, "A"); got != "claude-3-5-sonnet" {
		t.Errorf("ResolveModelAlias(A) = %q, want claude-3-5-sonnet", got)
	}
}

func TestResolveModelAliasMissingPassthrough(t *testing.T) {
	st := &fakeAliasStore{aliases: map[string]string{}}

	if got := ResolveModelAlias(st, "unknown"); got != "unknown" {
		t.Errorf("ResolveModelAlias(unknown) = %q, want unknown", got)
	}
}

func TestCreateAliasCycleRejected(t *testing.T) {
	st := &fakeAliasStore{aliases: map[string]string{
		"A": "B",
	}}

	err := CreateAlias(st, "B", "A")
	if err == nil {
		t.Fatal("CreateAlias(B,A) should reject cycle")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("CreateAlias(B,A) error = %v, want error containing 'cycle'", err)
	}
}

func TestCreateAliasSelfLoopRejected(t *testing.T) {
	st := &fakeAliasStore{aliases: map[string]string{}}

	err := CreateAlias(st, "A", "A")
	if err == nil {
		t.Fatal("CreateAlias(A,A) should reject self-loop")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("CreateAlias(A,A) error = %v, want error containing 'cycle'", err)
	}
}
