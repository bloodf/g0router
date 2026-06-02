package g0router_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/bloodf/g0router"
)

func TestUIIncludesBuiltDist(t *testing.T) {
	ui, err := g0router.UI()
	if err != nil {
		t.Fatalf("UI: %v", err)
	}

	body, err := fs.ReadFile(ui, "index.html")
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	content := string(body)
	if !strings.Contains(content, `<div id="root"></div>`) {
		t.Fatalf("index.html does not look like built UI: %q", content)
	}
}
