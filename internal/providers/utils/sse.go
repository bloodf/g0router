package utils

import (
	"bufio"
	"bytes"
	"io"
)

// SSEScanner reads Server-Sent Event lines from an io.Reader.
type SSEScanner struct {
	reader *bufio.Reader
}

// NewSSEScanner creates an SSE scanner from a response body reader.
func NewSSEScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{reader: bufio.NewReader(r)}
}

// Scan returns the next data line (without the "data: " prefix).
// It skips empty lines and non-data events.
// Returns io.EOF when the stream ends.
func (s *SSEScanner) Scan() (string, error) {
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return s.parseLine(line), nil
			}
			return "", err
		}
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))
		if len(line) == 0 {
			continue
		}
		data := s.parseLine(line)
		if data != "" {
			return data, nil
		}
	}
}

func (s *SSEScanner) parseLine(line []byte) string {
	const prefix = "data: "
	if bytes.HasPrefix(line, []byte(prefix)) {
		return string(bytes.TrimPrefix(line, []byte(prefix)))
	}
	return ""
}
