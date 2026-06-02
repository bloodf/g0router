package filters

import (
	"strings"
	"testing"
)

func TestNamedFiltersIncludesElevenComposableFilters(t *testing.T) {
	got := NamedFilters()
	wantNames := []string{
		"git_diff",
		"git_status",
		"grep",
		"find",
		"ls",
		"tree",
		"build_output",
		"dedup_log",
		"smart_truncate",
		"read_numbered",
		"search_list",
	}

	if len(got) != len(wantNames) {
		t.Fatalf("NamedFilters len = %d, want %d", len(got), len(wantNames))
	}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Fatalf("NamedFilters()[%d].Name = %q, want %q", i, got[i].Name, want)
		}
		if got[i].Filter == nil {
			t.Fatalf("NamedFilters()[%d].Filter is nil", i)
		}
	}
}

func TestGitDiffKeepsPatchShapeAndDropsMetadata(t *testing.T) {
	input := `diff --git a/internal/rtk/autodetect.go b/internal/rtk/autodetect.go
index 1234567..89abcde 100644
--- a/internal/rtk/autodetect.go
+++ b/internal/rtk/autodetect.go
@@ -1,5 +1,6 @@
 package rtk
 
 import "strings"
+const detectSampleBytes = 1024
 func DetectFormat(input string) ContentFormat {
 	return FormatPlainText
 }
`

	got := GitDiff(input)
	want := `diff --git a/internal/rtk/autodetect.go b/internal/rtk/autodetect.go
--- a/internal/rtk/autodetect.go
+++ b/internal/rtk/autodetect.go
@@ -1,5 +1,6 @@
 package rtk
 import "strings"
+const detectSampleBytes = 1024
 func DetectFormat(input string) ContentFormat {
 return FormatPlainText
 }`

	if got != want {
		t.Fatalf("GitDiff() =\n%s\nwant\n%s", got, want)
	}
}

func TestGitStatusSummarizesChangedPaths(t *testing.T) {
	input := `On branch codex/wave-2b-task-72
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  modified:   internal/rtk/filters/gitdiff.go
  deleted:    internal/rtk/filters/old.go

Untracked files:
  internal/rtk/filters/filters_test.go
`

	got := GitStatus(input)
	want := `branch codex/wave-2b-task-72
M internal/rtk/filters/gitdiff.go
D internal/rtk/filters/old.go
?? internal/rtk/filters/filters_test.go`

	if got != want {
		t.Fatalf("GitStatus() =\n%s\nwant\n%s", got, want)
	}
}

func TestGrepKeepsPathLineAndTrimmedMatch(t *testing.T) {
	input := `internal/rtk/autodetect.go:12:func DetectFormat(input string) ContentFormat
docs/REFERENCES.md:95:| open-sse/rtk/filters/buildOutput.js | internal/rtk/filters/buildoutput.go |
`

	got := Grep(input)
	want := `internal/rtk/autodetect.go:12 func DetectFormat(input string) ContentFormat
docs/REFERENCES.md:95 | open-sse/rtk/filters/buildOutput.js | internal/rtk/filters/buildoutput.go |`

	if got != want {
		t.Fatalf("Grep() =\n%s\nwant\n%s", got, want)
	}
}

func TestFindKeepsOnlyUsefulPaths(t *testing.T) {
	input := `.
./internal/rtk/autodetect.go
./internal/rtk/constants.go
./internal/rtk/filters
./internal/rtk/filters/gitdiff.go
`

	got := Find(input)
	want := `internal/rtk/autodetect.go
internal/rtk/constants.go
internal/rtk/filters
internal/rtk/filters/gitdiff.go`

	if got != want {
		t.Fatalf("Find() =\n%s\nwant\n%s", got, want)
	}
}

