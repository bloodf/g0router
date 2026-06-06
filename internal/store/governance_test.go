package store

import (
	"strings"
	"testing"
	"time"
)

func TestCreateTeam(t *testing.T) {
	s := openTestStore(t)

	team, err := s.CreateTeam("engineering", floatPtr(100.0), "monthly", intPtr(1000))
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.ID == 0 {
		t.Error("ID should be set")
	}
	if team.Name != "engineering" {
		t.Errorf("Name = %q, want engineering", team.Name)
	}
	if team.BudgetUSD == nil || *team.BudgetUSD != 100.0 {
		t.Errorf("BudgetUSD = %v, want 100.0", team.BudgetUSD)
	}
	if team.BudgetPeriod != "monthly" {
		t.Errorf("BudgetPeriod = %q, want monthly", team.BudgetPeriod)
	}
	if team.BudgetUsedUSD != 0 {
		t.Errorf("BudgetUsedUSD = %f, want 0", team.BudgetUsedUSD)
	}
	if team.RateLimitRPM == nil || *team.RateLimitRPM != 1000 {
		t.Errorf("RateLimitRPM = %v, want 1000", team.RateLimitRPM)
	}
}

func TestGetTeam(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateTeam("engineering", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	got, err := s.GetTeam(created.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.Name != "engineering" {
		t.Errorf("Name = %q, want engineering", got.Name)
	}
}

func TestGetTeamNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetTeam(999)
	if err == nil {
		t.Fatal("GetTeam should error for missing id")
	}
}

func TestListTeams(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateTeam("a", nil, "monthly", nil); err != nil {
		t.Fatalf("CreateTeam a: %v", err)
	}
	if _, err := s.CreateTeam("b", nil, "monthly", nil); err != nil {
		t.Fatalf("CreateTeam b: %v", err)
	}

	teams, err := s.ListTeams()
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("len(teams) = %d, want 2", len(teams))
	}
}

func TestUpdateTeam(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateTeam("old", floatPtr(50.0), "daily", intPtr(100))
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if err := s.UpdateTeam(created.ID, "new", floatPtr(200.0), "weekly", intPtr(500)); err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}

	got, err := s.GetTeam(created.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Name = %q, want new", got.Name)
	}
	if got.BudgetUSD == nil || *got.BudgetUSD != 200.0 {
		t.Errorf("BudgetUSD = %v, want 200.0", got.BudgetUSD)
	}
	if got.BudgetPeriod != "weekly" {
		t.Errorf("BudgetPeriod = %q, want weekly", got.BudgetPeriod)
	}
	if got.RateLimitRPM == nil || *got.RateLimitRPM != 500 {
		t.Errorf("RateLimitRPM = %v, want 500", got.RateLimitRPM)
	}
}

func TestDeleteTeam(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateTeam("temp", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if err := s.DeleteTeam(created.ID); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}

	_, err = s.GetTeam(created.ID)
	if err == nil {
		t.Fatal("GetTeam should error after delete")
	}
}

func TestCreateVirtualKey(t *testing.T) {
	s := openTestStore(t)

	key, raw, err := s.CreateVirtualKey("prod-key", nil, floatPtr(10.0), "monthly", intPtr(60), intPtr(10000), "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if key.ID == 0 {
		t.Error("ID should be set")
	}
	if key.Name != "prod-key" {
		t.Errorf("Name = %q, want prod-key", key.Name)
	}
	if !strings.HasPrefix(raw, "gvk-") {
		t.Fatalf("raw = %q, want gvk- prefix", raw)
	}
	if key.KeyPrefix != raw[:8] {
		t.Errorf("KeyPrefix = %q, want %q", key.KeyPrefix, raw[:8])
	}
	if !key.IsActive {
		t.Error("IsActive should be true")
	}
	if key.BudgetUSD == nil || *key.BudgetUSD != 10.0 {
		t.Errorf("BudgetUSD = %v, want 10.0", key.BudgetUSD)
	}
	if key.RateLimitRPM == nil || *key.RateLimitRPM != 60 {
		t.Errorf("RateLimitRPM = %v, want 60", key.RateLimitRPM)
	}
	if key.RateLimitTPM == nil || *key.RateLimitTPM != 10000 {
		t.Errorf("RateLimitTPM = %v, want 10000", key.RateLimitTPM)
	}
}

