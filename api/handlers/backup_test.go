package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeBackupStore struct {
	exportData []byte
	exportErr  error
	importErr  error
}

func (f *fakeBackupStore) ExportBackup() ([]byte, error) {
	if f.exportErr != nil {
		return nil, f.exportErr
	}
	return f.exportData, nil
}

func (f *fakeBackupStore) ImportBackup(data []byte) error {
	return f.importErr
}

func TestBackupHandler(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-backup-key")

	// Create some data
	conn := &store.Connection{Provider: "openai", AuthType: "oauth", AccessToken: stringPtr("secret-tok")}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		Backup(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var export map[string]any
	if err := json.Unmarshal(body, &export); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, body)
	}
	if export["schema_version"] != "1" {
		t.Fatalf("schema_version = %v", export["schema_version"])
	}

	// Ensure no secret values
	bodyStr := string(body)
	if strings.Contains(bodyStr, "secret-tok") {
		t.Fatal("backup contains secret token")
	}
}

func TestBackupHandlerStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		Backup(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestBackupHandlerExportError(t *testing.T) {
	fs := &fakeBackupStore{exportErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		Backup(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestRestoreHandler(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-backup-key")

	// Create data, export, modify, restore
	conn := &store.Connection{Provider: "openai", AuthType: "oauth", AccessToken: stringPtr("preserve-me")}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	backupData, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	var backup map[string]any
	if err := json.Unmarshal(backupData, &backup); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	connections := backup["connections"].([]any)
	c := connections[0].(map[string]any)
	c["name"] = "restored-name"

	modified, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, string(modified), func(ctx *fasthttp.RequestCtx) {
		Restore(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	// Verify restored
	conns, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Name != "restored-name" {
		t.Fatalf("name = %q, want restored-name", conns[0].Name)
	}
	if conns[0].AccessToken == nil || *conns[0].AccessToken != "preserve-me" {
		t.Fatalf("secret not preserved")
	}
}

func TestRestoreHandlerStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"schema_version":"1"}`, func(ctx *fasthttp.RequestCtx) {
		Restore(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestRestoreHandlerInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"schema_version":`, func(ctx *fasthttp.RequestCtx) {
		Restore(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRestoreHandlerBadSchema(t *testing.T) {
	fs := &fakeBackupStore{importErr: errors.New("unsupported schema version \"99\"")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"schema_version":"99"}`, func(ctx *fasthttp.RequestCtx) {
		Restore(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRestoreHandlerImportError(t *testing.T) {
	fs := &fakeBackupStore{importErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"schema_version":"1"}`, func(ctx *fasthttp.RequestCtx) {
		Restore(ctx, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestBackupHandlerMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Backup(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestRestoreHandlerMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Restore(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}
