package store

import (
	"testing"
)

func TestCreateRoutingRule(t *testing.T) {
	s := openTestStore(t)

	rule, err := s.CreateRoutingRule("rule-1", 10, "model", "equals", "gpt-4o", "openai", stringPtr("gpt-4o"))
	if err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}
	if rule.ID == 0 {
		t.Error("ID should be set")
	}
	if rule.Name != "rule-1" {
		t.Errorf("Name = %q, want rule-1", rule.Name)
	}
	if rule.Priority != 10 {
		t.Errorf("Priority = %d, want 10", rule.Priority)
	}
	if rule.CondField != "model" {
		t.Errorf("CondField = %q, want model", rule.CondField)
	}
	if rule.CondOperator != "equals" {
		t.Errorf("CondOperator = %q, want equals", rule.CondOperator)
	}
	if rule.CondValue != "gpt-4o" {
		t.Errorf("CondValue = %q, want gpt-4o", rule.CondValue)
	}
	if rule.TargetProvider != "openai" {
		t.Errorf("TargetProvider = %q, want openai", rule.TargetProvider)
	}
	if rule.TargetModel == nil || *rule.TargetModel != "gpt-4o" {
		t.Errorf("TargetModel = %v, want gpt-4o", rule.TargetModel)
	}
	if !rule.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestGetRoutingRule(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateRoutingRule("rule-1", 0, "model", "equals", "x", "openai", nil)
	if err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}

	got, err := s.GetRoutingRule(created.ID)
	if err != nil {
		t.Fatalf("GetRoutingRule: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.Name != "rule-1" {
		t.Errorf("Name = %q, want rule-1", got.Name)
	}
}

func TestGetRoutingRuleNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetRoutingRule(999)
	if err == nil {
		t.Fatal("GetRoutingRule should error for missing id")
	}
}

func TestListRoutingRules(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateRoutingRule("a", 1, "model", "equals", "x", "openai", nil); err != nil {
		t.Fatalf("CreateRoutingRule a: %v", err)
	}
	if _, err := s.CreateRoutingRule("b", 2, "provider", "equals", "openai", "anthropic", nil); err != nil {
		t.Fatalf("CreateRoutingRule b: %v", err)
	}

	rules, err := s.ListRoutingRules()
	if err != nil {
		t.Fatalf("ListRoutingRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}
	// Should be ordered by priority desc, then created_at
	if rules[0].Priority != 2 {
		t.Errorf("first priority = %d, want 2", rules[0].Priority)
	}
}

func TestUpdateRoutingRule(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateRoutingRule("old", 0, "model", "equals", "x", "openai", nil)
	if err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}

	if err := s.UpdateRoutingRule(created.ID, "new", 5, "provider", "contains", "azure", "azure", stringPtr("gpt-4o"), false); err != nil {
		t.Fatalf("UpdateRoutingRule: %v", err)
	}

	got, err := s.GetRoutingRule(created.ID)
	if err != nil {
		t.Fatalf("GetRoutingRule: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Name = %q, want new", got.Name)
	}
	if got.Priority != 5 {
		t.Errorf("Priority = %d, want 5", got.Priority)
	}
	if got.CondField != "provider" {
		t.Errorf("CondField = %q, want provider", got.CondField)
	}
	if got.CondOperator != "contains" {
		t.Errorf("CondOperator = %q, want contains", got.CondOperator)
	}
	if got.CondValue != "azure" {
		t.Errorf("CondValue = %q, want azure", got.CondValue)
	}
	if got.TargetProvider != "azure" {
		t.Errorf("TargetProvider = %q, want azure", got.TargetProvider)
	}
	if got.TargetModel == nil || *got.TargetModel != "gpt-4o" {
		t.Errorf("TargetModel = %v, want gpt-4o", got.TargetModel)
	}
	if got.IsActive {
		t.Error("IsActive should be false")
	}
}

func TestUpdateRoutingRuleNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.UpdateRoutingRule(999, "x", 0, "model", "equals", "x", "openai", nil, true)
	if err == nil {
		t.Fatal("UpdateRoutingRule should error for missing id")
	}
}

func TestDeleteRoutingRule(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateRoutingRule("temp", 0, "model", "equals", "x", "openai", nil)
	if err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}

	if err := s.DeleteRoutingRule(created.ID); err != nil {
		t.Fatalf("DeleteRoutingRule: %v", err)
	}

	_, err = s.GetRoutingRule(created.ID)
	if err == nil {
		t.Fatal("GetRoutingRule should error after delete")
	}
}

func TestCreateModelLimit(t *testing.T) {
	s := openTestStore(t)

	limit, err := s.CreateModelLimit("gpt-4o", intPtr(4096), intPtr(60), []string{"key-1", "key-2"})
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}
	if limit.ID == 0 {
		t.Error("ID should be set")
	}
	if limit.Model != "gpt-4o" {
		t.Errorf("Model = %q, want gpt-4o", limit.Model)
	}
	if limit.MaxTokens == nil || *limit.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %v, want 4096", limit.MaxTokens)
	}
	if limit.MaxRPM == nil || *limit.MaxRPM != 60 {
		t.Errorf("MaxRPM = %v, want 60", limit.MaxRPM)
	}
	if len(limit.AllowedKeyIDs) != 2 || limit.AllowedKeyIDs[0] != "key-1" {
		t.Errorf("AllowedKeyIDs = %v, want [key-1 key-2]", limit.AllowedKeyIDs)
	}
}

func TestCreateModelLimitDuplicateModel(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateModelLimit("gpt-4o", nil, nil, nil); err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}
	_, err := s.CreateModelLimit("gpt-4o", nil, nil, nil)
	if err == nil {
		t.Fatal("CreateModelLimit should error for duplicate model")
	}
}

func TestGetModelLimit(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateModelLimit("gpt-4o", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	got, err := s.GetModelLimit(created.ID)
	if err != nil {
		t.Fatalf("GetModelLimit: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
}

func TestGetModelLimitNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetModelLimit(999)
	if err == nil {
		t.Fatal("GetModelLimit should error for missing id")
	}
}

func TestGetModelLimitByModel(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateModelLimit("gpt-4o", intPtr(4096), nil, nil); err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	got, err := s.GetModelLimitByModel("gpt-4o")
	if err != nil {
		t.Fatalf("GetModelLimitByModel: %v", err)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("Model = %q, want gpt-4o", got.Model)
	}
	if got.MaxTokens == nil || *got.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %v, want 4096", got.MaxTokens)
	}
}

func TestListModelLimits(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateModelLimit("gpt-4o", nil, nil, nil); err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}
	if _, err := s.CreateModelLimit("claude-sonnet", nil, nil, nil); err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	limits, err := s.ListModelLimits()
	if err != nil {
		t.Fatalf("ListModelLimits: %v", err)
	}
	if len(limits) != 2 {
		t.Fatalf("len(limits) = %d, want 2", len(limits))
	}
}

func TestUpdateModelLimit(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateModelLimit("gpt-4o", intPtr(4096), intPtr(60), nil)
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	if err := s.UpdateModelLimit(created.ID, "gpt-4o", intPtr(8192), intPtr(120), []string{"key-1"}); err != nil {
		t.Fatalf("UpdateModelLimit: %v", err)
	}

	got, err := s.GetModelLimit(created.ID)
	if err != nil {
		t.Fatalf("GetModelLimit: %v", err)
	}
	if got.MaxTokens == nil || *got.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %v, want 8192", got.MaxTokens)
	}
	if got.MaxRPM == nil || *got.MaxRPM != 120 {
		t.Errorf("MaxRPM = %v, want 120", got.MaxRPM)
	}
	if len(got.AllowedKeyIDs) != 1 || got.AllowedKeyIDs[0] != "key-1" {
		t.Errorf("AllowedKeyIDs = %v, want [key-1]", got.AllowedKeyIDs)
	}
}

func TestUpdateModelLimitNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.UpdateModelLimit(999, "x", nil, nil, nil)
	if err == nil {
		t.Fatal("UpdateModelLimit should error for missing id")
	}
}

func TestDeleteModelLimit(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateModelLimit("gpt-4o", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	if err := s.DeleteModelLimit(created.ID); err != nil {
		t.Fatalf("DeleteModelLimit: %v", err)
	}

	_, err = s.GetModelLimit(created.ID)
	if err == nil {
		t.Fatal("GetModelLimit should error after delete")
	}
}

func TestCreateModelLimitEmptyAllowedKeyIDs(t *testing.T) {
	s := openTestStore(t)

	limit, err := s.CreateModelLimit("gpt-4o", nil, nil, []string{})
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}
	if len(limit.AllowedKeyIDs) != 0 {
		t.Fatalf("AllowedKeyIDs = %v, want empty", limit.AllowedKeyIDs)
	}
}

func TestUpdateModelLimitToEmptyAllowedKeyIDs(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateModelLimit("gpt-4o", nil, nil, []string{"key-1"})
	if err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	if err := s.UpdateModelLimit(created.ID, "gpt-4o", nil, nil, []string{}); err != nil {
		t.Fatalf("UpdateModelLimit: %v", err)
	}

	got, err := s.GetModelLimit(created.ID)
	if err != nil {
		t.Fatalf("GetModelLimit: %v", err)
	}
	if len(got.AllowedKeyIDs) != 0 {
		t.Fatalf("AllowedKeyIDs = %v, want empty", got.AllowedKeyIDs)
	}
}

func TestGetModelLimitByModelNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetModelLimitByModel("nonexistent")
	if err == nil {
		t.Fatal("GetModelLimitByModel should error for missing model")
	}
}

func TestCreateRoutingRuleNoTargetModel(t *testing.T) {
	s := openTestStore(t)

	rule, err := s.CreateRoutingRule("rule-1", 0, "model", "equals", "gpt-4o", "openai", nil)
	if err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}
	if rule.TargetModel != nil {
		t.Fatalf("TargetModel = %v, want nil", rule.TargetModel)
	}
}

func TestUpdateRoutingRuleNoTargetModel(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateRoutingRule("rule-1", 0, "model", "equals", "gpt-4o", "openai", stringPtr("gpt-4o"))
	if err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}

	if err := s.UpdateRoutingRule(created.ID, "rule-1", 0, "model", "equals", "gpt-4o", "openai", nil, true); err != nil {
		t.Fatalf("UpdateRoutingRule: %v", err)
	}

	got, err := s.GetRoutingRule(created.ID)
	if err != nil {
		t.Fatalf("GetRoutingRule: %v", err)
	}
	if got.TargetModel != nil {
		t.Fatalf("TargetModel = %v, want nil", got.TargetModel)
	}
}


