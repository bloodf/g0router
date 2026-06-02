package utils

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ParseSSE(r io.Reader, fn func(data string) error) error {
	scanner := bufio.NewScanner(r)
	var dataLines []string

	dispatch := func() (bool, error) {
		if dataLines == nil {
			return false, nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = nil
		if data == "[DONE]" {
			return true, nil
		}
		if err := fn(data); err != nil {
			return false, fmt.Errorf("parse sse callback: %w", err)
		}
		return false, nil
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			done, err := dispatch()
			if done || err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimPrefix(data, " ")
		dataLines = append(dataLines, data)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("parse sse: %w", err)
	}

	_, err := dispatch()
	return err
}
