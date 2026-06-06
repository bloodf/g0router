package proxy

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// telemetryWindow is the lookback duration used when fetching provider/model
// stats for the "fastest" and "cheapest" strategies.
const telemetryWindow = 24 * time.Hour

// autoHeavyContextThreshold is the approximate character count across all
// request messages above which the "auto" strategy treats a request as heavy
// context and routes it to the most capable (first) step.
const autoHeavyContextThreshold = 8000

// Task classifications produced by classifyRequestTask, ordered by descending
// capability need. The "heavy" tasks (vision, tools, code, large) route to the
// most capable step; "simple" routes to the cheapest.
const (
	taskVision = "vision"
	taskTools  = "tools"
	taskCode   = "code"
	taskLarge  = "large"
	taskSimple = "simple"
)

// codeSignalThreshold is the number of distinct code signals required before a
// fence-free message is classified as code. Kept conservative to avoid
// misclassifying ordinary prose that happens to contain a single brace or
// semicolon.
const codeSignalThreshold = 3

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
	return s.orderedStepsWithStats(strategy, steps, req, nil)
}

// orderedStepsWithStats is the same as orderedSteps but accepts pre-fetched
// telemetry stats so the caller can pass a single query result for all steps.
// A nil stats map causes fastest/cheapest to fall back to the stored order.
func (s *comboSelector) orderedStepsWithStats(strategy string, steps []ComboStep, req *providers.ChatRequest, stats map[string]store.ModelStat) ([]ComboStep, int) {
	switch strategy {
	case store.ComboStrategyRoundRobin:
		return s.roundRobinOrder(steps)
	case store.ComboStrategyLeastUsed:
		return s.leastUsedOrder(steps)
	case store.ComboStrategyAuto:
		return autoOrder(steps, req), -1
	case store.ComboStrategyFastest:
		return telemetryOrder(steps, stats, func(st store.ModelStat) float64 { return st.AvgLatencyMS }), -1
	case store.ComboStrategyCheapest:
		return telemetryOrder(steps, stats, func(st store.ModelStat) float64 { return st.AvgCostUSD }), -1
	default:
		return steps, -1
	}
}

// telemetryOrder sorts steps ascending by the metric extracted by scoreFn.
// Steps with no telemetry entry sort last. Ties preserve original order
// (sort.SliceStable). The full step list is always returned so remaining
// steps act as fallbacks.
//
// Telemetry key: "provider/model". A ComboStep carries Provider (a
// providers.ModelProvider) and Model; we join them as-is. When the step's
// provider is empty the key falls back to "/model", which will simply miss in
// the map and sort the step last — the same safe behaviour as unknown.
func telemetryOrder(steps []ComboStep, stats map[string]store.ModelStat, scoreFn func(store.ModelStat) float64) []ComboStep {
	ordered := make([]ComboStep, len(steps))
	copy(ordered, steps)

	sort.SliceStable(ordered, func(i, j int) bool {
		si, iOK := stats[telemetryKey(ordered[i])]
		sj, jOK := stats[telemetryKey(ordered[j])]
		if !iOK && !jOK {
			return false // both unknown, preserve order
		}
		if !iOK {
			return false // i unknown, j known → j first
		}
		if !jOK {
			return true // i known, j unknown → i first
		}
		return scoreFn(si) < scoreFn(sj)
	})
	return ordered
}

// telemetryKey builds the "provider/model" map key for a combo step.
// It uses the string value of the ModelProvider directly (e.g. "groq",
// "openai") which matches how LogRequest stores the provider column.
func telemetryKey(step ComboStep) string {
	return string(step.Provider) + "/" + step.Model
}

// fetchTelemetryStats fetches provider/model stats from the store for the
// standard telemetryWindow. On error it returns a nil map and logs nothing —
// callers fall back to stored step order.
type telemetryStore interface {
	ProviderModelStats(time.Time) (map[string]store.ModelStat, error)
}

