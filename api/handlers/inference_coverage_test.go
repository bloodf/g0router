package handlers

import (
	"testing"
)

func TestBytesTrimSpace(t *testing.T) {
	got := bytesTrimSpace([]byte("  hello  "))
	if string(got) != "hello" {
		t.Fatalf("bytesTrimSpace = %q, want hello", got)
	}
}
