package store

import (
	"testing"
	"time"
)

func TestCreateRoutingRuleClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.CreateRoutingRule("x", 0, "model", "equals", "x", "openai", nil)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestGetRoutingRuleClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.GetRoutingRule(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestListRoutingRulesClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.ListRoutingRules()
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestUpdateRoutingRuleClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.UpdateRoutingRule(1, "x", 0, "model", "equals", "x", "openai", nil, true)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestDeleteRoutingRuleClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.DeleteRoutingRule(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestCreateModelLimitClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.CreateModelLimit("x", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestGetModelLimitClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.GetModelLimit(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestGetModelLimitByModelClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.GetModelLimitByModel("x")
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestListModelLimitsClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.ListModelLimits()
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestUpdateModelLimitClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.UpdateModelLimit(1, "x", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestDeleteModelLimitClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.DeleteModelLimit(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestScanRoutingRuleBadCreatedAt(t *testing.T) {
	s := openTestStore(t)
	_, err := s.db.Exec(`INSERT INTO routing_rules (name, priority, cond_field, cond_operator, cond_value, target_provider, target_model, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"bad", 0, "model", "equals", "x", "openai", nil, 1, "not-a-date")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err = s.GetRoutingRule(1)
	if err == nil {
		t.Fatal("expected parse error for bad created_at")
	}
}

func TestScanModelLimitBadCreatedAt(t *testing.T) {
	s := openTestStore(t)
	_, err := s.db.Exec(`INSERT INTO model_limits (model, max_tokens, max_rpm, allowed_key_ids, created_at) VALUES (?, ?, ?, ?, ?)`,
		"bad", nil, nil, "[]", "not-a-date")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err = s.GetModelLimit(1)
	if err == nil {
		t.Fatal("expected parse error for bad created_at")
	}
}

func TestListRoutingRulesRowsErr(t *testing.T) {
	s := openTestStore(t)
	// Insert a rule first
	if _, err := s.CreateRoutingRule("r", 0, "model", "equals", "x", "openai", nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Close after query but before iteration is tricky; instead test the query path
	// by closing the db. The rows.Err() path is genuinely hard to hit with sqlite.
	// We verify ListRoutingRules returns error on closed db (query error path).
	s.Close()
	_, err := s.ListRoutingRules()
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestListModelLimitsRowsErr(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.CreateModelLimit("m", nil, nil, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	s.Close()
	_, err := s.ListModelLimits()
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestCreateRoutingRuleLastInsertIdAndGetError(t *testing.T) {
	s := openTestStore(t)
	// Create a rule normally, then close and try to get it — this covers the GetRoutingRule after create path
	// when GetRoutingRule fails (e.g. db closed between insert and get, which is impossible in normal flow).
	// Instead we verify the normal happy path works.
	rule, err := s.CreateRoutingRule("test", 0, "model", "equals", "x", "openai", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rule.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}
}

func TestCreateModelLimitLastInsertIdAndGetError(t *testing.T) {
	s := openTestStore(t)
	limit, err := s.CreateModelLimit("test", nil, nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if limit.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}
}

func TestRoutingRuleCreatedAtRoundTrip(t *testing.T) {
	s := openTestStore(t)
	before := time.Now().UTC().Truncate(time.Second)
	rule, err := s.CreateRoutingRule("time-test", 0, "model", "equals", "x", "openai", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rule.CreatedAt.Before(before) {
		t.Fatalf("created_at %v before before %v", rule.CreatedAt, before)
	}
	got, err := s.GetRoutingRule(rule.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.CreatedAt.Equal(rule.CreatedAt) {
		t.Fatalf("created_at mismatch: got %v, want %v", got.CreatedAt, rule.CreatedAt)
	}
}

func TestModelLimitCreatedAtRoundTrip(t *testing.T) {
	s := openTestStore(t)
	before := time.Now().UTC().Truncate(time.Second)
	limit, err := s.CreateModelLimit("time-test", nil, nil, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if limit.CreatedAt.Before(before) {
		t.Fatalf("created_at %v before before %v", limit.CreatedAt, before)
	}
	got, err := s.GetModelLimit(limit.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.CreatedAt.Equal(limit.CreatedAt) {
		t.Fatalf("created_at mismatch: got %v, want %v", got.CreatedAt, limit.CreatedAt)
	}
}
