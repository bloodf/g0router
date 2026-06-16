package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// sha256hex returns the lowercase hex-encoded SHA-256 digest of s. It is the
// deterministic lookup hash stored in virtual_keys.key (the reversible AES
// ciphertext of the raw value lives in key_enc).
func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Cipher encrypts and decrypts secret values stored in *_enc columns.
// AES-256-GCM with a random nonce prefixed to the ciphertext, base64-encoded.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher creates a Cipher from a 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("cipher key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt returns the base64(nonce || ciphertext) of plaintext.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short: %d bytes", len(raw))
	}
	plain, err := c.aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

// backfillEnc migrates legacy rows in `table` whose `encCol` is empty (''): it
// reads the plaintext `srcCol`, applies transform to produce the new src value
// and a reversible ciphertext, then writes both back in one UPDATE (encCol
// first, then srcCol). Idempotent via the encCol='' guard (already-migrated rows
// are skipped). Rows are collected before updating to avoid iterating a result
// set while writing on the single-conn DB. table/srcCol/encCol are compile-time
// constants at the call sites (not user input), so interpolating them is safe
// (SQL cannot bind identifiers).
func (s *Store) backfillEnc(table, srcCol, encCol string, transform func(raw string) (newSrc, enc string, err error)) error {
	rows, err := s.db.Query(fmt.Sprintf("SELECT id, %s FROM %s WHERE %s = ''", srcCol, table, encCol))
	if err != nil {
		return fmt.Errorf("query legacy %s: %w", table, err)
	}
	type legacy struct{ id, raw string }
	var pending []legacy
	for rows.Next() {
		var l legacy
		if err := rows.Scan(&l.id, &l.raw); err != nil {
			rows.Close()
			return fmt.Errorf("scan legacy %s: %w", table, err)
		}
		pending = append(pending, l)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate legacy %s: %w", table, err)
	}
	rows.Close()

	for _, l := range pending {
		newSrc, enc, err := transform(l.raw)
		if err != nil {
			return fmt.Errorf("encrypt legacy %s %s: %w", table, l.id, err)
		}
		if _, err := s.db.Exec(
			fmt.Sprintf("UPDATE %s SET %s = ?, %s = ? WHERE id = ?", table, encCol, srcCol),
			enc, newSrc, l.id,
		); err != nil {
			return fmt.Errorf("backfill %s %s: %w", table, l.id, err)
		}
	}
	return nil
}
