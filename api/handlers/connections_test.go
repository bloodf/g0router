package handlers

import (
	"bytes"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestConnectionsCreateListUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	createBody := `{"provider":"openai","name":"primary","auth_type":"api_key","api_key":"sk-test","is_active":true,"provider_specific_data":{"region":"us"},"model_locks":{"gpt-4o":123}}`
	ctx, body := runHandler(t, fasthttp.MethodPost, createBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)

	var created store.Connection
	decodeJSON(t, body, &created)
	if created.ID == "" || created.Name != "primary" || created.APIKey != nil {
		t.Fatalf("created connection = %+v", created)
	}
	stored, err := s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.APIKey == nil || *stored.APIKey != "sk-test" {
		t.Fatalf("stored API key = %v, want sk-test", stored.APIKey)
	}

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	var listed struct {
		Data []store.Connection `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].ID != created.ID {
		t.Fatalf("listed = %+v, want created connection", listed.Data)
	}

	updateBody := `{"provider":"openai","name":"renamed","auth_type":"api_key","api_key":"sk-test-2","is_active":false}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	var updated store.Connection
	decodeJSON(t, body, &updated)
	if updated.Name != "renamed" || updated.IsActive || updated.APIKey != nil {
		t.Fatalf("updated = %+v", updated)
	}
	stored, err = s.GetConnection(created.ID)
	if err != nil {
		t.Fatalf("GetConnection after update: %v", err)
	}
	if stored.APIKey == nil || *stored.APIKey != "sk-test-2" {
		t.Fatalf("stored updated API key = %v, want sk-test-2", stored.APIKey)
	}

	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsResponsesRedactCredentialsWithoutMutatingStore(t *testing.T) {
	s := newHandlerStore(t)

	createBody := `{"provider":"openai","name":"primary","auth_type":"oauth","access_token":"access-secret","refresh_token":"refresh-secret","api_key":"api-secret","is_active":true}`
	ctx, body := runHandler(t, fasthttp.MethodPost, createBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)

	var created store.Connection
	decodeJSON(t, body, &created)
	assertStoredCredentials(t, s, created.ID, "access-secret", "refresh-secret", "api-secret")

	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	assertStoredCredentials(t, s, created.ID, "access-secret", "refresh-secret", "api-secret")

	updateBody := `{"provider":"openai","name":"renamed","auth_type":"oauth","access_token":"access-new","refresh_token":"refresh-new","api_key":"api-new","is_active":false}`
	ctx, body = runHandler(t, fasthttp.MethodPut, updateBody, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoCredentialFields(t, body)
	assertStoredCredentials(t, s, created.ID, "access-new", "refresh-new", "api-new")
}

func TestConnectionsMissingReturnsNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestConnectionsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":`, func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func assertNoCredentialFields(t *testing.T, body []byte) {
	t.Helper()

	for _, field := range [][]byte{
		[]byte(`"AccessToken":`),
		[]byte(`"RefreshToken":`),
		[]byte(`"APIKey":`),
		[]byte(`"access_token":`),
		[]byte(`"refresh_token":`),
		[]byte(`"api_key":`),
	} {
		if bytes.Contains(body, field) {
			t.Fatalf("response serialized credential field %s: %s", field, body)
		}
	}
}

func assertStoredCredentials(t *testing.T, s *store.Store, id, accessToken, refreshToken, apiKey string) {
	t.Helper()

	conn, err := s.GetConnection(id)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if conn.AccessToken == nil || *conn.AccessToken != accessToken {
		t.Fatalf("stored access token = %v, want %s", conn.AccessToken, accessToken)
	}
	if conn.RefreshToken == nil || *conn.RefreshToken != refreshToken {
		t.Fatalf("stored refresh token = %v, want %s", conn.RefreshToken, refreshToken)
	}
	if conn.APIKey == nil || *conn.APIKey != apiKey {
		t.Fatalf("stored API key = %v, want %s", conn.APIKey, apiKey)
	}
}
