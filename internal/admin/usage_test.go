package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func wireUsageServices(t *testing.T, env *testEnv) {
	t.Helper()
	stats, resolver := BuildUsageServices(env.store, UsageDeps{})
	env.handlers.SetUsageServices(stats, resolver)
}

func TestUsageStatsRoute(t *testing.T) {
	env := newTestEnv(t)
	wireUsageServices(t, env)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.RequireSession(env.handlers.GetUsageStats), "GET", "/api/usage/stats?period=7d", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	if _, ok := data["total_requests"]; !ok {
		t.Errorf("response missing total_requests: %v", data)
	}

	status, envl = call(t, env.handlers.RequireSession(env.handlers.GetUsageStats), "GET", "/api/usage/stats?period=invalid", "", nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("invalid period status = %d, want 400", status)
	}
}

func TestRequestDetailsRouteValidation(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.RequireSession(env.handlers.GetRequestDetails), "GET", "/api/usage/request-details?page=0", "", nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("page=0 status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); msg == "" {
		t.Fatal("expected error message for page=0")
	}

	status, envl = call(t, env.handlers.RequireSession(env.handlers.GetRequestDetails), "GET", "/api/usage/request-details?pageSize=101", "", nil, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("pageSize=101 status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); msg == "" {
		t.Fatal("expected error message for pageSize=101")
	}

	status, _ = call(t, env.handlers.RequireSession(env.handlers.GetRequestDetails), "GET", "/api/usage/request-details?page=1&pageSize=50", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("valid params status = %d, want 200", status)
	}
}

func TestUsageChartAndLogsRoutes(t *testing.T) {
	env := newTestEnv(t)
	wireUsageServices(t, env)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, _ := call(t, env.handlers.RequireSession(env.handlers.GetUsageChart), "GET", "/api/usage/chart?period=7d", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("chart status = %d", status)
	}

	status, _ = call(t, env.handlers.RequireSession(env.handlers.GetUsageRequestLogs), "GET", "/api/usage/request-logs", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("request-logs status = %d", status)
	}
}

func TestBuildUsageServices(t *testing.T) {
	env := newTestEnv(t)
	stats, resolver := BuildUsageServices(env.store, UsageDeps{})
	if stats == nil {
		t.Fatal("BuildUsageServices returned nil StatsService")
	}
	if resolver == nil {
		t.Fatal("BuildUsageServices returned nil Resolver")
	}
}
