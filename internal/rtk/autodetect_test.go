package rtk

import "testing"

func TestDetectFormatRealisticToolOutputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ContentFormat
	}{
		{
			name: "json",
			input: `{
  "status": "ok",
  "items": [{"path": "internal/rtk/autodetect.go", "score": 0.97}]
}`,
			want: FormatJSON,
		},
		{
			name: "unified diff",
			input: `diff --git a/internal/rtk/autodetect.go b/internal/rtk/autodetect.go
index 1234567..89abcde 100644
--- a/internal/rtk/autodetect.go
+++ b/internal/rtk/autodetect.go
@@ -1,3 +1,4 @@
 package rtk
+const answer = 42
`,
			want: FormatGitDiff,
		},
		{
			name: "git status",
			input: `On branch codex/wave-2a-task-71
Changes not staged for commit:
  modified:   internal/rtk/autodetect.go
Untracked files:
  internal/rtk/autodetect_test.go
`,
			want: FormatGitStatus,
		},
		{
			name: "grep output",
			input: `internal/rtk/autodetect.go:12:func DetectFormat(input string) ContentFormat
docs/REFERENCES.md:85:| open-sse/rtk/autodetect.js | internal/rtk/autodetect.go |
`,
			want: FormatGrep,
		},
		{
			name: "find output",
			input: `./internal/rtk/autodetect.go
./internal/rtk/constants.go
./internal/rtk/filters/gitdiff.go
`,
			want: FormatFind,
		},
		{
			name: "ls output",
			input: `total 16
drwxr-xr-x  4 heitor  staff   128 Jun  2 14:10 .
-rw-r--r--  1 heitor  staff  1024 Jun  2 14:10 autodetect.go
`,
			want: FormatLS,
		},
		{
			name: "tree output",
			input: `internal
├── rtk
│   ├── autodetect.go
│   └── constants.go
└── store
    └── sqlite.go
`,
			want: FormatTree,
		},
		{
			name: "build output",
			input: `# github.com/bloodf/g0router/internal/rtk
internal/rtk/autodetect.go:14:9: undefined: FormatJSON
FAIL	github.com/bloodf/g0router/internal/rtk [build failed]
`,
			want: FormatBuildOutput,
		},
		{
			name: "numbered read output",
			input: `     1	package rtk
     2
     3	func DetectFormat(input string) ContentFormat {
`,
			want: FormatReadNumbered,
		},
		{
			name: "markdown",
			input: `# Phase 7

## Task 7.1

- Write tests first
- Detect tool output formats
`,
			want: FormatMarkdown,
		},
		{
			name:  "plain text fallback",
			input: "plain command output without a stronger structural signal",
			want:  FormatPlainText,
		},
		{
			name:  "empty",
			input: " \n\t",
			want:  FormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.input)
			if got != tt.want {
				t.Fatalf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectFormatInspectsOnlyFirstKilobyte(t *testing.T) {
	input := repeatString("plain text\n", 120) + `diff --git a/late b/late
--- a/late
+++ b/late
@@ -1 +1 @@
-old
+new
`

	got := DetectFormat(input)
	if got != FormatPlainText {
		t.Fatalf("DetectFormat() = %q, want %q", got, FormatPlainText)
	}
}

func TestContentFormatString(t *testing.T) {
	if FormatGitDiff.String() != "git_diff" {
		t.Fatalf("String() = %q, want git_diff", FormatGitDiff.String())
	}
}

func repeatString(value string, count int) string {
	var result string
	for i := 0; i < count; i++ {
		result += value
	}
	return result
}
