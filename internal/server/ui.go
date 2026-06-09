package server

import (
	"fmt"
	"io/fs"
	"mime"
	"path"
	"strings"

	"github.com/valyala/fasthttp"
)

const indexFile = "index.html"

func healthHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetContentTypeBytes([]byte("application/json"))
		ctx.SetBodyString(`{"status":"ok"}`)
	}
}

func uiHandler(uiFS fs.FS) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		reqPath := string(ctx.Path())
		relPath := strings.TrimPrefix(reqPath, "/")
		if relPath == "" {
			relPath = indexFile
		}

		if relPath != indexFile {
			if serveFile(ctx, uiFS, relPath) {
				return
			}
		}

		serveIndex(ctx, uiFS)
	}
}

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

func serveIndex(ctx *fasthttp.RequestCtx, uiFS fs.FS) {
	body, err := fs.ReadFile(uiFS, indexFile)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetContentTypeBytes([]byte("application/json"))
		ctx.SetBodyString(fmt.Sprintf(`{"error":"ui not built","hint":"run make ui-build"}`))
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("text/html; charset=utf-8"))
	ctx.SetBody(body)
}
