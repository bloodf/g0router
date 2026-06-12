package usage

import (
	"strings"
	"testing"
)

func TestDedupeRecent(t *testing.T) {
	mk := func(ts string, prompt, completion int64) RecentRequest {
		return RecentRequest{Timestamp: ts, Model: "gpt-4o", Provider: "openai", PromptTokens: prompt, CompletionTokens: completion, Status: "ok"}
	}

	tests := []struct {
		name  string
		in    []RecentRequest
		wantN int
		want0 string
	}{
		{
			name: "zero tokens dropped",
			in: []RecentRequest{
				mk("2026-06-12T10:00:00Z", 0, 0),
				mk("2026-06-12T10:00:01Z", 10, 5),
			},
			wantN: 1,
			want0: "2026-06-12T10:00:01Z",
		},
		{
			name: "same minute duplicate collapsed",
			in: []RecentRequest{
				mk("2026-06-12T10:00:05Z", 10, 5),
				mk("2026-06-12T10:00:55Z", 10, 5),
				mk("2026-06-12T10:01:00Z", 10, 5),
			},
			wantN: 2,
			want0: "2026-06-12T10:01:00Z",
		},
		{
			name: "different tokens kept",
			in: []RecentRequest{
				mk("2026-06-12T10:00:00Z", 10, 5),
				mk("2026-06-12T10:00:00Z", 20, 5),
				mk("2026-06-12T10:00:00Z", 10, 10),
			},
			wantN: 3,
			want0: "2026-06-12T10:00:00Z",
		},
		{
			name: "caps at 20",
			in: func() []RecentRequest {
				var out []RecentRequest
				for i := 0; i < 25; i++ {
					out = append(out, mk("2026-06-12T10:00:00Z", int64(i+1), 1))
				}
				return out
			}(),
			wantN: 20,
			want0: "2026-06-12T10:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DedupeRecent(tt.in)
			if len(got) != tt.wantN {
				t.Errorf("len = %d, want %d", len(got), tt.wantN)
			}
			if tt.wantN > 0 && got[0].Timestamp != tt.want0 {
				t.Errorf("first timestamp = %q, want %q", got[0].Timestamp, tt.want0)
			}
		})
	}
}

func TestDedupeRecentSortsNewestFirst(t *testing.T) {
	in := []RecentRequest{
		{Timestamp: "2026-06-12T09:00:00Z", Model: "m", Provider: "p", PromptTokens: 1, CompletionTokens: 1},
		{Timestamp: "2026-06-12T11:00:00Z", Model: "m", Provider: "p", PromptTokens: 1, CompletionTokens: 1},
		{Timestamp: "2026-06-12T10:00:00Z", Model: "m", Provider: "p", PromptTokens: 1, CompletionTokens: 1},
	}
	got := DedupeRecent(in)
	if got[0].Timestamp != "2026-06-12T11:00:00Z" {
		t.Errorf("first = %q, want newest", got[0].Timestamp)
	}
	if !strings.HasPrefix(got[0].Timestamp, "2026-06-12T11") {
		t.Errorf("newest-first sort failed: %v", got)
	}
}
