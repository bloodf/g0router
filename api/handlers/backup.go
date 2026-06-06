package handlers

import (
	"encoding/json"
	"log"

	"github.com/valyala/fasthttp"
)

type backupStore interface {
	ExportBackup() ([]byte, error)
	ImportBackup(data []byte) error
}

func Backup(ctx *fasthttp.RequestCtx, s backupStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	data, err := s.ExportBackup()
	if err != nil {
		log.Printf("export backup: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to export backup")
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(data)
}

func Restore(ctx *fasthttp.RequestCtx, s backupStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	if err := s.ImportBackup(ctx.PostBody()); err != nil {
		status := fasthttp.StatusInternalServerError
		if req.SchemaVersion != "" && req.SchemaVersion != "1" {
			status = fasthttp.StatusBadRequest
		}
		if status == fasthttp.StatusBadRequest {
			writeError(ctx, status, err.Error())
			return
		}
		log.Printf("import backup: %v", err)
		writeError(ctx, status, "failed to import backup")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"success": true})
}
