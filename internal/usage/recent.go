package usage

import (
	"fmt"
	"sort"
)

// RecentRequest is a single de-duplicated recent usage record.
type RecentRequest struct {
	Timestamp        string
	Model            string
	Provider         string
	PromptTokens     int64
	CompletionTokens int64
	Status           string
}

// DedupeRecent collapses identical model/provider/token/minute entries,
// drops zero-token rows, sorts newest-first, and caps the result at 20.
func DedupeRecent(in []RecentRequest) []RecentRequest {
	// Work on a copy so the input is not reordered.
	work := make([]RecentRequest, len(in))
	copy(work, in)

	sort.Slice(work, func(i, j int) bool {
		return work[i].Timestamp > work[j].Timestamp
	})

	seen := make(map[string]bool)
	out := make([]RecentRequest, 0, len(work))
	for _, r := range work {
		if r.PromptTokens == 0 && r.CompletionTokens == 0 {
			continue
		}
		minute := r.Timestamp
		if len(minute) > 16 {
			minute = minute[:16]
		}
		key := fmt.Sprintf("%s|%s|%d|%d|%s", r.Model, r.Provider, r.PromptTokens, r.CompletionTokens, minute)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
		if len(out) == 20 {
			break
		}
	}

	return out
}

