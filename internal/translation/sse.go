package translation

import (
	"encoding/json"
	"fmt"
)

// FormatSSE formats an event map for Server-Sent Events. For Claude-format
// events carrying a "type" key, it prefixes the frame with "event: <type>".
// All other formats emit the standard "data: <json>" frame.
func FormatSSE(format Format, event map[string]any) []byte {
	b, err := json.Marshal(event)
	if err != nil {
		return []byte("data: null\n\n")
	}
	if format == FormatClaude && event != nil {
		if _, ok := event["type"]; ok {
			return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event["type"], b))
		}
	}
	return []byte(fmt.Sprintf("data: %s\n\n", b))
}
