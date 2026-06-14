package store

import (
	"errors"
	"testing"
)

func TestCustomModelCRUD(t *testing.T) {
	st := newTestStore(t)

	// Create.
	created, err := st.CreateCustomModel(&CustomModel{
		Provider: "openai",
		ModelID:  "my-gpt",
		Name:     "My GPT",
		Config:   map[string]any{"context": float64(128000)},
	})
	if err != nil {
		t.Fatalf("CreateCustomModel: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("created custom model has empty id")
	}
	if created.ModelID != "my-gpt" || created.Provider != "openai" || created.Name != "My GPT" {
		t.Fatalf("created = %+v", created)
	}
	if created.CreatedAt == 0 {
		t.Fatalf("created CreatedAt = 0")
	}

	// List.
	list, err := st.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListCustomModels len = %d, want 1", len(list))
	}
	if list[0].ModelID != "my-gpt" {
		t.Fatalf("list[0] = %+v", list[0])
	}
	if got, _ := list[0].Config["context"].(float64); got != 128000 {
		t.Fatalf("config context = %v, want 128000", list[0].Config["context"])
	}

	// Delete.
	if err := st.DeleteCustomModel(created.ID); err != nil {
		t.Fatalf("DeleteCustomModel: %v", err)
	}
	list, err = st.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("after delete len = %d, want 0", len(list))
	}

	// Delete unknown → ErrNotFound.
	if err := st.DeleteCustomModel("nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown = %v, want ErrNotFound", err)
	}
}
