package store

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const secretFileName = "secret.key"

// LoadOrCreateSecret returns the 32-byte at-rest encryption key stored in
// dataDir/secret.key, generating it on first run. The key never leaves the
// data directory; no env vars are involved.
func LoadOrCreateSecret(dataDir string) ([]byte, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dataDir, err)
	}

	path := filepath.Join(dataDir, secretFileName)
	existing, err := os.ReadFile(path)
	if err == nil {
		if len(existing) != 32 {
			return nil, fmt.Errorf("secret file %s has %d bytes, want 32", path, len(existing))
		}
		return existing, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read secret file %s: %w", path, err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("write secret file %s: %w", path, err)
	}
	return key, nil
}
