package store

import (
	"errors"
	"testing"
)

func TestTeamCRUD(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateTeam(&Team{
		Name:         "Engineering",
		BudgetUSD:    2000,
		BudgetPeriod: "monthly",
		RateLimitRPM: 5000,
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created ID empty")
	}
	if created.Name != "Engineering" {
		t.Fatalf("Name = %q, want %q", created.Name, "Engineering")
	}
	if created.BudgetUSD != 2000 || created.BudgetPeriod != "monthly" || created.RateLimitRPM != 5000 {
		t.Fatalf("create fields not persisted: %+v", created)
	}
	if created.BudgetUsedUSD != 0 {
		t.Fatalf("BudgetUsedUSD = %v, want 0", created.BudgetUsedUSD)
	}
	if created.CreatedAt == 0 || created.UpdatedAt == 0 {
		t.Fatalf("timestamps not set: %+v", created)
	}

	got, err := st.GetTeamByID(created.ID)
	if err != nil {
		t.Fatalf("GetTeamByID: %v", err)
	}
	if got.Name != created.Name || got.BudgetUSD != created.BudgetUSD {
		t.Fatalf("GetTeamByID mismatch: %+v vs %+v", got, created)
	}

	created2, err := st.CreateTeam(&Team{Name: "Data Science", BudgetUSD: 1500, BudgetPeriod: "monthly", RateLimitRPM: 2000})
	if err != nil {
		t.Fatalf("CreateTeam second: %v", err)
	}

	list, err := st.ListTeams()
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	if err := st.UpdateTeam(&Team{
		ID:           created.ID,
		Name:         "Engineering-renamed",
		BudgetUSD:    3000,
		BudgetPeriod: "weekly",
		RateLimitRPM: 6000,
	}); err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	got, err = st.GetTeamByID(created.ID)
	if err != nil {
		t.Fatalf("GetTeamByID after update: %v", err)
	}
	if got.Name != "Engineering-renamed" || got.BudgetUSD != 3000 || got.BudgetPeriod != "weekly" || got.RateLimitRPM != 6000 {
		t.Fatalf("update not persisted: %+v", got)
	}

	if err := st.DeleteTeam(created2.ID); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	if _, err := st.GetTeamByID(created2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted team err = %v, want ErrNotFound", err)
	}

	list, err = st.ListTeams()
	if err != nil {
		t.Fatalf("ListTeams after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) after delete = %d, want 1", len(list))
	}

	// Unknown id returns ErrNotFound.
	if err := st.DeleteTeam("team-nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown err = %v, want ErrNotFound", err)
	}
	if err := st.UpdateTeam(&Team{ID: "team-nonexistent", Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update unknown err = %v, want ErrNotFound", err)
	}
	if _, err := st.GetTeamByID("team-nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get unknown err = %v, want ErrNotFound", err)
	}
}