func fetchTelemetryStats(s telemetryStore) map[string]store.ModelStat {
	if s == nil {
		return nil
	}
	stats, err := s.ProviderModelStats(time.Now().Add(-telemetryWindow))
	if err != nil {
		return nil
	}
	return stats
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

// selectAutoStepIndex implements the Cursor-style per-task heuristic. Step order
// is treated as strongest->cheapest (index 0 = most capable, last = cheapest).
// classifyRequestTask categorizes the request; any high-capability task (vision,
// tools, code, large context) routes to the most capable (first) step, while a
// simple/short chat routes to the cheapest/fastest (last) step. Deterministic
// and side-effect free.
func selectAutoStepIndex(steps []ComboStep, req *providers.ChatRequest) int {
	if len(steps) == 0 {
		return 0
	}
	if classifyRequestTask(req) == taskSimple {
		return len(steps) - 1
	}
	return 0
}

// classifyRequestTask categorizes a request into one of the task constants so
// the selection (and later logging) can reason about why a step was chosen.
// Detection is ordered by descending capability need: vision, tools, code, then
// large context, falling back to simple. Pure and deterministic.
func classifyRequestTask(req *providers.ChatRequest) string {
	if req == nil {
		return taskSimple
	}
	if requestHasVision(req) {
		return taskVision
	}
	if len(req.Tools) > 0 {
		return taskTools
	}
	text := requestText(req)
	if textLooksLikeCode(text) {
		return taskCode
	}
	if len(text) > autoHeavyContextThreshold {
		return taskLarge
	}
	return taskSimple
}

// requestHasVision reports whether any message carries image content. Structured
// multimodal messages (Content is a slice of parts) are inspected for an
// "image_url" or "image" part type. Because Message.Content is an untyped `any`
// that frequently arrives as a plain string, string content is also scanned for
// an "image_url" marker or an embedded image data URL. Limitation: a string that
// merely mentions these tokens without real image data is treated as vision.
func requestHasVision(req *providers.ChatRequest) bool {
	for _, msg := range req.Messages {
		switch content := msg.Content.(type) {
		case string:
			if stringHasImageMarker(content) {
				return true
			}
		case []any:
			if partsHaveImage(content) {
				return true
			}
		default:
			if stringHasImageMarker(fmt.Sprintf("%v", content)) {
				return true
			}
		}
	}
	return false
}

func partsHaveImage(parts []any) bool {
	for _, part := range parts {
		m, ok := part.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := m["type"].(string); t == "image_url" || t == "image" {
			return true
		}
		if _, ok := m["image_url"]; ok {
			return true
		}
	}
	return false
}

func stringHasImageMarker(s string) bool {
	return strings.Contains(s, "image_url") || strings.Contains(s, "data:image/")
}

// textLooksLikeCode reports whether text is likely code. A fenced block (```) is
// a strong positive. Otherwise it requires at least codeSignalThreshold distinct
// code signals, keeping a single brace or semicolon in prose from triggering.
func textLooksLikeCode(text string) bool {
	if strings.Contains(text, "```") {
		return true
	}
	signals := []string{"func ", "class ", "import ", "def ", "=>", ";\n", "{", "}"}
	hits := 0
	for _, sig := range signals {
		if strings.Contains(text, sig) {
			hits++
		}
	}
	return hits >= codeSignalThreshold
}

// requestText concatenates all message text content with newlines so character
// counts and code signals span the whole conversation.
func requestText(req *providers.ChatRequest) string {
	var b strings.Builder
	for _, msg := range req.Messages {
		switch content := msg.Content.(type) {
		case string:
			b.WriteString(content)
		case []any:
			b.WriteString(partsText(content))
		default:
			fmt.Fprintf(&b, "%v", content)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// partsText extracts the "text" fields from structured multimodal content parts.
func partsText(parts []any) string {
	var b strings.Builder
	for _, part := range parts {
		m, ok := part.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := m["text"].(string); ok {
			b.WriteString(t)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
