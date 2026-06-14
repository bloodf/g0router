// Package kiro implements the Kiro (AWS CodeWhisperer) provider adapter
// (w7-prov-special-b, PAR-PROV-022). Kiro speaks the AWS CodeWhisperer streaming
// API, which returns an `application/vnd.amazon.eventstream` BINARY frame stream.
// This package supplies the genuinely-new work: the AWS eventstream frame decoder
// that turns those bytes into the parsed event maps the EXISTING translation
// converter (kiroToOpenAIResponse, registry.go:172) consumes, plus a thin HTTP
// executor that feeds the converter via the registry.
package kiro

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
)

// eventStreamHeaderTypeString is the AWS eventstream header value type for a
// UTF-8 string (the only type Kiro emits for the headers we read).
const eventStreamHeaderTypeString = 7

// preludeLen is the byte length of the frame prelude: total length (4) +
// headers length (4) + prelude CRC (4).
const preludeLen = 12

// DecodeEventStream parses an AWS eventstream byte buffer
// (application/vnd.amazon.eventstream) into the parsed event maps expected by
// kiroToOpenAIResponse. Each frame yields one map carrying the `:event-type`
// header under the `_eventType` key merged with the JSON payload's top-level
// fields.
//
// Frame layout (all multi-byte integers big-endian), per the AWS wire format
// (ref open-sse/executors/kiro.js parseEventFrame):
//
//	[0:4]   total length (whole frame incl. both CRCs)
//	[4:8]   headers length
//	[8:12]  prelude CRC (CRC32/IEEE of bytes [0:8])
//	[12:..] headers
//	[..:..] payload
//	[last4] message CRC (CRC32/IEEE of bytes [0 : total-4])
//
// A truncated trailing frame is ignored (matches the ref's break-on-short).
// Frames with empty/whitespace payloads or non-JSON payloads contribute no
// event. Corrupted prelude or message CRCs are rejected.
func DecodeEventStream(data []byte) ([]map[string]any, error) {
	var events []map[string]any
	offset := 0

	for offset+preludeLen <= len(data) {
		total := int(binary.BigEndian.Uint32(data[offset : offset+4]))

		// Bounds: a valid frame is at least the prelude + message CRC, and must
		// fit within the remaining buffer. A short tail means the next frame has
		// not fully arrived — stop (ignore the partial frame).
		if total < preludeLen+4 || offset+total > len(data) {
			break
		}

		frame := data[offset : offset+total]
		event, err := parseEventFrame(frame)
		if err != nil {
			return nil, err
		}
		if event != nil {
			events = append(events, event)
		}
		offset += total
	}

	return events, nil
}

// parseEventFrame decodes a single complete eventstream frame. It returns nil
// (no error) when the frame carries no usable event payload.
func parseEventFrame(frame []byte) (map[string]any, error) {
	total := len(frame)
	headersLen := int(binary.BigEndian.Uint32(frame[4:8]))

	if preludeLen+headersLen+4 > total {
		return nil, fmt.Errorf("kiro eventstream: headers length %d exceeds frame size %d", headersLen, total)
	}

	// Validate the prelude CRC (CRC32/IEEE of the first 8 bytes).
	wantPrelude := binary.BigEndian.Uint32(frame[8:12])
	if got := crc32.ChecksumIEEE(frame[0:8]); got != wantPrelude {
		return nil, fmt.Errorf("kiro eventstream: prelude CRC mismatch (got %08x want %08x)", got, wantPrelude)
	}

	// Validate the message CRC (CRC32/IEEE of everything before the trailing 4).
	wantMessage := binary.BigEndian.Uint32(frame[total-4:])
	if got := crc32.ChecksumIEEE(frame[0 : total-4]); got != wantMessage {
		return nil, fmt.Errorf("kiro eventstream: message CRC mismatch (got %08x want %08x)", got, wantMessage)
	}

	headers := parseHeaders(frame[preludeLen : preludeLen+headersLen])

	payloadStart := preludeLen + headersLen
	payloadEnd := total - 4
	payload := frame[payloadStart:payloadEnd]

	eventType := headers[":event-type"]
	if eventType == "" {
		// No event-type header (e.g. exception/error envelope) — nothing the
		// converter consumes.
		return nil, nil
	}

	event := map[string]any{"_eventType": eventType}

	// Merge the JSON payload's top-level fields into the event map. Empty or
	// non-JSON payloads contribute no fields (the converter tolerates the bare
	// _eventType, e.g. messageStopEvent).
	if len(trimSpace(payload)) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(payload, &parsed); err == nil {
			for k, v := range parsed {
				if k == "_eventType" {
					continue
				}
				event[k] = v
			}
		}
	}

	return event, nil
}

// parseHeaders decodes the eventstream header block into a name->value map.
// Only the string value type (7) is read; any other type stops parsing (Kiro's
// frames only use string headers for the keys we consume).
func parseHeaders(buf []byte) map[string]string {
	headers := make(map[string]string)
	offset := 0
	n := len(buf)

	for offset < n {
		nameLen := int(buf[offset])
		offset++
		if offset+nameLen > n {
			break
		}
		name := string(buf[offset : offset+nameLen])
		offset += nameLen

		if offset >= n {
			break
		}
		headerType := buf[offset]
		offset++

		if headerType != eventStreamHeaderTypeString {
			break
		}
		if offset+2 > n {
			break
		}
		valueLen := int(binary.BigEndian.Uint16(buf[offset : offset+2]))
		offset += 2
		if offset+valueLen > n {
			break
		}
		headers[name] = string(buf[offset : offset+valueLen])
		offset += valueLen
	}

	return headers
}

// trimSpace reports the payload with leading/trailing ASCII whitespace removed,
// used only to detect empty/whitespace-only payloads.
func trimSpace(b []byte) []byte {
	start := 0
	for start < len(b) && isSpace(b[start]) {
		start++
	}
	end := len(b)
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
