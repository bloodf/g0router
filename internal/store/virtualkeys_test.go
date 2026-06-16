package store

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

func TestVirtualKeyCRUD(t *testing.T) {
	st := newTestStore(t)

	vk1 := &VirtualKey{
		VirtualKey: schemas.VirtualKey{
			Name: "vk-1",
			ProviderConfigs: []schemas.ProviderConfig{
				{Provider: "openai", AllowedModels: []string{"gpt-4o"}, KeyIDs: []string{"conn-1"}, Weight: ptrFloat64(1)},
			},
			Budget:       &schemas.Budget{Limit: 10, Period: "daily", Used: 0},
			RateLimitRPM: ptrInt(60),
		},
	}
	created1, err := st.CreateVirtualKey(vk1)
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if created1.ID == "" {
		t.Fatal("created ID empty")
	}
	if created1.Key == "" {
		t.Fatal("created Key empty")
	}
	if created1.Name != "vk-1" {
		t.Fatalf("Name = %q, want %q", created1.Name, "vk-1")
	}

	got, err := st.GetVirtualKeyByID(created1.ID)
	if err != nil {
		t.Fatalf("GetVirtualKeyByID: %v", err)
	}
	if got.Name != created1.Name || got.Key != created1.Key {
		t.Fatalf("GetVirtualKeyByID mismatch: %+v vs %+v", got, created1)
	}
	if len(got.ProviderConfigs) != 1 || got.ProviderConfigs[0].Provider != "openai" {
		t.Fatalf("ProviderConfigs not round-tripped: %+v", got.ProviderConfigs)
	}
	if got.Budget == nil || got.Budget.Limit != 10 || got.Budget.Period != "daily" {
		t.Fatalf("Budget not round-tripped: %+v", got.Budget)
	}
	if got.RateLimitRPM == nil || *got.RateLimitRPM != 60 {
		t.Fatalf("RateLimitRPM not round-tripped: %+v", got.RateLimitRPM)
	}
	if !got.IsActive {
		t.Fatal("new virtual key should be active")
	}

	byKey, err := st.GetVirtualKeyByKey(created1.Key)
	if err != nil {
		t.Fatalf("GetVirtualKeyByKey: %v", err)
	}
	if byKey.ID != created1.ID {
		t.Fatalf("GetVirtualKeyByKey ID = %q, want %q", byKey.ID, created1.ID)
	}

	vk2 := &VirtualKey{
		VirtualKey: schemas.VirtualKey{
			Name: "vk-2",
			ProviderConfigs: []schemas.ProviderConfig{
				{Provider: "anthropic", AllowedModels: []string{"claude-3-opus"}, KeyIDs: []string{"conn-2"}},
			},
		},
	}
	created2, err := st.CreateVirtualKey(vk2)
	if err != nil {
		t.Fatalf("CreateVirtualKey second: %v", err)
	}

	list, err := st.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	updated := &VirtualKey{
		VirtualKey: schemas.VirtualKey{
			ID:     created1.ID,
			Name:   "vk-1-renamed",
			ProviderConfigs: []schemas.ProviderConfig{
				{Provider: "gemini", AllowedModels: []string{"gemini-pro"}, KeyIDs: []string{"conn-3"}},
			},
			Budget:       &schemas.Budget{Limit: 20, Period: "monthly", Used: 5},
			RateLimitRPM: ptrInt(100),
		},
		IsActive: false,
	}
	if err := st.UpdateVirtualKey(updated); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}

	got, err = st.GetVirtualKeyByID(created1.ID)
	if err != nil {
		t.Fatalf("GetVirtualKeyByID after update: %v", err)
	}
	if got.Name != "vk-1-renamed" {
		t.Fatalf("Name after update = %q, want %q", got.Name, "vk-1-renamed")
	}
	if got.IsActive {
		t.Fatal("virtual key should be inactive after update")
	}
	if got.ProviderConfigs[0].Provider != "gemini" {
		t.Fatalf("ProviderConfigs not updated: %+v", got.ProviderConfigs)
	}
	if got.Budget.Period != "monthly" || got.Budget.Used != 5 {
		t.Fatalf("Budget not updated: %+v", got.Budget)
	}
	if got.RateLimitRPM == nil || *got.RateLimitRPM != 100 {
		t.Fatalf("RateLimitRPM not updated: %+v", got.RateLimitRPM)
	}

	if err := st.DeleteVirtualKey(created2.ID); err != nil {
		t.Fatalf("DeleteVirtualKey: %v", err)
	}
	if _, err := st.GetVirtualKeyByID(created2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted vk err = %v, want ErrNotFound", err)
	}

	list, err = st.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) after delete = %d, want 1", len(list))
	}

	// Duplicate key value rejected. The `key` column now holds sha256hex(raw)
	// (bf-gov-5), so colliding on the UNIQUE constraint requires inserting the
	// hash of an existing VK's raw key, not the raw key itself.
	id, err := newID()
	if err != nil {
		t.Fatalf("newID: %v", err)
	}
	_, err = st.DB().Exec(
		"INSERT INTO virtual_keys (id, key, name, config_json, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, sha256hex(created1.Key), "duplicate", "{}", 1, created1.CreatedAt, created1.UpdatedAt,
	)
	if err == nil {
		t.Fatal("duplicate key value accepted")
	}

	// Unknown id returns ErrNotFound.
	if err := st.DeleteVirtualKey("vk-nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown err = %v, want ErrNotFound", err)
	}
	if _, err := st.GetVirtualKeyByID("vk-nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get unknown err = %v, want ErrNotFound", err)
	}
}

