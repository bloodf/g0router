package auth

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateAPIKeyFormat(t *testing.T) {
	machineID := "deadbeefdeadbeef"
	key, keyID, err := GenerateAPIKey(machineID)
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	re := regexp.MustCompile(`^sk-[0-9a-f]{16}-[a-z0-9]{6}-[0-9a-f]{8}$`)
	if !re.MatchString(key) {
		t.Fatalf("key %q does not match expected format", key)
	}
	if len(keyID) != 6 {
		t.Fatalf("keyID length = %d, want 6", len(keyID))
	}
}

func TestCRCRecomputeMatches(t *testing.T) {
	machineID := "deadbeefdeadbeef"
	keyID := "abc123"
	secret := "endpoint-proxy-api-key-secret"

	crc1 := computeCRC(machineID, keyID, secret)
	crc2 := computeCRC(machineID, keyID, secret)
	if crc1 != crc2 {
		t.Fatalf("same inputs produced different CRCs: %q vs %q", crc1, crc2)
	}
	if len(crc1) != 8 {
		t.Fatalf("CRC length = %d, want 8", len(crc1))
	}

	other := computeCRC(machineID, keyID, "different-secret")
	if other == crc1 {
		t.Fatalf("different secret produced same CRC %q", other)
	}
}

func TestParseAPIKeyNewAndLegacy(t *testing.T) {
	machineID := "deadbeefdeadbeef"
	key, keyID, err := GenerateAPIKey(machineID)
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}

	parsed, err := ParseAPIKey(key)
	if err != nil {
		t.Fatalf("ParseAPIKey new: %v", err)
	}
	if parsed == nil {
		t.Fatal("ParseAPIKey new returned nil")
	}
	if !parsed.IsNewFormat {
		t.Fatal("new format key parsed as legacy")
	}
	if parsed.MachineID != machineID {
		t.Fatalf("MachineID = %q, want %q", parsed.MachineID, machineID)
	}
	if parsed.KeyID != keyID {
		t.Fatalf("KeyID = %q, want %q", parsed.KeyID, keyID)
	}

	// CRC corruption invalidates new format.
	corrupt := key[:len(key)-1] + "0"
	if corrupt == key {
		corrupt = key[:len(key)-1] + "1"
	}
	if p, _ := ParseAPIKey(corrupt); p != nil {
		t.Fatalf("corrupted key should be invalid, got %+v", p)
	}

	// Legacy format: sk-{random8}
	legacy := "sk-abcdef12"
	parsed, err = ParseAPIKey(legacy)
	if err != nil {
		t.Fatalf("ParseAPIKey legacy: %v", err)
	}
	if parsed == nil {
		t.Fatal("ParseAPIKey legacy returned nil")
	}
	if parsed.IsNewFormat {
		t.Fatal("legacy key parsed as new format")
	}
	if parsed.KeyID != "abcdef12" {
		t.Fatalf("legacy KeyID = %q, want %q", parsed.KeyID, "abcdef12")
	}

	// Invalid formats.
	for _, bad := range []string{"", "not-a-key", "sk-", "sk-a-b-c-d-e"} {
		if p, _ := ParseAPIKey(bad); p != nil {
			t.Fatalf("%q should be invalid, got %+v", bad, p)
		}
	}
}

func TestParseAPIKeyMalformed(t *testing.T) {
	machineID := "deadbeefdeadbeef"
	validKey, _, err := GenerateAPIKey(machineID)
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if p, _ := ParseAPIKey(validKey); p == nil {
		t.Fatalf("valid key %q should parse", validKey)
	}

	cases := []string{
		// New-format segment-length and character violations.
		"sk-deadbeefdeadbeef-abc12-12345678",   // keyID too short
		"sk-deadbeefdeadbeef-abc1234-12345678", // keyID too long
		"sk-deadbeefdeadbeef-abc123-1234567",   // CRC too short
		"sk-deadbeefdeadbeef-abc123-123456789", // CRC too long
		"sk-DEADBEEFDEADBEEF-abc123-12345678",  // machineID uppercase
		"sk-deadbeefdeadbeef-ABC123-12345678",  // keyID uppercase
		"sk-deadbeefdeadbeef-abc123-1234567G",  // CRC non-hex
		"sk-deadbeefdeadbeef--12345678",        // empty keyID
		"sk--abc123-12345678",                  // empty machineID
		"sk-deadbeefdeadbeef-abc123-",          // empty CRC
		// Legacy segment-length and character violations.
		"sk-abc1234",   // too short
		"sk-abc123456", // too long
		"sk-ABCDEF12",  // uppercase
		"sk-abc@123",   // invalid character
		// Extra/missing parts.
		"sk-deadbeefdeadbeef-abc123-12345678-extra",
		"sk-deadbeefdeadbeef-abc123",
	}
	for _, bad := range cases {
		if p, err := ParseAPIKey(bad); p != nil {
			t.Fatalf("%q should be invalid, got %+v (err=%v)", bad, p, err)
		}
	}
}

