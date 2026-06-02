package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSSENormal(t *testing.T) {
	input := strings.NewReader("data: first\n\ndata: second\n\n")
	var got []string

	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}

	want := []string{"first", "second"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseSSEWithDone(t *testing.T) {
	input := strings.NewReader("data: first\n\ndata: [DONE]\n\ndata: after\n\n")
	var got []string

	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}

	if len(got) != 1 || got[0] != "first" {
		t.Fatalf("got %v, want [first]", got)
	}
}

func TestParseSSEComments(t *testing.T) {
	input := strings.NewReader(": comment\n\ndata: payload\n\n: ignored\n")
	var got []string

	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}

	if len(got) != 1 || got[0] != "payload" {
		t.Fatalf("got %v, want [payload]", got)
	}
}

func TestParseSSEEmptyLines(t *testing.T) {
	input := strings.NewReader("\n\ndata: payload\n\n\n")
	var got []string

	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}

	if len(got) != 1 || got[0] != "payload" {
		t.Fatalf("got %v, want [payload]", got)
	}
}

func TestParseSSEMultilineEvent(t *testing.T) {
	input := strings.NewReader("data: first\ndata: second\n\n")
	var got []string

	err := ParseSSE(input, func(data string) error {
		got = append(got, data)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseSSE: %v", err)
	}

	if len(got) != 1 || got[0] != "first\nsecond" {
		t.Fatalf("got %v, want multiline payload", got)
	}
}

func TestParseSSECallbackError(t *testing.T) {
	want := errors.New("stop")
	input := strings.NewReader("data: payload\n\n")

	err := ParseSSE(input, func(data string) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}
