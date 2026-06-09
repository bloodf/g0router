package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"os"
	"path"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/bloodf/g0router"
)

// version is the build-time version string. It is overridden by the
// release pipeline; the default keeps `go test` and local development
// working without a linker flag.
var (
	version   = "0.2.0-dev"
	buildDate = ""
)

const (
	// defaultListen is the bind address used when G0ROUTER_LISTEN is
	// not set. The UI's vite dev server proxies to 127.0.0.1:20129,
	// so the backend defaults to 20128 to avoid a clash during
	// local development.
	defaultListen = ":20128"

	// healthPath is the public health endpoint that the dashboard
	// pings to verify the backend is up.
	healthPath = "/api/health"

	// indexFile is the SPA fallback served for any non-asset path.
	indexFile = "index.html"
)

// main is the entry point. It loads the (Phase 1 minimal) runtime
// configuration, opens the embedded UI filesystem, and starts a
// fasthttp server with two route classes:
//
//	GET  /api/health  — JSON status
//	all other paths   — embedded UI static files (with SPA fallback)
//
// Fatal startup failures (port already in use, etc.) are surfaced via
// log.Fatalf; runtime request errors are written back as HTTP
// responses and never panic.
func main() {
	listenAddr := os.Getenv("G0ROUTER_LISTEN")
	if listenAddr == "" {
		listenAddr = defaultListen
	}

	uiFS, err := g0router.UI()
	if err != nil {
		log.Fatalf("open embedded ui: %v", err)
	}

	server := &fasthttp.Server{
		Handler:            newHandler(uiFS),
		ReadTimeout:        0,
		WriteTimeout:       0,
		MaxRequestBodySize: 1 << 30, // 1 GiB; large enough for batch embeddings
	}

	versionLine := version
	if buildDate != "" {
		versionLine = fmt.Sprintf("%s (built %s)", version, buildDate)
	}
	log.Printf("g0router %s listening on %s", versionLine, listenAddr)

	if err := server.ListenAndServe(listenAddr); err != nil {
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
}

// newHandler builds the top-level fasthttp request handler that
// routes between the public health endpoint and the embedded UI.
func newHandler(uiFS fs.FS) fasthttp.RequestHandler {
	health := healthHandler()
	ui := uiHandler(uiFS)

	return func(ctx *fasthttp.RequestCtx) {
		// /api/health is the only API route in Phase 1. Everything
		// else is served from the embedded UI.
		if string(ctx.Path()) == healthPath && ctx.IsGet() {
			health(ctx)
			return
		}
		ui(ctx)
	}
}

// healthHandler returns a JSON 200 OK handler for /api/health.
func healthHandler() fasthttp.RequestHandler {
	body, err := json.Marshal(map[string]string{"status": "ok"})
	if err != nil {
		// json.Marshal on a fixed map[string]string cannot fail in
		// practice, but if it ever did we want a clear startup-time
		// error rather than a silent 500.
		log.Fatalf("marshal health response: %v", err)
	}

	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBody(body)
		ctx.SetContentTypeBytes([]byte("application/json"))
	}
}

// uiHandler serves static files from the embedded UI filesystem with
// SPA fallback. If a request maps to a file in the FS, that file is
// served with a Content-Type inferred from its extension. Otherwise,
// index.html is returned so client-side routing works.
func uiHandler(uiFS fs.FS) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		reqPath := string(ctx.Path())
		// Strip leading slash; fs.FS expects a clean relative
		// path.
		relPath := strings.TrimPrefix(reqPath, "/")
		if relPath == "" {
			relPath = indexFile
		}

		// If the request hits a static asset that exists, serve it.
		if relPath != indexFile {
			if serveFile(ctx, uiFS, relPath) {
				return
			}
		}

		// Otherwise (or for the index itself), serve the SPA
		// fallback. Phase 1's UI placeholder is just a stub, so
		// index.html is the only thing the catch-all has to return.
		serveIndex(ctx, uiFS)
	}
}

// serveFile attempts to serve a single file from the embedded UI
// filesystem. It returns true if the file was found and served,
// false if the caller should fall back to the SPA shell.
func serveFile(ctx *fasthttp.RequestCtx, uiFS fs.FS, relPath string) bool {
	body, err := fs.ReadFile(uiFS, relPath)
	if err != nil {
		return false
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	if ct := mime.TypeByExtension(path.Ext(relPath)); ct != "" {
		ctx.SetContentTypeBytes([]byte(ct))
	} else {
		ctx.SetContentTypeBytes([]byte("application/octet-stream"))
	}
	ctx.SetBody(body)
	return true
}

// serveIndex serves the SPA shell from the embedded UI filesystem.
func serveIndex(ctx *fasthttp.RequestCtx, uiFS fs.FS) {
	body, err := fs.ReadFile(uiFS, indexFile)
	if err != nil {
		// UI build hasn't shipped yet (parallel UI task is still
		// in flight) — return a clean 404 with a JSON body so the
		// dashboard can detect the missing UI explicitly.
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetContentTypeBytes([]byte("application/json"))
		ctx.SetBodyString(fmt.Sprintf(`{"error":"ui not built","hint":"run make ui-build"}`))
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("text/html; charset=utf-8"))
	ctx.SetBody(body)
}
