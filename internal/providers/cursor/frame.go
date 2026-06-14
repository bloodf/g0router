package cursor

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
)

// connect-RPC frame compression flags (cursor.js COMPRESS_FLAG).
const (
	compressNone = 0x00
	compressGzip = 0x01
)

// wrapConnectFrame builds a connect-RPC frame: 1-byte flags + 4-byte big-endian
// payload length + payload (cursorProtobuf.js wrapConnectRPCFrame). When compress
// is true the payload is gzip-compressed and the gzip flag is set.
func wrapConnectFrame(payload []byte, compress bool) []byte {
	flags := byte(compressNone)
	final := payload
	if compress {
		final = gzipBytes(payload)
		flags = compressGzip
	}
	frame := make([]byte, 5+len(final))
	frame[0] = flags
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(final)))
	copy(frame[5:], final)
	return frame
}

// parseConnectFrame parses one connect-RPC frame from the front of buf. It
// returns the flags, the (decompressed) payload, the number of bytes consumed,
// and ok=false when the buffer does not hold a complete frame
// (cursorProtobuf.js parseConnectRPCFrame; cursor.js frame loop).
func parseConnectFrame(buf []byte) (flags byte, payload []byte, consumed int, ok bool) {
	if len(buf) < 5 {
		return 0, nil, 0, false
	}
	flags = buf[0]
	length := int(binary.BigEndian.Uint32(buf[1:5]))
	if len(buf) < 5+length {
		return 0, nil, 0, false
	}
	payload = buf[5 : 5+length]
	if flags == compressGzip {
		if dec, err := gunzipBytes(payload); err == nil {
			payload = dec
		}
	}
	return flags, payload, 5 + length, true
}

// gzipBytes gzip-compresses b.
func gzipBytes(b []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

// gunzipBytes gzip-decompresses b.
func gunzipBytes(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
