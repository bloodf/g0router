package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := Open(filepath.Join(dir, "g0router.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestLoadOrCreateSecret(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if len(first) != 32 {
		t.Fatalf("secret length = %d, want 32", len(first))
	}

	second, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("secret not stable across loads")
	}

	info, err := os.Stat(filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatalf("stat secret.key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("secret.key perm = %o, want 600", perm)
	}
}

func TestOpenEnablesWALAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	path := filepath.Join(dir, "g0router.db")

	st, err := Open(path, secret)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	var mode string
	if err := st.DB().QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Re-opening runs migrations again; they must be additive-only no-ops.
	st2, err := Open(path, secret)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer st2.Close()
}

func TestEnsureColumnIsAdditive(t *testing.T) {
	st := newTestStore(t)

	if err := ensureColumn(st.DB(), "settings", "extra_test_col", "TEXT NOT NULL DEFAULT ''"); err != nil {
		t.Fatalf("ensureColumn new: %v", err)
	}
	// Second call must be a no-op, not an error.
	if err := ensureColumn(st.DB(), "settings", "extra_test_col", "TEXT NOT NULL DEFAULT ''"); err != nil {
		t.Fatalf("ensureColumn existing: %v", err)
	}
}

func TestCipherRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	ct, err := c.Encrypt("sk-super-secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ct == "sk-super-secret" {
		t.Fatal("ciphertext equals plaintext")
	}

	pt, err := c.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if pt != "sk-super-secret" {
		t.Fatalf("round trip = %q", pt)
	}

	// Empty string round trips to empty without error.
	ect, err := c.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	ept, err := c.Decrypt(ect)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if ept != "" {
		t.Fatalf("empty round trip = %q", ept)
	}

	// Wrong key must fail to decrypt.
	otherKey := make([]byte, 32)
	other, err := NewCipher(otherKey)
	if err != nil {
		t.Fatalf("NewCipher other: %v", err)
	}
	if _, err := other.Decrypt(ct); err == nil {
		t.Fatal("decrypt with wrong key succeeded")
	}
}

func TestNewCipherRejectsBadKeyLength(t *testing.T) {
	if _, err := NewCipher([]byte("short")); err == nil {
		t.Fatal("NewCipher accepted short key")
	}
}

func TestUserCRUD(t *testing.T) {
	st := newTestStore(t)

	n, err := st.CountUsers()
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if n != 0 {
		t.Fatalf("CountUsers = %d, want 0", n)
	}

	u, err := st.CreateUser("admin", "hash123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == "" {
		t.Fatal("user ID empty")
	}

	got, err := st.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.PasswordHash != "hash123" {
		t.Fatalf("PasswordHash = %q", got.PasswordHash)
	}

	byID, err := st.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID.Username != "admin" {
		t.Fatalf("Username = %q", byID.Username)
	}

	if err := st.UpdateUserPassword(u.ID, "newhash"); err != nil {
		t.Fatalf("UpdateUserPassword: %v", err)
	}
	got, err = st.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("GetUserByUsername after update: %v", err)
	}
	if got.PasswordHash != "newhash" {
		t.Fatalf("PasswordHash after update = %q", got.PasswordHash)
	}

	if _, err := st.GetUserByUsername("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}

	// Duplicate usernames rejected.
	if _, err := st.CreateUser("admin", "x"); err == nil {
		t.Fatal("duplicate username accepted")
	}
}

func TestSessionLifecycle(t *testing.T) {
	st := newTestStore(t)
	u, err := st.CreateUser("admin", "h")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	exp := time.Now().Add(time.Hour).Unix()
	if err := st.CreateSession("tok1", u.ID, exp); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sess, err := st.GetSession("tok1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.UserID != u.ID || sess.ExpiresAt != exp {
		t.Fatalf("session = %+v", sess)
	}

	// Expired sessions are not returned.
	if err := st.CreateSession("tok2", u.ID, time.Now().Add(-time.Hour).Unix()); err != nil {
		t.Fatalf("CreateSession expired: %v", err)
	}
	if _, err := st.GetSession("tok2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session err = %v, want ErrNotFound", err)
	}

	if err := st.DeleteSession("tok1"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := st.GetSession("tok1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted session err = %v, want ErrNotFound", err)
	}
}

