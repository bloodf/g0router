package admin

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestGetVersionReturnsInjectedInfo(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetVersionInfo("1.2.3", "2026-06-14")

	status, envl := call(t, env.handlers.GetVersion, "GET", "/api/version", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	if data["version"] != "1.2.3" {
		t.Fatalf("version = %v, want 1.2.3", data["version"])
	}
	if data["build_date"] != "2026-06-14" {
		t.Fatalf("build_date = %v, want 2026-06-14", data["build_date"])
	}
	// Default is deterministic: no live network in tests, so update_available is false.
	if data["update_available"] != false {
		t.Fatalf("update_available = %v, want false", data["update_available"])
	}
	if data["latest_version"] != "" {
		t.Fatalf("latest_version = %v, want empty", data["latest_version"])
	}
}

func TestGetVersionDefaultsWhenUnset(t *testing.T) {
	env := newTestEnv(t)
	// No SetVersionInfo call: fields default to "".
	status, envl := call(t, env.handlers.GetVersion, "GET", "/api/version", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	if data["version"] != "" {
		t.Fatalf("version = %v, want empty", data["version"])
	}
	if data["update_available"] != false {
		t.Fatalf("update_available = %v, want false", data["update_available"])
	}
}

func TestShutdownInvokesHookOnceWithoutExiting(t *testing.T) {
	env := newTestEnv(t)
	var called atomic.Int32
	env.handlers.SetShutdownFunc(func() { called.Add(1) })

	status, envl := call(t, env.handlers.Shutdown, "POST", "/api/version/shutdown", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d err = %q", status, errMessage(t, envl))
	}
	// Response-first: {ok:true} is returned before the hook fires.
	data := dataField[map[string]any](t, envl)
	if data["ok"] != true {
		t.Fatalf("ok = %v, want true", data["ok"])
	}

	// The hook fires asynchronously; wait (bounded) for exactly one invocation.
	deadline := time.Now().Add(2 * time.Second)
	for called.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := called.Load(); got != 1 {
		t.Fatalf("hook invoked %d times, want exactly 1", got)
	}

	// Give any erroneous extra invocation a chance to surface; must stay at 1.
	time.Sleep(50 * time.Millisecond)
	if got := called.Load(); got != 1 {
		t.Fatalf("hook invoked %d times after settle, want exactly 1", got)
	}
	// Reaching here proves the test process survived (no os.Exit).
}

func TestShutdownWithoutHookReturns501AndDoesNotExit(t *testing.T) {
	env := newTestEnv(t)
	// No SetShutdownFunc: the hook is nil; must respond 501 and not exit.
	status, envl := call(t, env.handlers.Shutdown, "POST", "/api/version/shutdown", "", nil, nil)
	if status != fasthttp.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", status)
	}
	data := dataField[map[string]any](t, envl)
	if data["ok"] != false {
		t.Fatalf("ok = %v, want false", data["ok"])
	}
	// Reaching here proves no exit happened on the nil-hook path.
}
