// Package cursor implements the Cursor IDE provider adapter
// (w7-prov-special-b, PAR-PROV-023). Cursor speaks the connect-protocol over
// HTTP with hand-rolled protobuf message framing
// (`application/connect+proto`, /aiserver.v1.ChatService/StreamUnifiedChatWithTools
// on api2.cursor.sh). This package supplies the genuinely-new wire-format work:
// a faithful Go port of the hand-rolled protobuf request encoder + response
// decoder (ref open-sse/utils/cursorProtobuf.js), the connect-RPC frame
// envelope, and the deterministic auth/checksum headers
// (ref open-sse/utils/cursorChecksum.js). The cursorToOpenAIResponse converter
// (registry.go:174) is a passthrough, so this adapter emits OpenAI-shaped chunks
// directly and runs them through the registry unchanged.
package cursor

import (
	"encoding/binary"

	"github.com/google/uuid"
)

// protobuf wire types.
const (
	wireVarint = 0
	wireLen    = 2
)

// FIELD numbers, ported verbatim from cursorProtobuf.js FIELD (only the subset
// this adapter encodes/decodes is included).
const (
	// StreamUnifiedChatRequestWithTools (top level)
	fieldRequest = 1

	// StreamUnifiedChatRequest
	fieldMessages        = 1
	fieldUnknown2        = 2
	fieldInstruction     = 3
	fieldUnknown4        = 4
	fieldModel           = 5
	fieldWebTool         = 8
	fieldUnknown13       = 13
	fieldCursorSetting   = 15
	fieldUnknown19       = 19
	fieldConversationID  = 23
	fieldMetadata        = 26
	fieldIsAgentic       = 27
	fieldSupportedTools  = 29
	fieldMessageIDs      = 30
	fieldMCPTools        = 34
	fieldLargeContext    = 35
	fieldUnknown38       = 38
	fieldUnifiedMode     = 46
	fieldUnknown47       = 47
	fieldShouldDisable   = 48
	fieldThinkingLevel   = 49
	fieldUnknown51       = 51
	fieldUnknown53       = 53
	fieldUnifiedModeName = 54

	// ConversationMessage
	fieldMsgContent  = 1
	fieldMsgRole     = 2
	fieldMsgID       = 13
	fieldMsgIsAgent  = 29
	fieldMsgUnified  = 47
	fieldMsgSupTools = 51

	// Model
	fieldModelName  = 1
	fieldModelEmpty = 4

	// Instruction
	fieldInstructionText = 1

	// CursorSetting
	fieldSettingPath     = 1
	fieldSettingUnknown3 = 3
	fieldSettingUnknown6 = 6
	fieldSettingUnknown8 = 8
	fieldSettingUnknown9 = 9
	fieldSetting6Field1  = 1
	fieldSetting6Field2  = 2

	// Metadata
	fieldMetaPlatform  = 1
	fieldMetaArch      = 2
	fieldMetaVersion   = 3
	fieldMetaCWD       = 4
	fieldMetaTimestamp = 5

	// MessageId
	fieldMsgIDID   = 1
	fieldMsgIDRole = 3

	// MCPTool
	fieldMCPToolName   = 1
	fieldMCPToolDesc   = 2
	fieldMCPToolParams = 3
	fieldMCPToolServer = 4

	// StreamUnifiedChatResponseWithTools (response)
	fieldToolCall = 1
	fieldResponse = 2

	// ClientSideToolV2Call (response tool call)
	fieldToolID        = 3
	fieldToolName      = 9
	fieldToolRawArgs   = 10
	fieldToolIsLast    = 11
	fieldToolMCPParams = 27

	// MCPParams / nested tool
	fieldMCPToolsList   = 1
	fieldMCPNestedName  = 1
	fieldMCPNestedParam = 3

	// StreamUnifiedChatResponse
	fieldResponseText = 1
	fieldThinking     = 25
	fieldThinkingText = 1
)

// role enum (cursorProtobuf.js ROLE).
const (
	roleUser      = 1
	roleAssistant = 2
)

