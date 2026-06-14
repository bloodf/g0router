package store

import (
	"reflect"
	"testing"
)

func TestGuardrailsDefaultOnFirstRead(t *testing.T) {
	st := newTestStore(t)

	g, err := st.GetGuardrails()
	if err != nil {
		t.Fatalf("GetGuardrails: %v", err)
	}
	if g.Enabled {
		t.Fatalf("default Enabled = true, want false")
	}
	if g.PIIRedactionEnabled {
		t.Fatalf("default PIIRedactionEnabled = true, want false")
	}
	if len(g.Blocklist) != 0 {
		t.Fatalf("default Blocklist = %v, want empty", g.Blocklist)
	}
	if len(g.PIIRedactionTypes) != 0 {
		t.Fatalf("default PIIRedactionTypes = %v, want empty", g.PIIRedactionTypes)
	}
}

func TestGuardrailsSetAndRoundTrip(t *testing.T) {
	st := newTestStore(t)

	in := &Guardrails{
		Enabled:             true,
		Blocklist:           []string{"password", "secret", "badword1"},
		PIIRedactionEnabled: true,
		PIIRedactionTypes:   []string{"email", "phone", "ssn"},
	}
	if err := st.SetGuardrails(in); err != nil {
		t.Fatalf("SetGuardrails: %v", err)
	}

	got, err := st.GetGuardrails()
	if err != nil {
		t.Fatalf("GetGuardrails: %v", err)
	}
	if !got.Enabled || !got.PIIRedactionEnabled {
		t.Fatalf("flags not round-tripped: %+v", got)
	}
	if !reflect.DeepEqual(got.Blocklist, []string{"password", "secret", "badword1"}) {
		t.Fatalf("Blocklist = %v", got.Blocklist)
	}
	if !reflect.DeepEqual(got.PIIRedactionTypes, []string{"email", "phone", "ssn"}) {
		t.Fatalf("PIIRedactionTypes = %v", got.PIIRedactionTypes)
	}
}

func TestGuardrailsSetIsSingleton(t *testing.T) {
	st := newTestStore(t)

	if err := st.SetGuardrails(&Guardrails{Enabled: true, Blocklist: []string{"a"}}); err != nil {
		t.Fatalf("SetGuardrails first: %v", err)
	}
	if err := st.SetGuardrails(&Guardrails{Enabled: false, Blocklist: []string{"b", "c"}}); err != nil {
		t.Fatalf("SetGuardrails second: %v", err)
	}

	got, err := st.GetGuardrails()
	if err != nil {
		t.Fatalf("GetGuardrails: %v", err)
	}
	if got.Enabled {
		t.Fatalf("second Set should overwrite enabled to false: %+v", got)
	}
	if !reflect.DeepEqual(got.Blocklist, []string{"b", "c"}) {
		t.Fatalf("second Set Blocklist = %v, want [b c]", got.Blocklist)
	}

	// Exactly one row in the singleton table.
	var n int
	if err := st.db.QueryRow("SELECT COUNT(*) FROM guardrails").Scan(&n); err != nil {
		t.Fatalf("count guardrails: %v", err)
	}
	if n != 1 {
		t.Fatalf("guardrails row count = %d, want 1", n)
	}
}
