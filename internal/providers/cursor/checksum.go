package cursor

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

// checksumAlphabet is the URL-safe base64 alphabet used by the Jyh cipher
// (cursorChecksum.js generateCursorChecksum).
const checksumAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// generateHashed64Hex returns the SHA-256 hex digest of input+salt
// (cursorChecksum.js generateHashed64Hex).
func generateHashed64Hex(input, salt string) string {
	sum := sha256.Sum256([]byte(input + salt))
	return hex.EncodeToString(sum[:])
}

// generateSessionID returns a UUID v5 (DNS namespace) derived from the token
// (cursorChecksum.js generateSessionId).
func generateSessionID(authToken string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(authToken)).String()
}

// generateCursorChecksum produces the x-cursor-checksum value (Jyh cipher)
// for the given machine id and timestamp (cursorChecksum.js
// generateCursorChecksum). The timestamp is Math.floor(Date.now()/1e6) in the
// ref; it is passed in here for determinism/testability.
func generateCursorChecksum(machineID string, timestamp int64) string {
	// 6-byte big-endian timestamp.
	b := []byte{
		byte((timestamp >> 40) & 0xff),
		byte((timestamp >> 32) & 0xff),
		byte((timestamp >> 24) & 0xff),
		byte((timestamp >> 16) & 0xff),
		byte((timestamp >> 8) & 0xff),
		byte(timestamp & 0xff),
	}

	// Jyh cipher obfuscation.
	t := byte(165)
	for i := 0; i < len(b); i++ {
		b[i] = (b[i] ^ t) + byte(i%256)
		t = b[i]
	}

	// URL-safe base64 (no padding), matching the ref's manual encoder.
	var sb strings.Builder
	for i := 0; i < len(b); i += 3 {
		a := b[i]
		var bb, c byte
		if i+1 < len(b) {
			bb = b[i+1]
		}
		if i+2 < len(b) {
			c = b[i+2]
		}
		sb.WriteByte(checksumAlphabet[a>>2])
		sb.WriteByte(checksumAlphabet[((a&3)<<4)|(bb>>4)])
		if i+1 < len(b) {
			sb.WriteByte(checksumAlphabet[((bb&15)<<2)|(c>>6)])
		}
		if i+2 < len(b) {
			sb.WriteByte(checksumAlphabet[c&63])
		}
	}

	return sb.String() + machineID
}

// buildCursorHeaders assembles the Cursor API headers (cursorChecksum.js
// buildCursorHeaders). The token is cleaned of any "::" prefix; the machine id,
// session id, client key, and checksum are derived deterministically. The
// timestamp is passed in for the checksum.
func buildCursorHeaders(accessToken, machineID string, ghostMode bool, timestamp int64) map[string]string {
	cleanToken := accessToken
	if i := strings.Index(accessToken, "::"); i >= 0 {
		cleanToken = accessToken[i+2:]
	}

	effectiveMachineID := machineID
	if effectiveMachineID == "" {
		effectiveMachineID = generateHashed64Hex(cleanToken, "machineId")
	}

	sessionID := generateSessionID(cleanToken)
	clientKey := generateHashed64Hex(cleanToken, "")
	checksum := generateCursorChecksum(effectiveMachineID, timestamp)

	ghost := "false"
	if ghostMode {
		ghost = "true"
	}

	return map[string]string{
		"authorization":            "Bearer " + cleanToken,
		"connect-accept-encoding":  "gzip",
		"connect-protocol-version": "1",
		"content-type":             "application/connect+proto",
		"user-agent":               "connect-es/1.6.1",
		"x-amzn-trace-id":          "Root=" + newUUID(),
		"x-client-key":             clientKey,
		"x-cursor-checksum":        checksum,
		"x-cursor-client-version":  "3.1.0",
		"x-cursor-client-type":     "ide",
		"x-cursor-config-version":  newUUID(),
		"x-ghost-mode":             ghost,
		"x-request-id":             newUUID(),
		"x-session-id":             sessionID,
	}
}
