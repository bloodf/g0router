package rtk

import (
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func toolStringMsg(content string) providers.ChatRequest {
	return providers.ChatRequest{
		Messages: []providers.Message{{Role: "tool", Content: content}},
	}
}

func compressedToolContent(t *testing.T, content string) string {
	t.Helper()
	out := CompressRequest(toolStringMsg(content))
	got, ok := out.Messages[0].Content.(string)
	if !ok {
		t.Fatalf("content not string: %T", out.Messages[0].Content)
	}
	return got
}

// Each format branch in compressContent is exercised by feeding tool content
// that DetectFormat classifies as that format.
func TestCompressContentAllFormatBranches(t *testing.T) {
	cases := map[string]string{
		"gitdiff":      "diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n",
		"gitstatus":    "On branch main\nChanges not staged for commit:\n\tmodified: x\n",
		"grep":         "file.go:12:func main\nfile.go:34:return\n",
		"find":         "./a\n./b\n./c\n",
		"ls":           "total 8\ndrwxr-xr-x 2 u g 64 Jan 1 x\n-rw-r--r-- 1 u g 10 Jan 1 y\n",
		"tree":         "root\n├── a\n└── b\n",
		"buildoutput":  "# github.com/x/y\nundefined: Foo\n[build failed]\n",
		"log":          "[INFO] starting\n[WARN] slow\n[ERROR] boom\n",
		"readnumbered": "   1\tpackage x\n   2\tfunc y(){}\n",
		"searchlist":   "- one\n- two\n- three\n",
		"plaintext":    "just some words here without structure",
	}
	for name, content := range cases {
		got := compressedToolContent(t, content)
		if got == "" && content != "" {
			t.Errorf("%s: compressed content empty", name)
		}
	}
}

func TestCompressStringBlocks(t *testing.T) {
	blocks := []map[string]string{
		{"type": "tool_result", "content": "file.go:1:hit\nfile.go:2:hit\n"},
		{"type": "text", "content": "leave me"},
	}
	req := providers.ChatRequest{Messages: []providers.Message{{Role: "user", Content: blocks}}}
	out := CompressRequest(req)
	got, ok := out.Messages[0].Content.([]map[string]string)
	if !ok {
		t.Fatalf("content type = %T", out.Messages[0].Content)
	}
	if got[1]["content"] != "leave me" {
		t.Fatalf("non-tool_result block modified: %q", got[1]["content"])
	}
	// Original blocks must be untouched (immutability).
	if blocks[0]["content"] == got[0]["content"] && strings.Contains(blocks[0]["content"], "\n\n") {
		t.Fatal("unexpected")
	}
}

func TestCompressMixedBlocks(t *testing.T) {
	blocks := []any{
		map[string]any{"type": "tool_result", "content": "diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n"},
		map[string]any{"type": "text", "content": "keep"},
		"raw string element",
		map[string]any{"type": "tool_result", "content": 123}, // non-string content untouched
	}
	req := providers.ChatRequest{Messages: []providers.Message{{Role: "user", Content: blocks}}}
	out := CompressRequest(req)
	got, ok := out.Messages[0].Content.([]any)
	if !ok {
		t.Fatalf("content type = %T", out.Messages[0].Content)
	}
	if got[1].(map[string]any)["content"] != "keep" {
		t.Fatal("text block modified")
	}
	if got[2] != "raw string element" {
		t.Fatal("raw element modified")
	}
	if got[3].(map[string]any)["content"] != 123 {
		t.Fatal("non-string tool_result content modified")
	}
}

func TestCompressToolResultBlocksDefault(t *testing.T) {
	// Unknown content type returned as-is.
	req := providers.ChatRequest{Messages: []providers.Message{{Role: "user", Content: 42}}}
	out := CompressRequest(req)
	if out.Messages[0].Content != 42 {
		t.Fatalf("content = %v, want 42", out.Messages[0].Content)
	}
}

func TestDetectFormatLogVariants(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  ContentFormat
	}{
		{"bracket", "[ERROR] something failed\nmore detail here\n", FormatLog},
		{"prefix", "INFO server started\nWARN slow query\n", FormatLog},
		{"isodate", "2026-01-02T starting up here now\nrunning along fine today\n", FormatLog},
	}
	for _, tc := range cases {
		if got := DetectFormat(tc.input); got != tc.want {
			t.Errorf("%s: DetectFormat = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDetectFormatReadNumberedTabSeparated(t *testing.T) {
	// digit run followed by tab -> read numbered.
	if got := DetectFormat("42\tcontent line\n"); got != FormatReadNumbered {
		t.Fatalf("DetectFormat = %q, want read_numbered", got)
	}
}

func TestAllDigitsEmpty(t *testing.T) {
	if allDigits("") {
		t.Fatal("allDigits(\"\") = true, want false")
	}
	if allDigits("12a") {
		t.Fatal("allDigits(\"12a\") = true, want false")
	}
	if !allDigits("12345") {
		t.Fatal("allDigits(\"12345\") = false, want true")
	}
}

func TestIsGrepRejectsNonNumericLineField(t *testing.T) {
	// "a:b:c" has non-digit second field -> not grep, classified otherwise.
	if got := DetectFormat("a:bcd:efg\nx:yz:w\n"); got == FormatGrep {
		t.Fatal("should not classify non-numeric as grep")
	}
}

func TestIsSearchListRequiresAllBullets(t *testing.T) {
	// Mixed bullet/non-bullet -> not search list.
	if got := DetectFormat("- one\nnot a bullet\n"); got == FormatSearchList {
		t.Fatal("mixed lines should not be search_list")
	}
	// Star bullets, two lines -> search list.
	if got := DetectFormat("* alpha\n* beta\n"); got != FormatSearchList {
		t.Fatalf("star bullets = %q, want search_list", got)
	}
	// Single bullet line -> too few lines.
	if got := DetectFormat("- only one"); got == FormatSearchList {
		t.Fatal("single line should not be search_list")
	}
}

func TestDetectFormatStructuredFormats(t *testing.T) {
	cases := []struct {
		input string
		want  ContentFormat
	}{
		{"<?xml version=\"1.0\"?>\n<root/>", FormatXML},
		{"<!DOCTYPE html><html><body>x</body></html>", FormatHTML},
		{"<html><head></head></html>", FormatHTML},
		{"# Title\nsome prose body text here\n", FormatMarkdown},
	}
	for _, tc := range cases {
		if got := DetectFormat(tc.input); got != tc.want {
			t.Errorf("DetectFormat(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDetectFormatEmptyAndWhitespace(t *testing.T) {
	if got := DetectFormat("   \n\t  "); got != FormatUnknown {
		t.Fatalf("whitespace-only = %q, want unknown", got)
	}
}

func TestIsGrepEdgeCases(t *testing.T) {
	// Colon at end of line (first == len-1) -> skip; line-number empty after.
	if got := DetectFormat("label:\nplain text follows here\n"); got == FormatGrep {
		t.Fatal("trailing colon should not be grep")
	}
	// No second colon -> skip.
	if got := DetectFormat("key: value only one colon here\n"); got == FormatGrep {
		t.Fatal("single colon should not be grep")
	}
}

func TestIsLogFalseForPlainProse(t *testing.T) {
	if got := DetectFormat("the quick brown fox jumps\nover the lazy dog today\n"); got == FormatLog {
		t.Fatal("plain prose should not be log")
	}
}