func TestSettingsGetPut(t *testing.T) {
	st := newTestStore(t)

	all, err := st.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("initial settings = %v, want empty", all)
	}

	if err := st.SetSettings(map[string]string{"theme": "dark", "log_level": "info"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	// Upsert overwrites.
	if err := st.SetSettings(map[string]string{"theme": "light"}); err != nil {
		t.Fatalf("SetSettings upsert: %v", err)
	}

	all, err = st.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if all["theme"] != "light" || all["log_level"] != "info" {
		t.Fatalf("settings = %v", all)
	}
}

func TestProviderCRUD(t *testing.T) {
	st := newTestStore(t)

	p := &ProviderRecord{Name: "OpenAI Main", Type: "openai", BaseURL: "https://api.openai.com/v1", Enabled: true}
	if err := st.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	if p.ID == "" {
		t.Fatal("provider ID empty")
	}

	list, err := st.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(list) != 1 || list[0].Name != "OpenAI Main" {
		t.Fatalf("list = %+v", list)
	}

	p.Name = "OpenAI Renamed"
	p.Enabled = false
	if err := st.UpdateProvider(p); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	got, err := st.GetProvider(p.ID)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if got.Name != "OpenAI Renamed" || got.Enabled {
		t.Fatalf("got = %+v", got)
	}

	if err := st.DeleteProvider(p.ID); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	if _, err := st.GetProvider(p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted provider err = %v, want ErrNotFound", err)
	}
	if err := st.UpdateProvider(p); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing provider err = %v, want ErrNotFound", err)
	}
	if err := st.DeleteProvider(p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing provider err = %v, want ErrNotFound", err)
	}
}

func TestConnectionCRUDEncryptsSecrets(t *testing.T) {
	st := newTestStore(t)

	p := &ProviderRecord{Name: "Anthropic", Type: "anthropic", Enabled: true}
	if err := st.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	c := &Connection{
		ProviderID:   p.ID,
		Name:         "main key",
		Kind:         "api_key",
		Secret:       "sk-ant-plaintext",
		AccessToken:  "at-plaintext",
		RefreshToken: "rt-plaintext",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		Metadata:     `{"scopes":["inference"]}`,
	}
	if err := st.CreateConnection(c); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if c.ID == "" {
		t.Fatal("connection ID empty")
	}

	// Raw *_enc columns must not contain plaintext.
	var secretEnc, accessEnc, refreshEnc string
	row := st.DB().QueryRow("SELECT secret_enc, access_token_enc, refresh_token_enc FROM connections WHERE id = ?", c.ID)
	if err := row.Scan(&secretEnc, &accessEnc, &refreshEnc); err != nil {
		t.Fatalf("scan raw columns: %v", err)
	}
	for name, raw := range map[string]string{"secret_enc": secretEnc, "access_token_enc": accessEnc, "refresh_token_enc": refreshEnc} {
		if raw == "" {
			t.Fatalf("%s is empty", name)
		}
		if strings.Contains(raw, "plaintext") {
			t.Fatalf("%s contains plaintext: %q", name, raw)
		}
	}

	got, err := st.GetConnection(c.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.Secret != "sk-ant-plaintext" || got.AccessToken != "at-plaintext" || got.RefreshToken != "rt-plaintext" {
		t.Fatalf("decrypted = %+v", got)
	}

	list, err := st.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(list) != 1 || list[0].Secret != "sk-ant-plaintext" {
		t.Fatalf("list = %+v", list)
	}

	got.Secret = "sk-ant-rotated"
	got.Name = "rotated"
	if err := st.UpdateConnection(got); err != nil {
		t.Fatalf("UpdateConnection: %v", err)
	}
	got2, err := st.GetConnection(c.ID)
	if err != nil {
		t.Fatalf("GetConnection after update: %v", err)
	}
	if got2.Secret != "sk-ant-rotated" || got2.Name != "rotated" {
		t.Fatalf("after update = %+v", got2)
	}

	if err := st.DeleteConnection(c.ID); err != nil {
		t.Fatalf("DeleteConnection: %v", err)
	}
	if _, err := st.GetConnection(c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted connection err = %v, want ErrNotFound", err)
	}
}

func TestOAuthSessionConsume(t *testing.T) {
	st := newTestStore(t)

	o := &OAuthSession{
		State:     "state-abc",
		Provider:  "anthropic",
		Verifier:  "verifier-plaintext",
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}
	if err := st.CreateOAuthSession(o); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}

	// Verifier stored encrypted.
	var verifierEnc string
	if err := st.DB().QueryRow("SELECT verifier_enc FROM oauth_sessions WHERE state = ?", o.State).Scan(&verifierEnc); err != nil {
		t.Fatalf("scan verifier_enc: %v", err)
	}
	if strings.Contains(verifierEnc, "plaintext") {
		t.Fatalf("verifier_enc contains plaintext: %q", verifierEnc)
	}

	got, err := st.ConsumeOAuthSession("state-abc")
	if err != nil {
		t.Fatalf("ConsumeOAuthSession: %v", err)
	}
	if got.Verifier != "verifier-plaintext" || got.Provider != "anthropic" {
		t.Fatalf("consumed = %+v", got)
	}

	// One-shot: second consume fails.
	if _, err := st.ConsumeOAuthSession("state-abc"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second consume err = %v, want ErrNotFound", err)
	}

	// Expired sessions cannot be consumed.
	expired := &OAuthSession{State: "state-old", Provider: "anthropic", Verifier: "v", ExpiresAt: time.Now().Add(-time.Minute).Unix()}
	if err := st.CreateOAuthSession(expired); err != nil {
		t.Fatalf("CreateOAuthSession expired: %v", err)
	}
	if _, err := st.ConsumeOAuthSession("state-old"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired consume err = %v, want ErrNotFound", err)
	}
}
