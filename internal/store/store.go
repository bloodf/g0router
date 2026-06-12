package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("store: not found")

// Store wraps the SQLite database and the at-rest cipher for secret columns.
type Store struct {
	db              *sql.DB
	cipher          *Cipher
	apiKeyGenerator apiKeyGenerator
}

// Open opens (or creates) the SQLite database at path, enables WAL mode,
// and runs the additive-only migrations. secret is the 32-byte key used to
// encrypt *_enc columns.
func Open(path string, secret []byte) (*Store, error) {
	cipher, err := NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	// SQLite handles one writer at a time; serialize access through a
	// single connection to avoid SQLITE_BUSY under concurrent handlers.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db, cipher: cipher, apiKeyGenerator: defaultAPIKeyGenerator}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying *sql.DB (used by tests and future repositories).
func (s *Store) DB() *sql.DB {
	return s.db
}

// randRead is a seam for injecting crypto/rand failures in tests.
var randRead = rand.Read

// newID returns a random 32-char hex identifier.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
