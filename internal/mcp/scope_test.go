package mcp

import (
	"reflect"
	"sort"
	"testing"
)

// names extracts the tool names from a []ServerTool for set comparison.
func names(tools []ServerTool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name)
	}
	sort.Strings(out)
	return out
}

// TestScopeTools proves the executeOnlyTools wildcard filter (D4): deny-empty,
// "*"=all, "<client>-*" prefix wildcard, exact "<client>-<tool>", bare "<tool>",
// and an unknown pattern matching nothing. The clientOf map says which client
// owns each (bare) tool name.
func TestScopeTools(t *testing.T) {
	global := []ServerTool{
		{Name: "search"},  // owned by exa
		{Name: "fetch"},   // owned by exa
		{Name: "read"},    // owned by fs
		{Name: "write"},   // owned by fs
	}
	clientOf := map[string]string{
		"search": "exa",
		"fetch":  "exa",
		"read":   "fs",
		"write":  "fs",
	}
	of := func(tool string) string { return clientOf[tool] }

	cases := []struct {
		name     string
		patterns []string
		want     []string
	}{
		{"nil deny-all", nil, []string{}},
		{"empty deny-all", []string{}, []string{}},
		{"star all", []string{"*"}, []string{"fetch", "read", "search", "write"}},
		{"client prefix wildcard", []string{"exa-*"}, []string{"fetch", "search"}},
		{"exact client-tool", []string{"exa-search"}, []string{"search"}},
		{"bare tool", []string{"read"}, []string{"read"}},
		{"mixed", []string{"exa-search", "fs-*"}, []string{"read", "search", "write"}},
		{"unknown no match", []string{"nope-*", "ghost"}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := names(scopeTools(global, tc.patterns, of))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("scopeTools(%v) = %v, want %v", tc.patterns, got, tc.want)
			}
		})
	}
}

// TestScopeToolsNarrows proves a restricted pattern set yields STRICTLY FEWER
// tools than the global catalog (the live-narrowing invariant, D3/D4).
func TestScopeToolsNarrows(t *testing.T) {
	global := []ServerTool{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	of := func(string) string { return "x" }
	got := scopeTools(global, []string{"a"}, of)
	if len(got) >= len(global) {
		t.Fatalf("restricted scope did not narrow: got %d, global %d", len(got), len(global))
	}
}

// TestValidateAutoExecuteSubset proves auto-execute ⊄ execute is rejected (D5/049)
// and a "*" execute admits any auto-execute.
func TestValidateAutoExecuteSubset(t *testing.T) {
	cases := []struct {
		name        string
		execute     []string
		autoExecute []string
		wantErr     bool
	}{
		{"empty both ok", nil, nil, false},
		{"subset ok", []string{"a", "b"}, []string{"a"}, false},
		{"equal ok", []string{"a"}, []string{"a"}, false},
		{"star admits any", []string{"*"}, []string{"a", "b-x"}, false},
		{"not subset rejected", []string{"a"}, []string{"a", "b"}, true},
		{"auto without execute rejected", nil, []string{"a"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAutoExecuteSubset(tc.execute, tc.autoExecute)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateAutoExecuteSubset(%v,%v) err=%v, wantErr=%v",
					tc.execute, tc.autoExecute, err, tc.wantErr)
			}
		})
	}
}
