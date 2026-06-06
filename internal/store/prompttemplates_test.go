package store

import (
	"testing"
)

func TestPromptTemplateCRUD(t *testing.T) {
	s := openTestStore(t)

	// Create
	tmpl, err := s.CreatePromptTemplate("default", "You are a helpful assistant.", []string{"gpt-4", "gpt-3.5-turbo"}, true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tmpl.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if tmpl.Name != "default" {
		t.Errorf("name = %q, want default", tmpl.Name)
	}
	if tmpl.SystemPrompt != "You are a helpful assistant." {
		t.Errorf("system_prompt = %q, want ...", tmpl.SystemPrompt)
	}
	if len(tmpl.Models) != 2 {
		t.Errorf("models = %v, want 2 items", tmpl.Models)
	}
	if !tmpl.IsActive {
		t.Error("expected active")
	}

	// List
	list, err := s.ListPromptTemplates()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// Get
	got, err := s.GetPromptTemplate(tmpl.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != tmpl.Name {
		t.Errorf("get name = %q, want %q", got.Name, tmpl.Name)
	}

	// Update
	if err := s.UpdatePromptTemplate(tmpl.ID, "default-updated", "Updated prompt.", []string{"gpt-4"}, false); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, err := s.GetPromptTemplate(tmpl.ID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if updated.Name != "default-updated" {
		t.Errorf("updated name = %q, want default-updated", updated.Name)
	}
	if updated.IsActive {
		t.Error("expected inactive")
	}

	// Delete
	if err := s.DeletePromptTemplate(tmpl.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = s.GetPromptTemplate(tmpl.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestPromptTemplateDuplicateName(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreatePromptTemplate("dup", "prompt", []string{"m1"}, true); err != nil {
		t.Fatalf("create first: %v", err)
	}
	_, err := s.CreatePromptTemplate("dup", "prompt2", []string{"m2"}, true)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestPromptTemplateUpdateDuplicateName(t *testing.T) {
	s := openTestStore(t)

	t1, _ := s.CreatePromptTemplate("a", "p", []string{"m"}, true)
	if _, err := s.CreatePromptTemplate("b", "p", []string{"m"}, true); err != nil {
		t.Fatalf("create second: %v", err)
	}

	err := s.UpdatePromptTemplate(t1.ID, "b", "p", []string{"m"}, true)
	if err == nil {
		t.Fatal("expected duplicate name error on update")
	}
}

func TestPromptTemplateUpdateNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.UpdatePromptTemplate(999, "x", "y", []string{"z"}, true)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestPromptTemplateDeleteNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.DeletePromptTemplate(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestPromptTemplateGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetPromptTemplate(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestPromptTemplateEmptyModels(t *testing.T) {
	s := openTestStore(t)

	tmpl, err := s.CreatePromptTemplate("empty", "prompt", []string{}, true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(tmpl.Models) != 0 {
		t.Errorf("models = %v, want empty", tmpl.Models)
	}
}

func TestPromptTemplateListOnClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()

	_, err := s.ListPromptTemplates()
	if err == nil {
		t.Error("expected error on closed db")
	}
}

func TestPromptTemplateGetOnClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()

	_, err := s.GetPromptTemplate(1)
	if err == nil {
		t.Error("expected error on closed db")
	}
}

func TestPromptTemplateCreateOnClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()

	_, err := s.CreatePromptTemplate("x", "y", []string{"z"}, true)
	if err == nil {
		t.Error("expected error on closed db")
	}
}

func TestPromptTemplateUpdateOnClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()

	err := s.UpdatePromptTemplate(1, "x", "y", []string{"z"}, true)
	if err == nil {
		t.Error("expected error on closed db")
	}
}

func TestPromptTemplateDeleteOnClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()

	err := s.DeletePromptTemplate(1)
	if err == nil {
		t.Error("expected error on closed db")
	}
}