func TestMachineIDDerivation(t *testing.T) {
	dir := t.TempDir()

	id1, err := MachineID(dir, "salt-one")
	if err != nil {
		t.Fatalf("MachineID: %v", err)
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{16}$`, id1); !matched {
		t.Fatalf("MachineID %q is not 16 hex chars", id1)
	}

	id2, err := MachineID(dir, "salt-two")
	if err != nil {
		t.Fatalf("MachineID salt-two: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("different salts produced same id %q", id1)
	}

	// Non-CLI salt should not depend on cli-secret.
	dirA := t.TempDir()
	dirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirA, "machine-id"), []byte("same-raw-id"), 0o600); err != nil {
		t.Fatalf("write machine-id A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "machine-id"), []byte("same-raw-id"), 0o600); err != nil {
		t.Fatalf("write machine-id B: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirA, "auth", "cli-secret"), []byte("secret-a"), 0o600); err != nil {
		if err := os.MkdirAll(filepath.Join(dirA, "auth"), 0o700); err != nil {
			t.Fatalf("mkdir auth A: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dirA, "auth", "cli-secret"), []byte("secret-a"), 0o600); err != nil {
			t.Fatalf("write cli-secret A: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dirB, "auth"), 0o700); err != nil {
		t.Fatalf("mkdir auth B: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "auth", "cli-secret"), []byte("secret-b"), 0o600); err != nil {
		t.Fatalf("write cli-secret B: %v", err)
	}

	nonCliA, err := MachineID(dirA, "some-salt")
	if err != nil {
		t.Fatalf("MachineID non-cli A: %v", err)
	}
	nonCliB, err := MachineID(dirB, "some-salt")
	if err != nil {
		t.Fatalf("MachineID non-cli B: %v", err)
	}
	if nonCliA != nonCliB {
		t.Fatalf("non-cli salt varied with cli-secret: %q vs %q", nonCliA, nonCliB)
	}

	// CLI salt must mix cli-secret.
	cliA, err := MachineID(dirA, "9r-cli-auth")
	if err != nil {
		t.Fatalf("MachineID cli A: %v", err)
	}
	cliB, err := MachineID(dirB, "9r-cli-auth")
	if err != nil {
		t.Fatalf("MachineID cli B: %v", err)
	}
	if cliA == cliB {
		t.Fatalf("cli salt ignored cli-secret: both %q", cliA)
	}
}

func TestMachineIDStable(t *testing.T) {
	dir := t.TempDir()

	id1, err := MachineID(dir, "stable-salt")
	if err != nil {
		t.Fatalf("MachineID first: %v", err)
	}
	id2, err := MachineID(dir, "stable-salt")
	if err != nil {
		t.Fatalf("MachineID second: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("MachineID not stable: %q vs %q", id1, id2)
	}
}

// TestKeyIDGenerationNoPlaceholder verifies that randomUUID returns an error
// instead of minting the deterministic placeholder when crypto/rand fails.
func TestKeyIDGenerationNoPlaceholder(t *testing.T) {
	prev := randReader
	t.Cleanup(func() { randReader = prev })

	randReader = &failingReader{}
	_, err := randomUUID()
	if err == nil {
		t.Fatal("randomUUID: want error with failing rand reader")
	}
}

// failingReader always returns an error so tests can exercise rand failures.
type failingReader struct{}

func (f *failingReader) Read(p []byte) (int, error) {
	return 0, errors.New("simulated rand failure")
}

// TestGenerateAPIKeyNoPlaceholder verifies that a broken rand reader during
// key ID generation returns an error and never produces the placeholder UUID.
func TestGenerateAPIKeyNoPlaceholder(t *testing.T) {
	prev := randRead
	t.Cleanup(func() { randRead = prev })

	randRead = func(b []byte) (int, error) {
		return 0, errors.New("simulated rand failure")
	}
	key, keyID, err := GenerateAPIKey("deadbeefdeadbeef")
	if err == nil {
		t.Fatalf("GenerateAPIKey: want error, got key=%q keyID=%q", key, keyID)
	}
	if strings.Contains(key, "0000000000000000") || strings.Contains(keyID, "0000000000000000") {
		t.Errorf("output contains placeholder UUID: key=%q keyID=%q", key, keyID)
	}
}
