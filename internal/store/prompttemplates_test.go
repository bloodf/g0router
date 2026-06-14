package store

import (
	"errors"
	"reflect"
	"testing"
)

func TestPromptTemplateCRUD(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreatePromptTemplate(&PromptTemplate{
		Name:         "Code Review",
		SystemPrompt: "You are a senior code reviewer.",
		Models:       []string{"gpt-4o", "claude-sonnet-4"},
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("CreatePromptTemplate: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created ID is zero")
	}
	if created.Name != "Code Review" || created.SystemPrompt != "You are a senior code reviewer." {
		t.Fatalf("create fields not persisted: %+v", created)
	}
	if !reflect.DeepEqual(created.Models, []string{"gpt-4o", "claude-sonnet-4"}) {
		t.Fatalf("Models = %v", created.Models)
	}
	if !created.IsActive {
		t.Fatalf("IsActive = false, want true")
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatalf("timestamps not set: %+v", created)
	}

	got, err := st.GetPromptTemplateByID(created.ID)
	if err != nil {
		t.Fatalf("GetPromptTemplateByID: %v", err)
	}
	if !reflect.DeepEqual(got.Models, created.Models) {
		t.Fatalf("models round-trip mismatch: %v vs %v", got.Models, created.Models)
	}

	created2, err := st.CreatePromptTemplate(&PromptTemplate{Name: "Docs", Models: []string{"gpt-4o-mini"}, IsActive: true})
	if err != nil {
		t.Fatalf("CreatePromptTemplate second: %v", err)
	}

	list, err := st.ListPromptTemplates()
	if err != nil {
		t.Fatalf("ListPromptTemplates: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	// ORDER BY id ASC.
	if list[0].ID != created.ID {
		t.Fatalf("list[0].ID = %d, want %d", list[0].ID, created.ID)
	}

	updated, err := st.UpdatePromptTemplate(created.ID, &PromptTemplate{
		Name:         "Code Review v2",
		SystemPrompt: "Be concise.",
		Models:       []string{"gpt-4o"},
		IsActive:     false,
	})
	if err != nil {
		t.Fatalf("UpdatePromptTemplate: %v", err)
	}
	if updated.Name != "Code Review v2" || updated.SystemPrompt != "Be concise." {
		t.Fatalf("update not persisted: %+v", updated)
	}
	if !reflect.DeepEqual(updated.Models, []string{"gpt-4o"}) {
		t.Fatalf("updated models = %v", updated.Models)
	}
	if updated.IsActive {
		t.Fatalf("updated IsActive = true, want false")
	}

	if err := st.DeletePromptTemplate(created2.ID); err != nil {
		t.Fatalf("DeletePromptTemplate: %v", err)
	}
	if _, err := st.GetPromptTemplateByID(created2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted template err = %v, want ErrNotFound", err)
	}

	// Unknown id paths.
	if err := st.DeletePromptTemplate(99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpdatePromptTemplate(99999, &PromptTemplate{Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update unknown err = %v, want ErrNotFound", err)
	}
	if _, err := st.GetPromptTemplateByID(99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get unknown err = %v, want ErrNotFound", err)
	}
}

func TestPromptTemplateEmptyModelsRoundTrip(t *testing.T) {
	st := newTestStore(t)
	created, err := st.CreatePromptTemplate(&PromptTemplate{Name: "NoModels", IsActive: true})
	if err != nil {
		t.Fatalf("CreatePromptTemplate: %v", err)
	}
	got, err := st.GetPromptTemplateByID(created.ID)
	if err != nil {
		t.Fatalf("GetPromptTemplateByID: %v", err)
	}
	if len(got.Models) != 0 {
		t.Fatalf("Models = %v, want empty", got.Models)
	}
}