func TestLSKeepsNamesAndMarksDirectories(t *testing.T) {
	input := `total 16
drwxr-xr-x  4 heitor  staff   128 Jun  2 14:10 filters
-rw-r--r--  1 heitor  staff  1024 Jun  2 14:10 autodetect.go
-rw-r--r--  1 heitor  staff   256 Jun  2 14:10 constants.go
`

	got := LS(input)
	want := `filters/
autodetect.go
constants.go`

	if got != want {
		t.Fatalf("LS() =\n%s\nwant\n%s", got, want)
	}
}

func TestTreeRemovesDrawingCharacters(t *testing.T) {
	input := `internal
├── rtk
│   ├── autodetect.go
│   └── constants.go
└── store
    └── sqlite.go
`

	got := Tree(input)
	want := `internal
rtk
autodetect.go
constants.go
store
sqlite.go`

	if got != want {
		t.Fatalf("Tree() =\n%s\nwant\n%s", got, want)
	}
}

func TestBuildOutputKeepsFailuresAndDiagnostics(t *testing.T) {
	input := `ok  	github.com/bloodf/g0router/internal/config	0.120s
# github.com/bloodf/g0router/internal/rtk
internal/rtk/filters/gitdiff.go:14:9: undefined: keepLine
FAIL	github.com/bloodf/g0router/internal/rtk [build failed]
?   	github.com/bloodf/g0router/cmd/g0router	[no test files]
`

	got := BuildOutput(input)
	want := `# github.com/bloodf/g0router/internal/rtk
internal/rtk/filters/gitdiff.go:14:9: undefined: keepLine
FAIL github.com/bloodf/g0router/internal/rtk [build failed]`

	if got != want {
		t.Fatalf("BuildOutput() =\n%s\nwant\n%s", got, want)
	}
}

func TestDedupLogRemovesRepeatedMessages(t *testing.T) {
	input := `2026-06-02T14:10:00Z INFO worker started id=1
2026-06-02T14:10:01Z INFO worker started id=1
2026-06-02T14:10:02Z WARN retrying provider=openai attempt=1
2026-06-02T14:10:03Z WARN retrying provider=openai attempt=1
2026-06-02T14:10:04Z ERROR request failed status=500
`

	got := DedupLog(input)
	want := `INFO worker started id=1
WARN retrying provider=openai attempt=1
ERROR request failed status=500`

	if got != want {
		t.Fatalf("DedupLog() =\n%s\nwant\n%s", got, want)
	}
}

func TestSmartTruncateLeavesSmallInputAloneAndMarksLargeInput(t *testing.T) {
	small := "short tool output"
	if got := SmartTruncate(small); got != small {
		t.Fatalf("SmartTruncate(small) = %q, want %q", got, small)
	}

	large := strings.Repeat("0123456789", 700)
	got := SmartTruncate(large)
	if len(got) >= len(large) {
		t.Fatalf("SmartTruncate length = %d, want less than %d", len(got), len(large))
	}
	if !strings.Contains(got, "\n... truncated ") {
		t.Fatalf("SmartTruncate() missing truncation marker: %q", got)
	}
	if !strings.HasPrefix(got, "0123456789") || !strings.HasSuffix(got, "0123456789") {
		t.Fatalf("SmartTruncate() should preserve both ends")
	}
}

func TestReadNumberedStripsLineNumbers(t *testing.T) {
	input := `     1	package rtk
     2
     3	func DetectFormat(input string) ContentFormat {
    42		return FormatPlainText
`

	got := ReadNumbered(input)
	want := `package rtk

func DetectFormat(input string) ContentFormat {
return FormatPlainText`

	if got != want {
		t.Fatalf("ReadNumbered() =\n%s\nwant\n%s", got, want)
	}
}

func TestSearchListStripsBulletsAndKeepsItems(t *testing.T) {
	input := `- internal/rtk/autodetect.go
* internal/rtk/constants.go
- docs/phases/phase-07-rtk-caveman.md
`

	got := SearchList(input)
	want := `internal/rtk/autodetect.go
internal/rtk/constants.go
docs/phases/phase-07-rtk-caveman.md`

	if got != want {
		t.Fatalf("SearchList() =\n%s\nwant\n%s", got, want)
	}
}
