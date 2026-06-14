package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestAlertChannelCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	actor := map[string]any{userKey: admin}

	// Create.
	status, envl := call(t, env.handlers.CreateAlertChannel, "POST", "/api/alert-channels",
		`{"name":"Webhook Alerts","channel_type":"webhook","config":{"url":"https://hooks.example.com/g0router"},"events":["quota_exceeded"],"is_active":true}`,
		actor, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	idF, ok := created["id"].(float64)
	if !ok {
		t.Fatalf("id is not numeric: %v", created["id"])
	}
	if created["name"] != "Webhook Alerts" || created["channel_type"] != "webhook" {
		t.Fatalf("created = %v", created)
	}
	if created["is_active"] != true {
		t.Fatalf("is_active = %v", created["is_active"])
	}
	cfg, _ := created["config"].(map[string]any)
	if cfg["url"] != "https://hooks.example.com/g0router" {
		t.Fatalf("config not echoed for edit form: %v", created["config"])
	}
	id := "1"
	_ = idF

	// Empty name → 400.
	status, _ = call(t, env.handlers.CreateAlertChannel, "POST", "/api/alert-channels",
		`{"name":"","channel_type":"webhook"}`, actor, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty-name create status = %d, want 400", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListAlertChannels, "GET", "/api/alert-channels", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	// Get.
	status, envl = call(t, env.handlers.GetAlertChannel, "GET", "/api/alert-channels/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	if dataField[map[string]any](t, envl)["name"] != "Webhook Alerts" {
		t.Fatalf("get name mismatch")
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetAlertChannel, "GET", "/api/alert-channels/99", "",
		map[string]any{"id": "99"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d, want 404", status)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateAlertChannel, "PUT", "/api/alert-channels/"+id,
		`{"name":"Webhook v2","channel_type":"webhook","config":{"url":"https://new.example.com"},"events":["provider_error"],"is_active":false}`,
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	if dataField[map[string]any](t, envl)["name"] != "Webhook v2" {
		t.Fatalf("update name mismatch")
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateAlertChannel, "PUT", "/api/alert-channels/99",
		`{"name":"x"}`, map[string]any{"id": "99", userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d, want 404", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteAlertChannel, "DELETE", "/api/alert-channels/"+id, "",
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteAlertChannel, "DELETE", "/api/alert-channels/"+id, "",
		map[string]any{"id": id, userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}
}

func TestCreateAlertChannelWritesAudit(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	status, _ := call(t, env.handlers.CreateAlertChannel, "POST", "/api/alert-channels",
		`{"name":"Audited","channel_type":"webhook","config":{"url":"https://x"}}`,
		map[string]any{userKey: admin}, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	status, envl := call(t, env.handlers.GetAudit, "GET", "/api/audit", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("audit status = %d", status)
	}
	page := dataField[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, envl)
	if page.Total < 1 {
		t.Fatalf("expected an audit entry after alert-channel create, total=%d", page.Total)
	}
	if page.Items[0]["actor"] != "admin" || page.Items[0]["target"] != "Audited" {
		t.Fatalf("audit entry = %v", page.Items[0])
	}
}

func TestTestAlertChannelDeterministicAndNoSecretLeak(t *testing.T) {
	env := newTestEnv(t)
	admin, err := env.store.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}

	// Deterministic loopback target (no external network); the URL path embeds a
	// secret so the no-leak assertion is meaningful.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	secretURL := srv.URL + "/SUPERSECRETTOKEN"
	created, err := env.store.CreateAlertChannel(&store.AlertChannel{
		Name:        "Webhook",
		ChannelType: "webhook",
		Config:      map[string]any{"url": secretURL, "token": "tok-XYZSECRET"},
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}
	idStr := strconv.FormatInt(created.ID, 10)

	status, envl := call(t, env.handlers.TestAlertChannel, "POST", "/api/alert-channels/"+idStr+"/test", "",
		map[string]any{"id": idStr, userKey: admin}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("test status = %d err = %q", status, errMessage(t, envl))
	}
	out := dataField[map[string]any](t, envl)
	if out["ok"] != true {
		t.Fatalf("ok = %v, want true", out["ok"])
	}
	if _, hasMsg := out["message"]; !hasMsg {
		t.Fatalf("missing message: %v", out)
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "SUPERSECRETTOKEN") || strings.Contains(string(raw), secretURL) {
		t.Fatalf("test response leaks secret: %s", raw)
	}
	if strings.Contains(string(raw), "tok-XYZSECRET") {
		t.Fatalf("test response leaks token: %s", raw)
	}

	// Missing channel → 404.
	status, _ = call(t, env.handlers.TestAlertChannel, "POST", "/api/alert-channels/99/test", "",
		map[string]any{"id": "99", userKey: admin}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("test missing status = %d, want 404", status)
	}
}