func TestValidateVirtualKeyCorrect(t *testing.T) {
	s := openTestStore(t)

	created, raw, err := s.CreateVirtualKey("prod-key", nil, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	got, ok, err := s.ValidateVirtualKey(raw)
	if err != nil {
		t.Fatalf("ValidateVirtualKey: %v", err)
	}
	if !ok {
		t.Fatal("ValidateVirtualKey should return ok")
	}
	if got == nil {
		t.Fatal("ValidateVirtualKey returned nil")
	}
	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.Name != "prod-key" {
		t.Errorf("Name = %q, want prod-key", got.Name)
	}
}

func TestValidateVirtualKeyWrong(t *testing.T) {
	s := openTestStore(t)

	if _, _, err := s.CreateVirtualKey("prod-key", nil, nil, "monthly", nil, nil, ""); err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	got, ok, err := s.ValidateVirtualKey("gvk-wrong")
	if err != nil {
		t.Fatalf("ValidateVirtualKey: %v", err)
	}
	if ok {
		t.Fatal("ValidateVirtualKey should not return ok")
	}
	if got != nil {
		t.Fatalf("got = %+v, want nil", got)
	}
}

func TestValidateVirtualKeyInactive(t *testing.T) {
	s := openTestStore(t)

	created, raw, err := s.CreateVirtualKey("prod-key", nil, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if err := s.UpdateVirtualKey(created.ID, "prod-key", nil, nil, "monthly", nil, nil, false, ""); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}

	got, ok, err := s.ValidateVirtualKey(raw)
	if err != nil {
		t.Fatalf("ValidateVirtualKey: %v", err)
	}
	if !ok {
		t.Fatal("ValidateVirtualKey should return ok even for inactive key (governance enforces active status)")
	}
	if got == nil {
		t.Fatal("ValidateVirtualKey should return the key")
	}
	if got.IsActive {
		t.Error("IsActive should be false")
	}
}

func TestListVirtualKeys(t *testing.T) {
	s := openTestStore(t)

	created, raw, err := s.CreateVirtualKey("prod-key", nil, nil, "monthly", nil, nil, "")
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

	got := keys[0]
	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.KeyPrefix != raw[:8] {
		t.Errorf("KeyPrefix = %q, want %q", got.KeyPrefix, raw[:8])
	}
	if got.KeyHash != "" {
		t.Error("ListVirtualKeys should not expose key hash")
	}
	if strings.Contains(got.KeyPrefix, raw[8:]) {
		t.Error("listed key should not expose raw key material")
	}
}

func TestUpdateVirtualKey(t *testing.T) {
	s := openTestStore(t)

	team, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	created, _, err := s.CreateVirtualKey("old", nil, floatPtr(5.0), "daily", intPtr(10), intPtr(100), "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	if err := s.UpdateVirtualKey(created.ID, "new", &team.ID, floatPtr(15.0), "weekly", intPtr(20), intPtr(200), false, ""); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}

	got, err := s.GetVirtualKey(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKey: %v", err)
	}
	if got.Name != "new" {
		t.Errorf("Name = %q, want new", got.Name)
	}
	if got.TeamID == nil || *got.TeamID != team.ID {
		t.Errorf("TeamID = %v, want %d", got.TeamID, team.ID)
	}
	if got.BudgetUSD == nil || *got.BudgetUSD != 15.0 {
		t.Errorf("BudgetUSD = %v, want 15.0", got.BudgetUSD)
	}
	if got.BudgetPeriod != "weekly" {
		t.Errorf("BudgetPeriod = %q, want weekly", got.BudgetPeriod)
	}
	if got.RateLimitRPM == nil || *got.RateLimitRPM != 20 {
		t.Errorf("RateLimitRPM = %v, want 20", got.RateLimitRPM)
	}
	if got.RateLimitTPM == nil || *got.RateLimitTPM != 200 {
		t.Errorf("RateLimitTPM = %v, want 200", got.RateLimitTPM)
	}
	if got.IsActive {
		t.Error("IsActive should be false")
	}
}

func TestDeleteVirtualKey(t *testing.T) {
	s := openTestStore(t)

	created, _, err := s.CreateVirtualKey("temp", nil, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	if err := s.DeleteVirtualKey(created.ID); err != nil {
		t.Fatalf("DeleteVirtualKey: %v", err)
	}

	_, err = s.GetVirtualKey(created.ID)
	if err == nil {
		t.Fatal("GetVirtualKey should error after delete")
	}
}

func TestAddVirtualKeyBudgetUsed(t *testing.T) {
	s := openTestStore(t)

	created, _, err := s.CreateVirtualKey("key", nil, floatPtr(100.0), "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	if err := s.AddVirtualKeyBudgetUsed(created.ID, 12.5); err != nil {
		t.Fatalf("AddVirtualKeyBudgetUsed: %v", err)
	}

	got, err := s.GetVirtualKey(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKey: %v", err)
	}
	if got.BudgetUsedUSD != 12.5 {
		t.Errorf("BudgetUsedUSD = %f, want 12.5", got.BudgetUsedUSD)
	}

	if err := s.AddVirtualKeyBudgetUsed(created.ID, 3.5); err != nil {
		t.Fatalf("AddVirtualKeyBudgetUsed: %v", err)
	}

	got, err = s.GetVirtualKey(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKey: %v", err)
	}
	if got.BudgetUsedUSD != 16.0 {
		t.Errorf("BudgetUsedUSD = %f, want 16.0", got.BudgetUsedUSD)
	}
}

func TestResetVirtualKeyBudget(t *testing.T) {
	s := openTestStore(t)

	created, _, err := s.CreateVirtualKey("key", nil, floatPtr(100.0), "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if err := s.AddVirtualKeyBudgetUsed(created.ID, 50.0); err != nil {
		t.Fatalf("AddVirtualKeyBudgetUsed: %v", err)
	}

	nextReset := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	if err := s.ResetVirtualKeyBudget(created.ID, nextReset); err != nil {
		t.Fatalf("ResetVirtualKeyBudget: %v", err)
	}

	got, err := s.GetVirtualKey(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKey: %v", err)
	}
	if got.BudgetUsedUSD != 0 {
		t.Errorf("BudgetUsedUSD = %f, want 0", got.BudgetUsedUSD)
	}
	if got.BudgetResetAt == nil || !got.BudgetResetAt.Equal(nextReset) {
		t.Errorf("BudgetResetAt = %v, want %v", got.BudgetResetAt, nextReset)
	}
}

func TestAddTeamBudgetUsed(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateTeam("eng", floatPtr(1000.0), "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if err := s.AddTeamBudgetUsed(created.ID, 100.0); err != nil {
		t.Fatalf("AddTeamBudgetUsed: %v", err)
	}
	if err := s.AddTeamBudgetUsed(created.ID, 50.0); err != nil {
		t.Fatalf("AddTeamBudgetUsed: %v", err)
	}

	got, err := s.GetTeam(created.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.BudgetUsedUSD != 150.0 {
		t.Errorf("BudgetUsedUSD = %f, want 150.0", got.BudgetUsedUSD)
	}
}

func TestResetTeamBudget(t *testing.T) {
	s := openTestStore(t)

	created, err := s.CreateTeam("eng", floatPtr(1000.0), "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if err := s.AddTeamBudgetUsed(created.ID, 500.0); err != nil {
		t.Fatalf("AddTeamBudgetUsed: %v", err)
	}

	nextReset := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	if err := s.ResetTeamBudget(created.ID, nextReset); err != nil {
		t.Fatalf("ResetTeamBudget: %v", err)
	}

	got, err := s.GetTeam(created.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.BudgetUsedUSD != 0 {
		t.Errorf("BudgetUsedUSD = %f, want 0", got.BudgetUsedUSD)
	}
	if got.BudgetResetAt == nil || !got.BudgetResetAt.Equal(nextReset) {
		t.Errorf("BudgetResetAt = %v, want %v", got.BudgetResetAt, nextReset)
	}
}

func TestVirtualKeyTeamAssociation(t *testing.T) {
	s := openTestStore(t)

	team, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	key, _, err := s.CreateVirtualKey("team-key", &team.ID, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	got, err := s.GetVirtualKey(key.ID)
	if err != nil {
		t.Fatalf("GetVirtualKey: %v", err)
	}
	if got.TeamID == nil || *got.TeamID != team.ID {
		t.Errorf("TeamID = %v, want %d", got.TeamID, team.ID)
	}
}

func TestListVirtualKeysOmitsHash(t *testing.T) {
	s := openTestStore(t)

	if _, _, err := s.CreateVirtualKey("k1", nil, nil, "monthly", nil, nil, ""); err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	keys, err := s.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	for _, k := range keys {
		if k.KeyHash != "" {
			t.Errorf("ListVirtualKeys exposed key_hash for id %d", k.ID)
		}
	}
}

func TestCreateVirtualKeyDuplicateNameAllowed(t *testing.T) {
	s := openTestStore(t)

	if _, _, err := s.CreateVirtualKey("same", nil, nil, "monthly", nil, nil, ""); err != nil {
		t.Fatalf("first CreateVirtualKey: %v", err)
	}
	if _, _, err := s.CreateVirtualKey("same", nil, nil, "monthly", nil, nil, ""); err != nil {
		t.Fatalf("second CreateVirtualKey should allow duplicate names: %v", err)
	}
}

func TestCreateTeamDuplicateName(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateTeam("same", nil, "monthly", nil); err != nil {
		t.Fatalf("first CreateTeam: %v", err)
	}
	if _, err := s.CreateTeam("same", nil, "monthly", nil); err == nil {
		t.Fatal("second CreateTeam should fail")
	}
}

func TestGetVirtualKeyNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetVirtualKey(999)
	if err == nil {
		t.Fatal("GetVirtualKey should error for missing id")
	}
}
