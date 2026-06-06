package store

import (
	"testing"
	"time"
)

func TestCreateTeamDefaultBudgetPeriod(t *testing.T) {
	s := openTestStore(t)
	team, err := s.CreateTeam("eng", floatPtr(100.0), "", intPtr(1000))
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.BudgetPeriod != "monthly" {
		t.Errorf("BudgetPeriod = %q, want monthly", team.BudgetPeriod)
	}
}

func TestCreateVirtualKeyDefaultBudgetPeriod(t *testing.T) {
	s := openTestStore(t)
	key, _, err := s.CreateVirtualKey("vk", nil, floatPtr(100.0), "", nil, nil)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if key.BudgetPeriod != "monthly" {
		t.Errorf("BudgetPeriod = %q, want monthly", key.BudgetPeriod)
	}
}

func TestListTeamsWithAllFields(t *testing.T) {
	s := openTestStore(t)
	_, err := s.CreateTeam("a", floatPtr(50.0), "weekly", intPtr(100))
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	teams, err := s.ListTeams()
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("len(teams) = %d, want 1", len(teams))
	}
	if teams[0].BudgetUSD == nil || *teams[0].BudgetUSD != 50.0 {
		t.Errorf("BudgetUSD = %v, want 50.0", teams[0].BudgetUSD)
	}
	if teams[0].RateLimitRPM == nil || *teams[0].RateLimitRPM != 100 {
		t.Errorf("RateLimitRPM = %v, want 100", teams[0].RateLimitRPM)
	}
}

func TestListVirtualKeysWithAllFields(t *testing.T) {
	s := openTestStore(t)
	team, _ := s.CreateTeam("eng", floatPtr(1000.0), "monthly", intPtr(500))
	_, _, err := s.CreateVirtualKey("vk", &team.ID, floatPtr(100.0), "weekly", intPtr(60), intPtr(10000))
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	keys, err := s.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if keys[0].BudgetUSD == nil || *keys[0].BudgetUSD != 100.0 {
		t.Errorf("BudgetUSD = %v, want 100.0", keys[0].BudgetUSD)
	}
	if keys[0].RateLimitRPM == nil || *keys[0].RateLimitRPM != 60 {
		t.Errorf("RateLimitRPM = %v, want 60", keys[0].RateLimitRPM)
	}
	if keys[0].RateLimitTPM == nil || *keys[0].RateLimitTPM != 10000 {
		t.Errorf("RateLimitTPM = %v, want 10000", keys[0].RateLimitTPM)
	}
	if keys[0].TeamID == nil || *keys[0].TeamID != team.ID {
		t.Errorf("TeamID = %v, want %d", keys[0].TeamID, team.ID)
	}
}

func TestUpdateTeamNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.UpdateTeam(9999, "eng", nil, "monthly", nil)
	if err == nil {
		t.Fatal("expected error for missing team")
	}
}

func TestScanTeamBadBudgetResetAt(t *testing.T) {
	s := openTestStore(t)
	team, _ := s.CreateTeam("eng", floatPtr(100.0), "monthly", nil)
	s.db.Exec(`UPDATE teams SET budget_reset_at = 'not-a-date' WHERE id = ?`, team.ID)
	_, err := s.GetTeam(team.ID)
	if err == nil {
		t.Fatal("expected error for bad budget_reset_at")
	}
}

func TestScanTeamBadCreatedAt(t *testing.T) {
	s := openTestStore(t)
	team, _ := s.CreateTeam("eng", floatPtr(100.0), "monthly", nil)
	s.db.Exec(`UPDATE teams SET created_at = 'not-a-date' WHERE id = ?`, team.ID)
	_, err := s.GetTeam(team.ID)
	if err == nil {
		t.Fatal("expected error for bad created_at")
	}
}

func TestScanVirtualKeyBadBudgetResetAt(t *testing.T) {
	s := openTestStore(t)
	key, _, _ := s.CreateVirtualKey("vk", nil, floatPtr(100.0), "monthly", nil, nil)
	_, _ = s.db.Exec(`UPDATE virtual_keys SET budget_reset_at = 'not-a-date' WHERE id = ?`, key.ID)
	_, err := s.GetVirtualKey(key.ID)
	if err == nil {
		t.Fatal("expected error for bad budget_reset_at")
	}
}

func TestScanVirtualKeyBadCreatedAt(t *testing.T) {
	s := openTestStore(t)
	key, _, _ := s.CreateVirtualKey("vk", nil, floatPtr(100.0), "monthly", nil, nil)
	_, _ = s.db.Exec(`UPDATE virtual_keys SET created_at = 'not-a-date' WHERE id = ?`, key.ID)
	_, err := s.GetVirtualKey(key.ID)
	if err == nil {
		t.Fatal("expected error for bad created_at")
	}
}

func TestScanVirtualKeyListBadBudgetResetAt(t *testing.T) {
	s := openTestStore(t)
	_, _, _ = s.CreateVirtualKey("vk", nil, floatPtr(100.0), "monthly", nil, nil)
	s.db.Exec(`UPDATE virtual_keys SET budget_reset_at = 'not-a-date'`)
	_, err := s.ListVirtualKeys()
	if err == nil {
		t.Fatal("expected error for bad budget_reset_at in list")
	}
}

func TestScanVirtualKeyListBadCreatedAt(t *testing.T) {
	s := openTestStore(t)
	_, _, _ = s.CreateVirtualKey("vk", nil, floatPtr(100.0), "monthly", nil, nil)
	s.db.Exec(`UPDATE virtual_keys SET created_at = 'not-a-date'`)
	_, err := s.ListVirtualKeys()
	if err == nil {
		t.Fatal("expected error for bad created_at in list")
	}
}

func TestAddTeamBudgetUsedRowsAffectedError(t *testing.T) {
	s := openTestStore(t)
	// Force rows affected error by using a subquery that returns multiple rows
	// SQLite doesn't support this pattern for UPDATE, so we test via closing the DB
	s.Close()
	err := s.AddTeamBudgetUsed(1, 10.0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResetTeamBudgetRowsAffectedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.ResetTeamBudget(1, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddVirtualKeyBudgetUsedRowsAffectedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.AddVirtualKeyBudgetUsed(1, 10.0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResetVirtualKeyBudgetRowsAffectedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.ResetVirtualKeyBudget(1, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateVirtualKeyLastInsertIdError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateTeamLastInsertIdError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateVirtualKeyRowsAffectedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.UpdateVirtualKey(1, "vk", nil, nil, "monthly", nil, nil, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateTeamRowsAffectedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.UpdateTeam(1, "eng", nil, "monthly", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
