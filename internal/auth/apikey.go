package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"
)

const (
	defaultAPIKeySecret = "endpoint-proxy-api-key-secret"
	defaultMachineIDSalt = "endpoint-proxy-salt"
	cliAuthSalt          = "9r-cli-auth"
)

// apiKeySecret returns the HMAC secret used for API key CRCs.
func apiKeySecret() string {
	if s := os.Getenv("API_KEY_SECRET"); s != "" {
		return s
	}
	return defaultAPIKeySecret
}

// ParsedKey is the result of parsing an API key.
type ParsedKey struct {
	MachineID   string
	KeyID       string
	IsNewFormat bool
}

// GenerateAPIKey creates a new API key in the format sk-{machineId}-{keyId}-{crc8}
// and returns the full key together with its keyId.
func GenerateAPIKey(machineID string) (key, keyID string, err error) {
	keyID, err = generateKeyID()
	if err != nil {
		return "", "", err
	}
	crc := computeCRC(machineID, keyID, apiKeySecret())
	key = fmt.Sprintf("sk-%s-%s-%s", machineID, keyID, crc)
	return key, keyID, nil
}

// generateKeyID returns a 6-character random identifier using lowercase letters
// and digits.
func generateKeyID() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("generate key id: %w", err)
	}
	var sb strings.Builder
	for _, x := range b {
		// Rejection sampling to avoid modulo bias.
		if int(x) >= 252 {
			continue
		}
		sb.WriteByte(chars[int(x)%len(chars)])
		if sb.Len() == 6 {
			break
		}
	}
	if sb.Len() < 6 {
		// Should be vanishingly rare; fall back to a second pass.
		return generateKeyID()
	}
	return sb.String(), nil
}

// computeCRC returns the first 8 hex characters of an HMAC-SHA256 over
// machineID+keyID using the supplied secret.
func computeCRC(machineID, keyID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(machineID + keyID))
	return hex.EncodeToString(mac.Sum(nil))[:8]
}

// ParseAPIKey parses an API key. It supports the new format
// sk-{machineId}-{keyId}-{crc8} and the legacy format sk-{random8}.
// It returns nil when the key is malformed or the CRC does not match.
func ParseAPIKey(apiKey string) (*ParsedKey, error) {
	if !strings.HasPrefix(apiKey, "sk-") {
		return nil, nil
	}
	parts := strings.Split(apiKey, "-")

	// New format: sk-{machineId}-{keyId}-{crc8} = 4 parts.
	if len(parts) == 4 {
		machineID := parts[1]
		keyID := parts[2]
		crc := parts[3]
		expected := computeCRC(machineID, keyID, apiKeySecret())
		if crc != expected {
			return nil, nil
		}
		return &ParsedKey{MachineID: machineID, KeyID: keyID, IsNewFormat: true}, nil
	}

	// Legacy format: sk-{random8} = 2 parts.
	if len(parts) == 2 && parts[1] != "" {
		return &ParsedKey{KeyID: parts[1], IsNewFormat: false}, nil
	}

	return nil, nil
}

// MachineID returns a stable 16-character hex identifier derived from the
// persisted raw machine ID and the supplied salt. When salt is empty the
// default salt is used. When salt matches the CLI auth salt, the persisted
// CLI secret is mixed into the derivation.
func MachineID(dataDir, salt string) (string, error) {
	if salt == "" {
		salt = defaultMachineIDSalt
	}
	raw, err := loadRawMachineID(dataDir)
	if err != nil {
		return "", fmt.Errorf("load raw machine id: %w", err)
	}
	extra := ""
	if salt == cliAuthSalt {
		secret, err := loadCliSecret(dataDir)
		if err != nil {
			return "", fmt.Errorf("load cli secret: %w", err)
		}
		extra = secret
	}
	h := sha256.New()
	h.Write([]byte(raw + salt + extra))
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// loadRawMachineID returns the raw machine identifier for this host. It reads
// a persisted file in dataDir first, then falls back to the OS machine id, and
// finally to a random UUID. The resolved value is persisted to dataDir with
// mode 0600 so that all entrypoints see the same value.
func loadRawMachineID(dataDir string) (string, error) {
	path := filepath.Join(dataDir, "machine-id")
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return string(b), nil
	}

	raw, err := osMachineID()
	if err != nil {
		raw = randomUUID()
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return raw, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		return raw, fmt.Errorf("persist machine id: %w", err)
	}
	return raw, nil
}

// osMachineID reads the OS machine identifier. On Linux this is
// /etc/machine-id.
func osMachineID() (string, error) {
	b, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return "", fmt.Errorf("read os machine id: %w", err)
	}
	raw := strings.TrimSpace(string(b))
	if raw == "" {
		return "", fmt.Errorf("os machine id is empty")
	}
	return raw, nil
}

// randomUUID returns a random UUID string.
func randomUUID() string {
	b := make([]byte, 16)
	if _, err := randRead(b); err != nil {
		// crypto/rand failing is unrecoverable; this path should not be reached
		// in normal operation. Return a deterministic placeholder to avoid panic.
		return "0000000000000000"
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// APIKeyLookup resolves a raw API key to its machine identifier and active
// flag. It is supplied by the store layer so that this package stays free of
// store imports.
type APIKeyLookup func(key string) (machineID string, isActive bool, err error)

// NewAPIKeyValidator returns a guard validator that extracts an API key from
// the Authorization: Bearer or x-api-key header, parses/verifies it, and looks
// it up via lookup.
func NewAPIKeyValidator(lookup APIKeyLookup) func(*fasthttp.RequestCtx) bool {
	return func(ctx *fasthttp.RequestCtx) bool {
		key := extractAPIKey(ctx)
		if key == "" {
			return false
		}
		parsed, err := ParseAPIKey(key)
		if err != nil || parsed == nil {
			return false
		}
		_, isActive, err := lookup(key)
		if err != nil {
			return false
		}
		return isActive
	}
}

func extractAPIKey(ctx *fasthttp.RequestCtx) string {
	header := string(ctx.Request.Header.Peek("Authorization"))
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return after
	}
	return string(ctx.Request.Header.Peek("x-api-key"))
}

// NewCLITokenValidator returns a guard validator that accepts the
// x-9r-cli-token header when its value equals MachineID(dataDir, "9r-cli-auth").
func NewCLITokenValidator(dataDir string) func(*fasthttp.RequestCtx) bool {
	return func(ctx *fasthttp.RequestCtx) bool {
		token := string(ctx.Request.Header.Peek("x-9r-cli-token"))
		if token == "" {
			return false
		}
		expected, err := MachineID(dataDir, cliAuthSalt)
		if err != nil {
			return false
		}
		return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
	}
}

// loadCliSecret returns the persisted CLI auth secret, generating and
// persisting a random 32-byte hex secret on first use.
func loadCliSecret(dataDir string) (string, error) {
	authDir := filepath.Join(dataDir, "auth")
	path := filepath.Join(authDir, "cli-secret")
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return string(b), nil
	}

	b := make([]byte, 32)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("generate cli secret: %w", err)
	}
	secret := hex.EncodeToString(b)

	if err := os.MkdirAll(authDir, 0o700); err != nil {
		return "", fmt.Errorf("create auth dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return "", fmt.Errorf("persist cli secret: %w", err)
	}
	return secret, nil
}
