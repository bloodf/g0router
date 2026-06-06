package update

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		current string
		latest  string
		want    bool
	}{
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"1.0.0", "2.0.0", true},
		{"0.9.0", "1.0.0", true},
		{"1.0.0", "1.0.1", true},
		{"v1.0.0", "v1.1.0", true},
		{"1.0.0", "v1.1.0", true},
		{"v1.0.0", "1.1.0", true},
		{"1.0.0-alpha", "1.0.0", true},
		{"1.0.0", "1.0.0-alpha", false},
	}

	for _, tc := range cases {
		got := isNewer(tc.current, tc.latest)
		if got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"v2.1.3-beta", "2.1.3-beta"},
	}

	for _, tc := range cases {
		got := normalizeVersion(tc.input)
		if got != tc.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
