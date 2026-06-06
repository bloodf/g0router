package store

import (
	"reflect"
	"testing"
)

func TestGetGuardrailsConfigDefaults(t *testing.T) {
	s := openTestStore(t)

	cfg, err := s.GetGuardrailsConfig()
	if err != nil {
		t.Fatalf("GetGuardrailsConfig: %v", err)
	}
	if cfg.GuardrailsEnabled {
		t.Error("GuardrailsEnabled should default to false")
	}
	if len(cfg.GuardrailsBlocklist) != 0 {
		t.Errorf("GuardrailsBlocklist = %v, want empty", cfg.GuardrailsBlocklist)
	}
	if cfg.PIIRedactionEnabled {
		t.Error("PIIRedactionEnabled should default to false")
	}
	if len(cfg.PIIRedactionTypes) != 0 {
		t.Errorf("PIIRedactionTypes = %v, want empty", cfg.PIIRedactionTypes)
	}
}

func TestUpdateAndGetGuardrailsConfigRoundTrip(t *testing.T) {
	s := openTestStore(t)

	want := GuardrailsConfig{
		GuardrailsEnabled:   true,
		GuardrailsBlocklist: []string{"badword", "forbidden"},
		PIIRedactionEnabled: true,
		PIIRedactionTypes:   []string{"email", "phone"},
	}

	if err := s.UpdateGuardrailsConfig(want); err != nil {
		t.Fatalf("UpdateGuardrailsConfig: %v", err)
	}

	got, err := s.GetGuardrailsConfig()
	if err != nil {
		t.Fatalf("GetGuardrailsConfig: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cfg = %+v, want %+v", got, want)
	}
}

func TestUpdateGuardrailsConfigEmptySlices(t *testing.T) {
	s := openTestStore(t)

	cfg := GuardrailsConfig{
		GuardrailsEnabled:   false,
		GuardrailsBlocklist: []string{},
		PIIRedactionEnabled: false,
		PIIRedactionTypes:   []string{},
	}

	if err := s.UpdateGuardrailsConfig(cfg); err != nil {
		t.Fatalf("UpdateGuardrailsConfig: %v", err)
	}

	got, err := s.GetGuardrailsConfig()
	if err != nil {
		t.Fatalf("GetGuardrailsConfig: %v", err)
	}
	if got.GuardrailsEnabled {
		t.Error("GuardrailsEnabled should be false")
	}
	if len(got.GuardrailsBlocklist) != 0 {
		t.Errorf("GuardrailsBlocklist = %v, want empty", got.GuardrailsBlocklist)
	}
}

func TestUpdateGuardrailsConfigIdempotent(t *testing.T) {
	s := openTestStore(t)

	cfg := GuardrailsConfig{
		GuardrailsEnabled:   true,
		GuardrailsBlocklist: []string{"term"},
		PIIRedactionEnabled: true,
		PIIRedactionTypes:   []string{"ssn"},
	}

	if err := s.UpdateGuardrailsConfig(cfg); err != nil {
		t.Fatalf("first UpdateGuardrailsConfig: %v", err)
	}
	if err := s.UpdateGuardrailsConfig(cfg); err != nil {
		t.Fatalf("second UpdateGuardrailsConfig: %v", err)
	}

	got, err := s.GetGuardrailsConfig()
	if err != nil {
		t.Fatalf("GetGuardrailsConfig: %v", err)
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("cfg = %+v, want %+v", got, cfg)
	}
}
