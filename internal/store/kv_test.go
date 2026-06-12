package store

import (
	"testing"
)

func TestKVRoundTrip(t *testing.T) {
	st := newTestStore(t)

	scope := "test"
	if err := st.SetKV(scope, "k1", "v1"); err != nil {
		t.Fatalf("SetKV: %v", err)
	}
	if err := st.SetKV(scope, "k2", "v2"); err != nil {
		t.Fatalf("SetKV: %v", err)
	}

	v, err := st.GetKV(scope, "k1")
	if err != nil {
		t.Fatalf("GetKV k1: %v", err)
	}
	if v != "v1" {
		t.Errorf("GetKV k1 = %q, want v1", v)
	}

	// Missing key returns ("", nil), matching settings.go ErrNoRows convention.
	missing, err := st.GetKV(scope, "no-such-key")
	if err != nil {
		t.Fatalf("GetKV missing: expected nil err, got %v", err)
	}
	if missing != "" {
		t.Errorf("GetKV missing = %q, want empty", missing)
	}

	all, err := st.ListKV(scope)
	if err != nil {
		t.Fatalf("ListKV: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListKV len = %d, want 2", len(all))
	}
	if all["k1"] != "v1" || all["k2"] != "v2" {
		t.Errorf("ListKV = %v, want k1=v1, k2=v2", all)
	}

	// Upsert updates existing key.
	if err := st.SetKV(scope, "k1", "v1-updated"); err != nil {
		t.Fatalf("SetKV update: %v", err)
	}
	v, err = st.GetKV(scope, "k1")
	if err != nil {
		t.Fatalf("GetKV updated k1: %v", err)
	}
	if v != "v1-updated" {
		t.Errorf("GetKV updated k1 = %q, want v1-updated", v)
	}

	// Different scopes are isolated.
	if err := st.SetKV("other", "k1", "v-other"); err != nil {
		t.Fatalf("SetKV other scope: %v", err)
	}
	sameScope, err := st.ListKV(scope)
	if err != nil {
		t.Fatalf("ListKV scope after other: %v", err)
	}
	if sameScope["k1"] != "v1-updated" {
		t.Errorf("ListKV scope k1 = %q, want v1-updated", sameScope["k1"])
	}
}