// unified mode enum (cursorProtobuf.js UNIFIED_MODE).
const (
	unifiedModeChat  = 1
	unifiedModeAgent = 2
)

// thinking level enum (cursorProtobuf.js THINKING_LEVEL).
const (
	thinkingUnspecified = 0
	thinkingMedium      = 1
	thinkingHigh        = 2
)

// encodeVarint encodes a uint64 as a protobuf base-128 varint
// (cursorProtobuf.js encodeVarint).
func encodeVarint(v uint64) []byte {
	var out []byte
	for v >= 0x80 {
		out = append(out, byte(v&0x7f)|0x80)
		v >>= 7
	}
	out = append(out, byte(v))
	return out
}

// encodeFieldVarint encodes a VARINT field (tag + varint value).
func encodeFieldVarint(fieldNum int, value uint64) []byte {
	tag := uint64(fieldNum<<3) | wireVarint
	return append(encodeVarint(tag), encodeVarint(value)...)
}

// encodeFieldLen encodes a length-delimited field (tag + length + bytes).
func encodeFieldLen(fieldNum int, data []byte) []byte {
	tag := uint64(fieldNum<<3) | wireLen
	out := encodeVarint(tag)
	out = append(out, encodeVarint(uint64(len(data)))...)
	out = append(out, data...)
	return out
}

// encodeFieldStr is encodeFieldLen for a string value.
func encodeFieldStr(fieldNum int, s string) []byte {
	return encodeFieldLen(fieldNum, []byte(s))
}

// decodeVarint decodes a varint starting at offset, returning the value and the
// new offset (cursorProtobuf.js decodeVarint).
func decodeVarint(buf []byte, offset int) (uint64, int) {
	var result uint64
	var shift uint
	pos := offset
	for pos < len(buf) {
		b := buf[pos]
		result |= uint64(b&0x7f) << shift
		pos++
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, pos
}

// decodedField is one field parsed from a protobuf message.
type decodedField struct {
	wireType int
	value    []byte // LEN payload bytes; for VARINT, varintValue holds the number
	varint   uint64
}

// decodeMessage parses a protobuf message into a field-number -> occurrences map
// (cursorProtobuf.js decodeMessage). Only VARINT and LEN wire types are needed
// for the cursor response schema.
func decodeMessage(data []byte) map[int][]decodedField {
	fields := make(map[int][]decodedField)
	pos := 0
	for pos < len(data) {
		tag, p1 := decodeVarint(data, pos)
		if p1 == pos {
			break
		}
		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x07)
		pos = p1

		switch wireType {
		case wireVarint:
			v, p2 := decodeVarint(data, pos)
			fields[fieldNum] = append(fields[fieldNum], decodedField{wireType: wireType, varint: v})
			pos = p2
		case wireLen:
			length, p2 := decodeVarint(data, pos)
			end := p2 + int(length)
			if end > len(data) || end < p2 {
				return fields
			}
			fields[fieldNum] = append(fields[fieldNum], decodedField{wireType: wireType, value: data[p2:end]})
			pos = end
		case 1: // fixed64
			if pos+8 > len(data) {
				return fields
			}
			pos += 8
		case 5: // fixed32
			if pos+4 > len(data) {
				return fields
			}
			pos += 4
		default:
			return fields
		}
	}
	return fields
}

// firstLen returns the first LEN-typed value for a field, or nil.
func firstLen(fields map[int][]decodedField, fieldNum int) ([]byte, bool) {
	occ, ok := fields[fieldNum]
	if !ok || len(occ) == 0 {
		return nil, false
	}
	return occ[0].value, true
}

// firstVarint returns the first VARINT value for a field, or (0,false).
func firstVarint(fields map[int][]decodedField, fieldNum int) (uint64, bool) {
	occ, ok := fields[fieldNum]
	if !ok || len(occ) == 0 {
		return 0, false
	}
	return occ[0].varint, true
}

// be32 reads a big-endian uint32.
func be32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }

// newUUID returns a random UUID v4 string (cursorProtobuf.js uuidv4()).
func newUUID() string { return uuid.NewString() }
