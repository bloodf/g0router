package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
)

// SSEScanner reads Server-Sent Event lines from an io.Reader.
type SSEScanner struct {
	reader       *bufio.Reader
	ndjson       bool
	pendingEvent string
	lastEvent    string
}

// NewSSEScanner creates an SSE scanner from a response body reader.
func NewSSEScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{reader: bufio.NewReader(r)}
}

// NewNDJSONScanner creates an SSE scanner in NDJSON mode.
// In this mode, lines starting with '{' are treated as raw JSON payloads
// and all other lines are skipped.
func NewNDJSONScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{reader: bufio.NewReader(r), ndjson: true}
}

// Event returns the event name associated with the most recently
// returned payload. It is cleared on the next Scan call.
func (s *SSEScanner) Event() string {
	return s.lastEvent
}

// Scan returns the next data line (without the "data: " prefix).
// It skips empty lines and non-data events.
// Returns io.EOF when the stream ends.
func (s *SSEScanner) Scan() (string, error) {
	s.lastEvent = ""
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				data, ok := s.parseLine(line)
				if ok {
					s.lastEvent = s.pendingEvent
					s.pendingEvent = ""
					return data, nil
				}
				return "", io.EOF
			}
			return "", err
		}
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))
		if len(line) == 0 {
			continue
		}
		data, ok := s.parseLine(line)
		if ok {
			s.lastEvent = s.pendingEvent
			s.pendingEvent = ""
			return data, nil
		}
	}
}

func (s *SSEScanner) parseLine(line []byte) (string, bool) {
	if s.ndjson {
		if len(line) > 0 && line[0] == '{' {
			if json.Valid(line) {
				return string(line), true
			}
			return "", false
		}
		return "", false
	}

	const dataPrefix = "data: "
	const eventPrefix = "event: "

	if bytes.HasPrefix(line, []byte(eventPrefix)) {
		s.pendingEvent = string(bytes.TrimPrefix(line, []byte(eventPrefix)))
		return "", false
	}

	if bytes.HasPrefix(line, []byte(dataPrefix)) {
		payload := string(bytes.TrimPrefix(line, []byte(dataPrefix)))
		if payload == "[DONE]" {
			return payload, true
		}
		if len(payload) > 0 && (payload[0] == '{' || payload[0] == '[') {
			if !json.Valid([]byte(payload)) {
				n := len(payload)
				if n > 0 && n < 1000 {
					preview := payload
					if len(preview) > 100 {
						preview = preview[:100]
					}
					log.Printf("[WARN] failed to parse SSE line (%d chars): %s", n, preview)
				}
				return "", false
			}
		}
		return payload, true
	}

	return "", false
}
