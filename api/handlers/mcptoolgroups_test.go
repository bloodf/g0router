package handlers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeMCPToolGroupStore struct {
	groups []store.MCPToolGroup
	nextID int64
}

func (f *fakeMCPToolGroupStore) ListMCPToolGroups() ([]store.MCPToolGroup, error) {
	return f.groups, nil
}

func (f *fakeMCPToolGroupStore) GetMCPToolGroup(id int64) (*store.MCPToolGroup, error) {
	for i := range f.groups {
		if f.groups[i].ID == id {
			return &f.groups[i], nil
		}
	}
	return nil, store.ErrNotFound
}

func (f *fakeMCPToolGroupStore) CreateMCPToolGroup(name string, toolIDs []string, isActive bool) (*store.MCPToolGroup, error) {
	f.nextID++
	g := store.MCPToolGroup{
		ID: f.nextID, Name: name, ToolIDs: toolIDs, IsActive: isActive,
		CreatedAt: "now", UpdatedAt: "now",
	}
	f.groups = append(f.groups, g)
	return &g, nil
}

func (f *fakeMCPToolGroupStore) UpdateMCPToolGroup(id int64, name string, toolIDs []string, isActive bool) error {
	for i := range f.groups {
		if f.groups[i].ID == id {
			f.groups[i].Name = name
			f.groups[i].ToolIDs = toolIDs
			f.groups[i].IsActive = isActive
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakeMCPToolGroupStore) DeleteMCPToolGroup(id int64) error {
	for i := range f.groups {
		if f.groups[i].ID == id {
			f.groups = append(f.groups[:i], f.groups[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

func TestMCPToolGroupsList(t *testing.T) {
	s := &fakeMCPToolGroupStore{
		groups: []store.MCPToolGroup{
			{ID: 1, Name: "g1", ToolIDs: []string{"t1"}, IsActive: true},
		},
	}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/mcp/tool-groups", nil)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusOK)

	var resp listResponse[mcpToolGroupResponse]
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len = %d, want 1", len(resp.Data))
	}
}

func TestMCPToolGroupsGet(t *testing.T) {
	s := &fakeMCPToolGroupStore{
		groups: []store.MCPToolGroup{
			{ID: 1, Name: "g1", ToolIDs: []string{"t1"}, IsActive: true},
		},
	}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/mcp/tool-groups/1", nil)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusOK)

	var resp mcpToolGroupResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("id = %d, want 1", resp.ID)
	}
}

func TestMCPToolGroupsGetNotFound(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/mcp/tool-groups/1", nil)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNotFound)
}

func TestMCPToolGroupsCreate(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	body, _ := json.Marshal(mcpToolGroupRequest{
		Name: "g1", ToolIDs: []string{"t1"}, IsActive: true,
	})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/mcp/tool-groups", body)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusCreated)

	var resp mcpToolGroupResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Name != "g1" {
		t.Errorf("name = %q, want g1", resp.Name)
	}
}

func TestMCPToolGroupsCreateMissingName(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	body, _ := json.Marshal(mcpToolGroupRequest{ToolIDs: []string{"t1"}})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/mcp/tool-groups", body)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestMCPToolGroupsCreateDuplicateName(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	body, _ := json.Marshal(mcpToolGroupRequest{Name: "g1"})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/mcp/tool-groups", body)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusCreated)

	s2 := &fakeMCPToolGroupStoreWithErrors{createErr: store.ErrDuplicateName}
	ctx2 := newTestCtx(fasthttp.MethodPost, "/api/mcp/tool-groups", body)
	MCPToolGroups(ctx2, s2, "")
	assertStatus(t, ctx2, fasthttp.StatusConflict)
}

func TestMCPToolGroupsUpdate(t *testing.T) {
	s := &fakeMCPToolGroupStore{
		groups: []store.MCPToolGroup{
			{ID: 1, Name: "g1", ToolIDs: []string{"t1"}, IsActive: true},
		},
	}
	body, _ := json.Marshal(mcpToolGroupRequest{
		Name: "g1-updated", ToolIDs: []string{"t2"}, IsActive: false,
	})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/mcp/tool-groups/1", body)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNoContent)
}

func TestMCPToolGroupsUpdateNotFound(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	body, _ := json.Marshal(mcpToolGroupRequest{Name: "g1", ToolIDs: []string{"t1"}})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/mcp/tool-groups/1", body)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNotFound)
}

func TestMCPToolGroupsDelete(t *testing.T) {
	s := &fakeMCPToolGroupStore{
		groups: []store.MCPToolGroup{
			{ID: 1, Name: "g1", ToolIDs: []string{"t1"}, IsActive: true},
		},
	}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/mcp/tool-groups/1", nil)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNoContent)
	if len(s.groups) != 0 {
		t.Errorf("groups len = %d, want 0", len(s.groups))
	}
}

func TestMCPToolGroupsDeleteNotFound(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/mcp/tool-groups/1", nil)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusNotFound)
}

func TestMCPToolGroupsNilStore(t *testing.T) {
	ctx := newTestCtx(fasthttp.MethodGet, "/api/mcp/tool-groups", nil)
	MCPToolGroups(ctx, nil, "")
	assertStatus(t, ctx, fasthttp.StatusServiceUnavailable)
}

func TestMCPToolGroupsInvalidMethod(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodPatch, "/api/mcp/tool-groups", nil)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusMethodNotAllowed)
}

func TestMCPToolGroupsCreateInvalidJSON(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodPost, "/api/mcp/tool-groups", []byte("not-json"))
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestMCPToolGroupsUpdateInvalidJSON(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodPut, "/api/mcp/tool-groups/1", []byte("not-json"))
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestMCPToolGroupsUpdateMissingID(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	body, _ := json.Marshal(mcpToolGroupRequest{Name: "g1"})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/mcp/tool-groups", body)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestMCPToolGroupsDeleteMissingID(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/mcp/tool-groups", nil)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

func TestMCPToolGroupsGetInvalidID(t *testing.T) {
	s := &fakeMCPToolGroupStore{}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/mcp/tool-groups/abc", nil)
	MCPToolGroups(ctx, s, "abc")
	assertStatus(t, ctx, fasthttp.StatusBadRequest)
}

type fakeMCPToolGroupStoreWithErrors struct {
	fakeMCPToolGroupStore
	listErr   error
	createErr error
	updateErr error
	deleteErr error
}

func (f *fakeMCPToolGroupStoreWithErrors) ListMCPToolGroups() ([]store.MCPToolGroup, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.fakeMCPToolGroupStore.ListMCPToolGroups()
}

func (f *fakeMCPToolGroupStoreWithErrors) CreateMCPToolGroup(name string, toolIDs []string, isActive bool) (*store.MCPToolGroup, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.fakeMCPToolGroupStore.CreateMCPToolGroup(name, toolIDs, isActive)
}

func (f *fakeMCPToolGroupStoreWithErrors) UpdateMCPToolGroup(id int64, name string, toolIDs []string, isActive bool) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	return f.fakeMCPToolGroupStore.UpdateMCPToolGroup(id, name, toolIDs, isActive)
}

