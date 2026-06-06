package handlers

import (
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/update"
	"github.com/valyala/fasthttp"
)

type fakeUpdateChecker struct {
	result *update.CheckResult
	err    error
}

func (f *fakeUpdateChecker) Check(current string) (*update.CheckResult, error) {
	return f.result, f.err
}

type fakeUpdater struct {
	err error
}

func (f *fakeUpdater) Apply(current, dataDir string) error {
	return f.err
}

func TestVersion(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Version(ctx, "1.0.0", "2024-01-01")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data struct {
			Version   string `json:"version"`
			GoVersion string `json:"go_version"`
			BuildDate string `json:"build_date"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if decoded.Data.Version != "1.0.0" {
		t.Fatalf("version = %q, want 1.0.0", decoded.Data.Version)
	}
	if decoded.Data.GoVersion == "" {
		t.Fatal("go_version should not be empty")
	}
	if decoded.Data.BuildDate != "2024-01-01" {
		t.Fatalf("build_date = %q, want 2024-01-01", decoded.Data.BuildDate)
	}
}

func TestUpdateCheckSuccess(t *testing.T) {
	checker := &fakeUpdateChecker{
		result: &update.CheckResult{
			Current:         "1.0.0",
			Latest:          "1.1.0",
			UpdateAvailable: true,
			ChangelogURL:    "https://github.com/bloodf/g0router/releases/tag/v1.1.0",
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		UpdateCheck(ctx, "1.0.0", checker)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data UpdateCheckResult `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if decoded.Data.Current != "1.0.0" {
		t.Fatalf("current = %q, want 1.0.0", decoded.Data.Current)
	}
	if decoded.Data.Latest != "1.1.0" {
		t.Fatalf("latest = %q, want 1.1.0", decoded.Data.Latest)
	}
	if !decoded.Data.UpdateAvailable {
		t.Fatal("expected update_available true")
	}
}

func TestUpdateCheckError(t *testing.T) {
	checker := &fakeUpdateChecker{err: errors.New("network error")}

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		UpdateCheck(ctx, "1.0.0", checker)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "check failed") {
		t.Fatalf("body = %s, want check failed error", body)
	}
}

func TestUpdateApplySuccess(t *testing.T) {
	s := newHandlerStore(t)
	updater := &fakeUpdater{}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, ``, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		UpdateApply(ctx, "1.0.0", updater, "/tmp/g0router", s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data struct {
			Staged bool `json:"staged"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if !decoded.Data.Staged {
		t.Fatal("expected staged true")
	}

	entry := lastAuditEntry(t, s, "update.apply")
	if entry == nil {
		t.Fatal("expected audit entry for update apply")
	}
}

func TestUpdateApplyNonAdmin(t *testing.T) {
	s := newHandlerStore(t)
	updater := &fakeUpdater{}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, ``, "1", "user", func(ctx *fasthttp.RequestCtx) {
		UpdateApply(ctx, "1.0.0", updater, "/tmp/g0router", s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestUpdateApplyError(t *testing.T) {
	s := newHandlerStore(t)
	updater := &fakeUpdater{err: errors.New("checksum mismatch")}

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, ``, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		UpdateApply(ctx, "1.0.0", updater, "/tmp/g0router", s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "checksum mismatch") {
		t.Fatalf("body = %s, want checksum mismatch error", body)
	}
}

func TestUpdateApplyStoreNil(t *testing.T) {
	updater := &fakeUpdater{}

	ctx, _ := runHandlerWithSession(t, fasthttp.MethodPost, ``, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		UpdateApply(ctx, "1.0.0", updater, "/tmp/g0router", nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestUpdateCheckerNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		UpdateCheck(ctx, "1.0.0", nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestUpdateUpdaterNil(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandlerWithSession(t, fasthttp.MethodPost, ``, "1", "admin", func(ctx *fasthttp.RequestCtx) {
		UpdateApply(ctx, "1.0.0", nil, "/tmp/g0router", s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}
