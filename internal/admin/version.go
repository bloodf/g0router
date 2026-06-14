package admin

import (
	"github.com/valyala/fasthttp"
)

// versionDTO is the GET /api/version response payload. version/build_date come
// from the injected binary fields (SetVersionInfo). update_available/latest_version
// are best-effort: they default off so the handler never performs a live network
// call (deterministic in tests). A future serial follow-up may wire an injectable
// latest-version source.
type versionDTO struct {
	Version         string `json:"version"`
	BuildDate       string `json:"build_date"`
	UpdateAvailable bool   `json:"update_available"`
	LatestVersion   string `json:"latest_version"`
}

// GetVersion handles GET /api/version (PAR-UI-102). It reports the injected
// version/build date and a deterministic, network-free update status.
func (h *Handlers) GetVersion(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, versionDTO{
		Version:         h.version,
		BuildDate:       h.buildDate,
		UpdateAvailable: false,
		LatestVersion:   "",
	})
}

// Shutdown handles POST /api/version/shutdown (PAR-UI-103). It triggers the
// injected graceful-shutdown hook. The hook is invoked ASYNCHRONOUSLY, after the
// response is written, so the response is flushed before the process tears down
// and so the handler body itself never terminates the process or closes the
// server directly. When no hook is wired, it responds 501 and does nothing
// (nil-safe). The real teardown path lives only in the injected hook (main.go).
func (h *Handlers) Shutdown(ctx *fasthttp.RequestCtx) {
	fn := h.shutdownFunc
	if fn == nil {
		writeData(ctx, fasthttp.StatusNotImplemented, map[string]any{
			"ok":      false,
			"message": "shutdown not wired",
		})
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"ok": true})
	go fn()
}