func (f *fakeMCPToolGroupStoreWithErrors) DeleteMCPToolGroup(id int64) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.fakeMCPToolGroupStore.DeleteMCPToolGroup(id)
}

func TestMCPToolGroupsListError(t *testing.T) {
	s := &fakeMCPToolGroupStoreWithErrors{listErr: errors.New("boom")}
	ctx := newTestCtx(fasthttp.MethodGet, "/api/mcp/tool-groups", nil)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestMCPToolGroupsCreateError(t *testing.T) {
	s := &fakeMCPToolGroupStoreWithErrors{createErr: errors.New("boom")}
	body, _ := json.Marshal(mcpToolGroupRequest{Name: "g1", ToolIDs: []string{"t1"}})
	ctx := newTestCtx(fasthttp.MethodPost, "/api/mcp/tool-groups", body)
	MCPToolGroups(ctx, s, "")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestMCPToolGroupsUpdateError(t *testing.T) {
	s := &fakeMCPToolGroupStoreWithErrors{updateErr: errors.New("boom")}
	body, _ := json.Marshal(mcpToolGroupRequest{Name: "g1", ToolIDs: []string{"t1"}})
	ctx := newTestCtx(fasthttp.MethodPut, "/api/mcp/tool-groups/1", body)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}

func TestMCPToolGroupsDeleteError(t *testing.T) {
	s := &fakeMCPToolGroupStoreWithErrors{deleteErr: errors.New("boom")}
	ctx := newTestCtx(fasthttp.MethodDelete, "/api/mcp/tool-groups/1", nil)
	MCPToolGroups(ctx, s, "1")
	assertStatus(t, ctx, fasthttp.StatusInternalServerError)
}
