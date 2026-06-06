package cli

import (
	"path/filepath"
	"testing"
)

func TestCopyFileNotFound(t *testing.T) {
	err := copyFile("/nonexistent/path/to/file", filepath.Join(t.TempDir(), "dst"), 0644)
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}
