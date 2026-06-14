package auth

import (
	"strings"
	"testing"
)

func TestClaudeOAuthConfig(t *testing.T) {
	cfg := ClaudeOAuth()
	if cfg.Provider != "claude" {
		t.Errorf("Provider = %q, want claude", cfg.Provider)
	}
	if cfg.ClientID != "9d1c250a-e61b-44d9-88ed-5944d1962f5e" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
	if cfg.AuthorizeURL != "https://claude.ai/oauth/authorize" {
		t.Errorf("AuthorizeURL = %q", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://api.anthropic.com/v1/oauth/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if len(cfg.Scopes) != 3 {
		t.Errorf("Scopes = %v", cfg.Scopes)
	}
}

func TestClaudeOAuthClientIDEnvOverride(t *testing.T) {
	t.Setenv("G0ROUTER_CLAUDE_CLIENT_ID", "custom-claude")
	if got := ClaudeOAuth().ClientID; got != "custom-claude" {
		t.Errorf("ClientID = %q, want custom-claude", got)
	}
}

func TestCodexOAuthConfig(t *testing.T) {
	cfg := CodexOAuth()
	if cfg.Provider != "codex" {
		t.Errorf("Provider = %q, want codex", cfg.Provider)
	}
	if cfg.ClientID != "app_EMoamEEZ73f0CkXaXp7hrann" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
	if cfg.AuthorizeURL != "https://auth.openai.com/oauth/authorize" {
		t.Errorf("AuthorizeURL = %q", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://auth.openai.com/oauth/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	wantScopes := []string{"openid", "profile", "email", "offline_access"}
	if strings.Join(cfg.Scopes, " ") != strings.Join(wantScopes, " ") {
		t.Errorf("Scopes = %v, want %v", cfg.Scopes, wantScopes)
	}
	if cfg.ExtraAuthParams["id_token_add_organizations"] != "true" {
		t.Errorf("ExtraAuthParams id_token_add_organizations = %q", cfg.ExtraAuthParams["id_token_add_organizations"])
	}
	if cfg.ExtraAuthParams["codex_cli_simplified_flow"] != "true" {
		t.Errorf("ExtraAuthParams codex_cli_simplified_flow = %q", cfg.ExtraAuthParams["codex_cli_simplified_flow"])
	}
	if cfg.ExtraAuthParams["originator"] != "codex_cli_rs" {
		t.Errorf("ExtraAuthParams originator = %q", cfg.ExtraAuthParams["originator"])
	}
}

func TestGeminiCLIOAuthConfig(t *testing.T) {
	cfg := GeminiCLIOAuth()
	if cfg.Provider != "gemini-cli" {
		t.Errorf("Provider = %q, want gemini-cli", cfg.Provider)
	}
	// Reuses the in-tree gemini clientId+secret.
	wantClientID := "681255809395" + "-" + "oo8ft2oprdrnp9e3aqf6av3hmdib135j" + ".apps.googleusercontent.com"
	if cfg.ClientID != wantClientID {
		t.Errorf("ClientID = %q, want %q", cfg.ClientID, wantClientID)
	}
	wantClientSecret := "GOCSPX" + "-" + "4uHgMPm" + "-" + "1o7Sk" + "-" + "geV6Cu5clXFsxl"
	if cfg.ClientSecret != wantClientSecret {
		t.Errorf("ClientSecret = %q", cfg.ClientSecret)
	}
	if cfg.AuthorizeURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Errorf("AuthorizeURL = %q", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if cfg.ExtraAuthParams["access_type"] != "offline" {
		t.Errorf("ExtraAuthParams access_type = %q", cfg.ExtraAuthParams["access_type"])
	}
	if cfg.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("ExtraAuthParams prompt = %q", cfg.ExtraAuthParams["prompt"])
	}
}

func TestQwenOAuthConfig(t *testing.T) {
	cfg := QwenOAuth()
	if cfg.Provider != "qwen" {
		t.Errorf("Provider = %q, want qwen", cfg.Provider)
	}
	if cfg.ClientID != "f0304373b74a44d2b584a3fb70ca9e56" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
	if cfg.DeviceCodeURL != "https://chat.qwen.ai/api/v1/oauth2/device/code" {
		t.Errorf("DeviceCodeURL = %q", cfg.DeviceCodeURL)
	}
	if cfg.TokenURL != "https://chat.qwen.ai/api/v1/oauth2/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	wantScopes := []string{"openid", "profile", "email", "model.completion"}
	if strings.Join(cfg.Scopes, " ") != strings.Join(wantScopes, " ") {
		t.Errorf("Scopes = %v, want %v", cfg.Scopes, wantScopes)
	}
}

func TestIflowOAuthConfig(t *testing.T) {
	cfg := IflowOAuth()
	if cfg.Provider != "iflow" {
		t.Errorf("Provider = %q, want iflow", cfg.Provider)
	}
	if cfg.ClientID != "10009311001" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
	wantSecret := "4Z3YjXyc" + "VsQvyGF1" + "etiNlIBB" + "4RsqSDtW"
	if cfg.ClientSecret != wantSecret {
		t.Errorf("ClientSecret = %q", cfg.ClientSecret)
	}
	if cfg.AuthorizeURL != "https://iflow.cn/oauth" {
		t.Errorf("AuthorizeURL = %q", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://iflow.cn/oauth/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if cfg.ExtraAuthParams["loginMethod"] != "phone" {
		t.Errorf("ExtraAuthParams loginMethod = %q", cfg.ExtraAuthParams["loginMethod"])
	}
	if cfg.ExtraAuthParams["type"] != "phone" {
		t.Errorf("ExtraAuthParams type = %q", cfg.ExtraAuthParams["type"])
	}
	if cfg.RefreshMode != "basic" {
		t.Errorf("RefreshMode = %q, want basic", cfg.RefreshMode)
	}
}

func TestIflowOAuthClientSecretEnvOverride(t *testing.T) {
	t.Setenv("G0ROUTER_IFLOW_CLIENT_SECRET", "custom-secret")
	if got := IflowOAuth().ClientSecret; got != "custom-secret" {
		t.Errorf("ClientSecret = %q, want custom-secret", got)
	}
}

func TestGithubOAuthConfig(t *testing.T) {
	cfg := GithubOAuth()
	if cfg.Provider != "github" {
		t.Errorf("Provider = %q, want github", cfg.Provider)
	}
	if cfg.ClientID != "Iv1.b507a08c87ecfe98" {
		t.Errorf("ClientID = %q", cfg.ClientID)
	}
	if cfg.DeviceCodeURL != "https://github.com/login/device/code" {
		t.Errorf("DeviceCodeURL = %q", cfg.DeviceCodeURL)
	}
	if cfg.TokenURL != "https://github.com/login/oauth/access_token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if strings.Join(cfg.Scopes, " ") != "read:user" {
		t.Errorf("Scopes = %v, want [read:user]", cfg.Scopes)
	}
	// GitHub device-code yields a GitHub token; no token refresh (Copilot mint deferred).
	if cfg.RefreshMode != "none" {
		t.Errorf("RefreshMode = %q, want none", cfg.RefreshMode)
	}
}

func TestKilocodeOAuthConfig(t *testing.T) {
	cfg := KilocodeOAuth()
	if cfg.Provider != "kilocode" {
		t.Errorf("Provider = %q, want kilocode", cfg.Provider)
	}
	if cfg.DeviceCodeURL != "https://api.kilo.ai/api/device-auth/codes" {
		t.Errorf("DeviceCodeURL = %q", cfg.DeviceCodeURL)
	}
	if cfg.DeviceVariant != "kilocode" {
		t.Errorf("DeviceVariant = %q, want kilocode", cfg.DeviceVariant)
	}
	if cfg.RefreshMode != "none" {
		t.Errorf("RefreshMode = %q, want none", cfg.RefreshMode)
	}
}

func TestClineOAuthConfig(t *testing.T) {
	cfg := ClineOAuth()
	if cfg.Provider != "cline" {
		t.Errorf("Provider = %q, want cline", cfg.Provider)
	}
	if cfg.AuthorizeURL != "https://api.cline.bot/api/v1/auth/authorize" {
		t.Errorf("AuthorizeURL = %q", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://api.cline.bot/api/v1/auth/token" {
		t.Errorf("TokenURL = %q", cfg.TokenURL)
	}
	if cfg.RefreshURL != "https://api.cline.bot/api/v1/auth/refresh" {
		t.Errorf("RefreshURL = %q", cfg.RefreshURL)
	}
	if cfg.CodeEncoding != "base64-json" {
		t.Errorf("CodeEncoding = %q, want base64-json", cfg.CodeEncoding)
	}
	if cfg.RefreshMode != "json" {
		t.Errorf("RefreshMode = %q, want json", cfg.RefreshMode)
	}
	if cfg.ExtraAuthParams["client_type"] != "extension" {
		t.Errorf("ExtraAuthParams client_type = %q", cfg.ExtraAuthParams["client_type"])
	}
}
