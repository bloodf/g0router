package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestPutSettingsStripsOIDCSecretFromPlaintext(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	// Configure OIDC via the flat settings endpoint, including the secret.
	body := `{"auth_mode":"oidc","oidc_issuer_url":"https://idp.example.com","oidc_client_id":"cid","oidc_client_secret":"top-secret"}`
	status, envl := call(t, env.handlers.RequireSession(env.handlers.PutSettings),
		"PUT", "/api/settings", body, nil,
		map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusOK {
		t.Fatalf("put settings status = %d, err = %q", status, errMessage(t, envl))
	}

	// The response (and GetSettings) must NOT echo the plaintext secret.
	returned := dataField[map[string]string](t, envl)
	if returned["oidc_client_secret"] != "" {
		t.Fatalf("PutSettings echoed plaintext secret: %q", returned["oidc_client_secret"])
	}

	settings, err := env.store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings["oidc_client_secret"] != "" {
		t.Fatalf("plaintext secret persisted in settings: %q", settings["oidc_client_secret"])
	}

	// But the secret is stored encrypted and oidcConfigured sees it.
	got, err := env.store.GetOIDCSecret()
	if err != nil {
		t.Fatalf("GetOIDCSecret: %v", err)
	}
	if got != "top-secret" {
		t.Fatalf("stored secret = %q, want %q", got, "top-secret")
	}
	if !env.handlers.oidcConfigured(settings) {
		t.Fatalf("oidcConfigured = false, want true after secret set")
	}
}
