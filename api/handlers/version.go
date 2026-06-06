package handlers

import (
	"log"
	"runtime"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/update"
	"github.com/valyala/fasthttp"
)

// UpdateCheckResult is the public shape of the update check response.
type UpdateCheckResult struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	ChangelogURL    string `json:"changelog_url"`
}

// UpdateChecker is the narrow interface for checking updates.
type UpdateChecker interface {
	Check(current string) (*update.CheckResult, error)
}

// Updater is the narrow interface for applying updates.
type Updater interface {
	Apply(current, dataDir string) error
}

// Version handles GET /api/version.
func Version(ctx *fasthttp.RequestCtx, version, buildDate string) {
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"data": map[string]any{
			"version":    version,
			"go_version": runtime.Version(),
			"build_date": buildDate,
		},
	})
}

// UpdateCheck handles POST /api/update/check.
func UpdateCheck(ctx *fasthttp.RequestCtx, current string, checker UpdateChecker) {
	if checker == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "update checker unavailable")
		return
	}
	result, err := checker.Check(current)
	if err != nil {
		log.Printf("update check: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "update check failed")
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"data": UpdateCheckResult{
			Current:         result.Current,
			Latest:          result.Latest,
			UpdateAvailable: result.UpdateAvailable,
			ChangelogURL:    result.ChangelogURL,
		},
	})
}

// UpdateApply handles POST /api/update/apply (admin only, audited).
func UpdateApply(ctx *fasthttp.RequestCtx, current string, updater Updater, dataDir string, settings settingsStore, audit auditWriter) {
	if isStoreNil(settings) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	role, _ := ctx.UserValue("g0router.session_role").(string)
	if role != "admin" {
		writeError(ctx, fasthttp.StatusForbidden, "admin access required")
		return
	}

	if updater == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "updater unavailable")
		return
	}

	if err := updater.Apply(current, dataDir); err != nil {
		log.Printf("update apply: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
		return
	}

	_ = audit.AppendAudit(store.AuditEntry{
		Action: "update.apply",
		Target: current,
	})

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": map[string]any{"staged": true}})
}
