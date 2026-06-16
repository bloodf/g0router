package store

import "testing"

// TestSHA256Hex pins sha256hex to the known lowercase-hex SHA-256 digest of a
// fixed input and verifies it is deterministic (the lookup hash for VK values).
func TestSHA256Hex(t *testing.T) {
	const input = "g0vk-abc"
	const want = "0af71e4c7f251cd41b56b6d1f1f4cddcc77d81be6b0f14abcc4ca3335c19e271"

	got := sha256hex(input)
	if got != want {
		t.Fatalf("sha256hex(%q) = %q, want %q", input, got, want)
	}
	if len(got) != 64 {
		t.Fatalf("digest length = %d, want 64", len(got))
	}
	if again := sha256hex(input); again != got {
		t.Fatalf("sha256hex not deterministic: %q != %q", again, got)
	}
}
