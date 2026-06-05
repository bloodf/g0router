package proxy

import (
	"fmt"
	"sort"
	"sync"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// autoHeavyContextThreshold is the approximate character count across all
// request messages above which the "auto" strategy treats a request as heavy
// context and routes it to the most capable (first) step.
const autoHeavyContextThreshold = 8000

// comboSelector holds mutable per-combo selection state shared across calls.
// Access is guarded by mu so round-robin cursors and dispatch counts stay
// consistent under concurrent dispatch.
type comboSelector struct {
	mu     sync.Mutex
	cursor int
	counts []int
}

// orderedSteps returns the steps to try, in priority order, for the combo's
// strategy. The returned slice always contains every step so fallback to the
// remaining steps is preserved. selectIdx, when >= 0, is the index in the
// original steps slice whose usage count should be incremented once dispatched
// (used by least_used); -1 means no count tracking applies.
func (s *comboSelector) orderedSteps(strategy string, steps []ComboStep, req *providers.ChatRequest) ([]ComboStep, int) {
	switch strategy {
	case store.ComboStrategyRoundRobin:
		return s.roundRobinOrder(steps)
	case store.ComboStrategyLeastUsed:
		return s.leastUsedOrder(steps)
	case store.ComboStrategyAuto:
		return autoOrder(steps, req), -1
	default:
		return steps, -1
	}
}

func (s *comboSelector) roundRobinOrder(steps []ComboStep) ([]ComboStep, int) {
	s.mu.Lock()
	start := s.cursor % len(steps)
	s.cursor++
	s.mu.Unlock()
	return rotate(steps, start), -1
}

func (s *comboSelector) leastUsedOrder(steps []ComboStep) ([]ComboStep, int) {
	s.mu.Lock()
	if len(s.counts) != len(steps) {
		s.counts = make([]int, len(steps))
	}
	order := make([]int, len(steps))
	for i := range order {
		order[i] = i
	}
	counts := s.counts
	sort.SliceStable(order, func(a, b int) bool {
		return counts[order[a]] < counts[order[b]]
	})
	chosen := order[0]
	s.counts[chosen]++
	s.mu.Unlock()

	ordered := make([]ComboStep, len(steps))
	for i, idx := range order {
		ordered[i] = steps[idx]
	}
	return ordered, chosen
}

// rotate returns steps starting at start, wrapping around so every step
// remains present as a fallback.
func rotate(steps []ComboStep, start int) []ComboStep {
	ordered := make([]ComboStep, 0, len(steps))
	for i := 0; i < len(steps); i++ {
		ordered = append(ordered, steps[(start+i)%len(steps)])
	}
	return ordered
}

// autoOrder places the heuristically chosen step first, then the rest in their
// original order as fallbacks.
func autoOrder(steps []ComboStep, req *providers.ChatRequest) []ComboStep {
	pick := selectAutoStepIndex(steps, req)
	ordered := make([]ComboStep, 0, len(steps))
	ordered = append(ordered, steps[pick])
	for i, step := range steps {
		if i != pick {
			ordered = append(ordered, step)
		}
	}
	return ordered
}

// selectAutoStepIndex implements the Cursor-style heuristic. Step order is
// treated as strongest->cheapest (index 0 = most capable). A request carrying
// tools or a large amount of context routes to the most capable (first) step;
// otherwise it routes to the cheapest/fastest (last) step.
func selectAutoStepIndex(steps []ComboStep, req *providers.ChatRequest) int {
	if len(steps) == 0 {
		return 0
	}
	if req != nil && (len(req.Tools) > 0 || requestCharLen(req) > autoHeavyContextThreshold) {
		return 0
	}
	return len(steps) - 1
}

// requestCharLen estimates total characters across all message contents.
func requestCharLen(req *providers.ChatRequest) int {
	total := 0
	for _, msg := range req.Messages {
		switch content := msg.Content.(type) {
		case string:
			total += len(content)
		default:
			total += len(fmt.Sprintf("%v", content))
		}
	}
	return total
}