// TestVirtualKeyTeamIDRoundTrip verifies the additive team_id column round-trips
// through Create/Get/List/Update, and that a VK created without a team has an
// empty team_id (the empty-string "un-teamed" sentinel, D4).
func TestVirtualKeyTeamIDRoundTrip(t *testing.T) {
	st := newTestStore(t)

	// Created with a team_id round-trips.
	withTeam := &VirtualKey{
		VirtualKey: schemas.VirtualKey{
			Name: "vk-teamed",
			ProviderConfigs: []schemas.ProviderConfig{
				{Provider: "openai", AllowedModels: []string{"gpt-4o"}},
			},
		},
		TeamID: "team-123",
	}
	created, err := st.CreateVirtualKey(withTeam)
	if err != nil {
		t.Fatalf("CreateVirtualKey with team: %v", err)
	}
	if created.TeamID != "team-123" {
		t.Fatalf("created TeamID = %q, want %q", created.TeamID, "team-123")
	}

	got, err := st.GetVirtualKeyByID(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKeyByID: %v", err)
	}
	if got.TeamID != "team-123" {
		t.Fatalf("GetVirtualKeyByID TeamID = %q, want %q", got.TeamID, "team-123")
	}

	byKey, err := st.GetVirtualKeyByKey(created.Key)
	if err != nil {
		t.Fatalf("GetVirtualKeyByKey: %v", err)
	}
	if byKey.TeamID != "team-123" {
		t.Fatalf("GetVirtualKeyByKey TeamID = %q, want %q", byKey.TeamID, "team-123")
	}

	// Created without a team has an empty team_id.
	noTeam := &VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "vk-unteamed"},
	}
	createdNoTeam, err := st.CreateVirtualKey(noTeam)
	if err != nil {
		t.Fatalf("CreateVirtualKey without team: %v", err)
	}
	if createdNoTeam.TeamID != "" {
		t.Fatalf("un-teamed created TeamID = %q, want empty", createdNoTeam.TeamID)
	}
	gotNoTeam, err := st.GetVirtualKeyByID(createdNoTeam.ID)
	if err != nil {
		t.Fatalf("GetVirtualKeyByID un-teamed: %v", err)
	}
	if gotNoTeam.TeamID != "" {
		t.Fatalf("un-teamed TeamID = %q, want empty", gotNoTeam.TeamID)
	}

	// List preserves team_id.
	list, err := st.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	for _, vk := range list {
		if vk.ID == created.ID && vk.TeamID != "team-123" {
			t.Fatalf("list TeamID for teamed vk = %q, want %q", vk.TeamID, "team-123")
		}
	}

	// Update mutates team_id.
	got.TeamID = "team-456"
	if err := st.UpdateVirtualKey(got); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}
	reread, err := st.GetVirtualKeyByID(created.ID)
	if err != nil {
		t.Fatalf("GetVirtualKeyByID after update: %v", err)
	}
	if reread.TeamID != "team-456" {
		t.Fatalf("TeamID after update = %q, want %q", reread.TeamID, "team-456")
	}
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }
