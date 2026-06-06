package store

import (
	"errors"
	"testing"
)

func TestDisabledModelCreateListIsDisabled(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateDisabledModel("openai", "gpt-4")
	if err != nil {
		t.Fatalf("CreateDisabledModel: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID should be set after create")
	}
	if created.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", created.Provider)
	}
	if created.Model != "gpt-4" {
		t.Fatalf("Model = %q, want gpt-4", created.Model)
	}
	if created.CreatedAt == "" {
		t.Fatal("CreatedAt should be set after create")
	}

	list, err := s.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Provider != "openai" || list[0].Model != "gpt-4" {
		t.Fatalf("list[0] = %+v, want openai/gpt-4", list[0])
	}

	disabled, err := s.IsModelDisabled("openai", "gpt-4")
	if err != nil {
		t.Fatalf("IsModelDisabled: %v", err)
	}
	if !disabled {
		t.Fatal("expected model to be disabled")
	}
}

func TestCreateDisabledModelDuplicate(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateDisabledModel("openai", "gpt-4"); err != nil {
		t.Fatalf("first CreateDisabledModel: %v", err)
	}
	_, err := s.CreateDisabledModel("openai", "gpt-4")
	if err == nil {
		t.Fatal("expected error for duplicate disabled model")
	}
	if !errors.Is(err, ErrDisabledModelExists) {
		t.Fatalf("expected ErrDisabledModelExists, got %v", err)
	}
}

func TestDeleteDisabledModel(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateDisabledModel("openai", "gpt-4"); err != nil {
		t.Fatalf("CreateDisabledModel: %v", err)
	}

	if err := s.DeleteDisabledModel("openai", "gpt-4"); err != nil {
		t.Fatalf("DeleteDisabledModel: %v", err)
	}

	disabled, err := s.IsModelDisabled("openai", "gpt-4")
	if err != nil {
		t.Fatalf("IsModelDisabled: %v", err)
	}
	if disabled {
		t.Fatal("expected model not to be disabled after delete")
	}

	list, err := s.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("len(list) = %d, want 0", len(list))
	}
}

func TestDeleteDisabledModelNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.DeleteDisabledModel("missing", "model")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCustomModelCreateListGetRoundTrip(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateCustomModel("openai", "gpt-custom", "My Custom GPT")
	if err != nil {
		t.Fatalf("CreateCustomModel: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID should be set after create")
	}
	if created.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", created.Provider)
	}
	if created.Model != "gpt-custom" {
		t.Fatalf("Model = %q, want gpt-custom", created.Model)
	}
	if created.DisplayName != "My Custom GPT" {
		t.Fatalf("DisplayName = %q, want My Custom GPT", created.DisplayName)
	}
	if created.CreatedAt == "" {
		t.Fatal("CreatedAt should be set after create")
	}

	list, err := s.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].ID != created.ID {
		t.Fatalf("list[0].ID = %q, want %q", list[0].ID, created.ID)
	}

	got, err := s.GetCustomModel(created.ID)
	if err != nil {
		t.Fatalf("GetCustomModel: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", got.Provider)
	}
	if got.Model != "gpt-custom" {
		t.Fatalf("Model = %q, want gpt-custom", got.Model)
	}
	if got.DisplayName != "My Custom GPT" {
		t.Fatalf("DisplayName = %q, want My Custom GPT", got.DisplayName)
	}
}

func TestCreateCustomModelDuplicate(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateCustomModel("openai", "gpt-custom", "My Custom GPT"); err != nil {
		t.Fatalf("first CreateCustomModel: %v", err)
	}
	_, err := s.CreateCustomModel("openai", "gpt-custom", "Duplicate")
	if err == nil {
		t.Fatal("expected error for duplicate custom model")
	}
	if !errors.Is(err, ErrCustomModelExists) {
		t.Fatalf("expected ErrCustomModelExists, got %v", err)
	}
}

func TestDeleteCustomModel(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateCustomModel("openai", "gpt-custom", "My Custom GPT")
	if err != nil {
		t.Fatalf("CreateCustomModel: %v", err)
	}

	if err := s.DeleteCustomModel(created.ID); err != nil {
		t.Fatalf("DeleteCustomModel: %v", err)
	}

	_, err = s.GetCustomModel(created.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetCustomModelNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetCustomModel("nonexistent-id")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
