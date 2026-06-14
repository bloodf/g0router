package kiro

import (
	"encoding/binary"
	"hash/crc32"
	"reflect"
	"testing"
)

// buildFrame assembles a single real AWS eventstream binary frame from a set of
// string headers and a payload. It computes the genuine prelude CRC and message
// CRC (CRC-32/IEEE, big-endian) exactly as the AWS wire format requires, so the
// decoder under test is exercised against authentic wire bytes (hermetic golden
// fixtures — no network).
//
// Frame layout (all multi-byte integers big-endian):
//
//	[0:4]   total length (whole frame incl. both CRCs)
//	[4:8]   headers length
//	[8:12]  prelude CRC (CRC32 of bytes [0:8])
//	[12:..] headers
//	[..:..] payload
//	[last4] message CRC (CRC32 of bytes [0 : total-4])
//
// Each header is: 1-byte name length, name, 1-byte value type, and for the
// string type (7) a 2-byte big-endian value length followed by the value bytes.
func buildFrame(headers [][2]string, payload []byte) []byte {
	var hdr []byte
	for _, h := range headers {
		name := []byte(h[0])
		val := []byte(h[1])
		hdr = append(hdr, byte(len(name)))
		hdr = append(hdr, name...)
		hdr = append(hdr, 7) // string type
		var vl [2]byte
		binary.BigEndian.PutUint16(vl[:], uint16(len(val)))
		hdr = append(hdr, vl[:]...)
		hdr = append(hdr, val...)
	}

	headersLen := len(hdr)
	total := 12 + headersLen + len(payload) + 4

	frame := make([]byte, total)
	binary.BigEndian.PutUint32(frame[0:4], uint32(total))
	binary.BigEndian.PutUint32(frame[4:8], uint32(headersLen))
	binary.BigEndian.PutUint32(frame[8:12], crc32.ChecksumIEEE(frame[0:8]))
	copy(frame[12:], hdr)
	copy(frame[12+headersLen:], payload)
	binary.BigEndian.PutUint32(frame[total-4:], crc32.ChecksumIEEE(frame[0:total-4]))
	return frame
}

// TestDecodeEventStreamAssistantResponse verifies a single assistantResponseEvent
// frame is decoded into a parsed event map carrying _eventType + payload fields,
// in the shape resolveKiroEvent (kiro_openai_response.go) expects.
func TestDecodeEventStreamAssistantResponse(t *testing.T) {
	frame := buildFrame(
		[][2]string{
			{":message-type", "event"},
			{":event-type", "assistantResponseEvent"},
			{":content-type", "application/json"},
		},
		[]byte(`{"content":"Hello"}`),
	)

	events, err := DecodeEventStream(frame)
	if err != nil {
		t.Fatalf("DecodeEventStream error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	want := map[string]any{
		"_eventType": "assistantResponseEvent",
		"content":    "Hello",
	}
	if !reflect.DeepEqual(events[0], want) {
		t.Fatalf("event = %#v, want %#v", events[0], want)
	}
}

// TestDecodeEventStreamMultipleFrames verifies several frames concatenated in one
// buffer are each decoded in order.
func TestDecodeEventStreamMultipleFrames(t *testing.T) {
	var buf []byte
	buf = append(buf, buildFrame(
		[][2]string{{":event-type", "assistantResponseEvent"}},
		[]byte(`{"content":"foo"}`),
	)...)
	buf = append(buf, buildFrame(
		[][2]string{{":event-type", "assistantResponseEvent"}},
		[]byte(`{"content":"bar"}`),
	)...)
	buf = append(buf, buildFrame(
		[][2]string{{":event-type", "messageStopEvent"}},
		[]byte(`{}`),
	)...)

	events, err := DecodeEventStream(buf)
	if err != nil {
		t.Fatalf("DecodeEventStream error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0]["content"] != "foo" || events[1]["content"] != "bar" {
		t.Errorf("frame contents = %v / %v, want foo / bar", events[0]["content"], events[1]["content"])
	}
	if events[2]["_eventType"] != "messageStopEvent" {
		t.Errorf("frame 3 _eventType = %v, want messageStopEvent", events[2]["_eventType"])
	}
}

// TestDecodeEventStreamToolUseEvent verifies a toolUseEvent payload (with nested
// object input) round-trips through the JSON decode unchanged.
func TestDecodeEventStreamToolUseEvent(t *testing.T) {
	frame := buildFrame(
		[][2]string{{":event-type", "toolUseEvent"}},
		[]byte(`{"toolUseId":"tu_1","name":"search","input":{"q":"go"}}`),
	)
	events, err := DecodeEventStream(frame)
	if err != nil {
		t.Fatalf("DecodeEventStream error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	ev := events[0]
	if ev["_eventType"] != "toolUseEvent" || ev["toolUseId"] != "tu_1" || ev["name"] != "search" {
		t.Errorf("toolUseEvent fields = %#v", ev)
	}
	input, ok := ev["input"].(map[string]any)
	if !ok || input["q"] != "go" {
		t.Errorf("input = %#v, want map with q=go", ev["input"])
	}
}

// TestDecodeEventStreamRejectsBadPreludeCRC verifies a corrupted prelude CRC is
// rejected (the wire format guards integrity with this checksum).
func TestDecodeEventStreamRejectsBadPreludeCRC(t *testing.T) {
	frame := buildFrame(
		[][2]string{{":event-type", "assistantResponseEvent"}},
		[]byte(`{"content":"x"}`),
	)
	frame[8] ^= 0xFF // corrupt prelude CRC
	if _, err := DecodeEventStream(frame); err == nil {
		t.Fatal("expected error for corrupted prelude CRC, got nil")
	}
}

// TestDecodeEventStreamRejectsBadMessageCRC verifies a corrupted message CRC is
// rejected.
func TestDecodeEventStreamRejectsBadMessageCRC(t *testing.T) {
	frame := buildFrame(
		[][2]string{{":event-type", "assistantResponseEvent"}},
		[]byte(`{"content":"x"}`),
	)
	frame[len(frame)-1] ^= 0xFF // corrupt message CRC
	if _, err := DecodeEventStream(frame); err == nil {
		t.Fatal("expected error for corrupted message CRC, got nil")
	}
}

// TestDecodeEventStreamIgnoresPartialTrailingFrame verifies an incomplete frame
// at the end of the buffer is not emitted (matches the ref's break-on-short).
func TestDecodeEventStreamIgnoresPartialTrailingFrame(t *testing.T) {
	full := buildFrame(
		[][2]string{{":event-type", "assistantResponseEvent"}},
		[]byte(`{"content":"done"}`),
	)
	buf := append([]byte{}, full...)
	// Append a truncated next frame (only the first 8 bytes of a prelude).
	buf = append(buf, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x10)

	events, err := DecodeEventStream(buf)
	if err != nil {
		t.Fatalf("DecodeEventStream error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (partial trailing frame ignored)", len(events))
	}
	if events[0]["content"] != "done" {
		t.Errorf("content = %v, want done", events[0]["content"])
	}
}
