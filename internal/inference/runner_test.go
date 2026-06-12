package inference

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
)

func TestAccountRunnerWrapsTransient(t *testing.T) {
	cs := &fakeConnStore{conns: []*store.Connection{makeConn("c1", "openai")}}
	sel := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)
	runner := &AccountRunner{sel: sel}

	cases := []struct {
		status        int
		wantTransient bool
	}{
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{401, false},
		{429, false},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			err := runner.RunModel("gpt-4", func(conn *store.Connection) (Verdict, error) {
				return VerdictUnknown, &schemas.ProviderError{StatusCode: tc.status, Message: "boom"}
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if got := errors.Is(err, ErrModelTransient); got != tc.wantTransient {
				t.Errorf("errors.Is(ErrModelTransient) = %v, want %v", got, tc.wantTransient)
			}
			var pe *schemas.ProviderError
			if !errors.As(err, &pe) {
				t.Errorf("original ProviderError not reachable: %v", err)
			} else if pe.StatusCode != tc.status {
				t.Errorf("ProviderError.StatusCode = %d, want %d", pe.StatusCode, tc.status)
			}
		})
	}
}

func TestAccountRunnerDelegatesToSelection(t *testing.T) {
	cs := &fakeConnStore{conns: []*store.Connection{makeConn("c1", "openai"), makeConn("c2", "openai")}}
	sel := NewSelectionEngine(cs, &fakeSettingStore{}, &fakeCooldownForSelection{}, time.Now)
	runner := &AccountRunner{sel: sel}

	var seen []string
	call := 0
	err := runner.RunModel("gpt-4", func(conn *store.Connection) (Verdict, error) {
		seen = append(seen, conn.ID)
		call++
		if call == 1 {
			return VerdictRateLimit, nil
		}
		return VerdictUnknown, nil
	})
	if err != nil {
		t.Fatalf("RunModel: %v", err)
	}
	if len(seen) != 2 || seen[0] != "c1" || seen[1] != "c2" {
		t.Errorf("connections = %v, want [c1, c2]", seen)
	}
}
