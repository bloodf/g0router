// Package semcache implements g0router's phase-19 semantic cache. bf-core-2
// ships ONLY the deterministic exact-key-hash half: a sha256(normalized prompt
// + model) key feeds an O(1) SQLite lookup that short-circuits the provider on
// a hit and writes through on a miss. The semantic-similarity (cosine over
// embeddings) half is deferred — there is no embedder, no cosine engine, and no
// semantic branch in this package (see plan bf-core-2 §0/§2/D2).
package semcache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// keyVersion namespaces the hash so a future normalization change can rotate
// keys without colliding with existing rows.
const keyVersion = "v1"

// CacheKey returns the exact-key cache key for (model, prompt): the hex sha256
// of a deterministic, normalized encoding of the model and prompt (D1).
//
// Normalization trims insignificant surrounding whitespace from each field so
// semantically identical inputs map to the same key. A newline separator keeps
// the model and prompt fields distinct, so character shifts across the boundary
// cannot forge a collision. The normalization choice is recorded in
// open-questions (a canonical message-shape form can replace it later by
// bumping keyVersion).
func CacheKey(model, prompt string) string {
	normModel := strings.TrimSpace(model)
	normPrompt := strings.TrimSpace(prompt)
	sum := sha256.Sum256([]byte(keyVersion + "\n" + normModel + "\n" + normPrompt))
	return hex.EncodeToString(sum[:])
}
