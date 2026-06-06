package store

import (
	"testing"
	"time"
)

func TestCreateTeamClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestGetTeamError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.GetTeam(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestListTeamsError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.ListTeams()
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestUpdateTeamError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.UpdateTeam(1, "eng", nil, "monthly", nil)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestDeleteTeamError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.DeleteTeam(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestAddTeamBudgetUsedZeroDelta(t *testing.T) {
	s := openTestStore(t)
	team, _ := s.CreateTeam("eng", floatPtr(100.0), "monthly", nil)
	err := s.AddTeamBudgetUsed(team.ID, 0)
	if err != nil {
		t.Fatalf("zero delta should return nil: %v", err)
	}
}

func TestAddTeamBudgetUsedNegativeDelta(t *testing.T) {
	s := openTestStore(t)
	team, _ := s.CreateTeam("eng", floatPtr(100.0), "monthly", nil)
	err := s.AddTeamBudgetUsed(team.ID, -5.0)
	if err != nil {
		t.Fatalf("negative delta should return nil: %v", err)
	}
}

func TestAddTeamBudgetUsedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.AddTeamBudgetUsed(1, 10.0)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestAddTeamBudgetUsedNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.AddTeamBudgetUsed(9999, 10.0)
	if err == nil {
		t.Fatal("expected error for missing team")
	}
}

func TestResetTeamBudgetError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.ResetTeamBudget(1, time.Now())
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestResetTeamBudgetNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.ResetTeamBudget(9999, time.Now())
	if err == nil {
		t.Fatal("expected error for missing team")
	}
}

func TestCreateVirtualKeyClosedDB(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, _, err := s.CreateVirtualKey("vk", nil, nil, "monthly", nil, nil)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestGetVirtualKeyError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.GetVirtualKey(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestListVirtualKeysError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, err := s.ListVirtualKeys()
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestValidateVirtualKeyError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	_, _, err := s.ValidateVirtualKey("gvk-test")
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestUpdateVirtualKeyError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.UpdateVirtualKey(1, "vk", nil, nil, "monthly", nil, nil, true)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestUpdateVirtualKeyNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.UpdateVirtualKey(9999, "vk", nil, nil, "monthly", nil, nil, true)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestDeleteVirtualKeyError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.DeleteVirtualKey(1)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestAddVirtualKeyBudgetUsedZeroDelta(t *testing.T) {
	s := openTestStore(t)
	key, _, _ := s.CreateVirtualKey("vk", nil, floatPtr(100.0), "monthly", nil, nil)
	err := s.AddVirtualKeyBudgetUsed(key.ID, 0)
	if err != nil {
		t.Fatalf("zero delta should return nil: %v", err)
	}
}

func TestAddVirtualKeyBudgetUsedNegativeDelta(t *testing.T) {
	s := openTestStore(t)
	key, _, _ := s.CreateVirtualKey("vk", nil, floatPtr(100.0), "monthly", nil, nil)
	err := s.AddVirtualKeyBudgetUsed(key.ID, -5.0)
	if err != nil {
		t.Fatalf("negative delta should return nil: %v", err)
	}
}

func TestAddVirtualKeyBudgetUsedError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.AddVirtualKeyBudgetUsed(1, 10.0)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestAddVirtualKeyBudgetUsedNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.AddVirtualKeyBudgetUsed(9999, 10.0)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestResetVirtualKeyBudgetError(t *testing.T) {
	s := openTestStore(t)
	s.Close()
	err := s.ResetVirtualKeyBudget(1, time.Now())
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

func TestResetVirtualKeyBudgetNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.ResetVirtualKeyBudget(9999, time.Now())
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}
