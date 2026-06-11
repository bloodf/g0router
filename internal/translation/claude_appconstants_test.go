package translation

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCCDefaultToolsCount(t *testing.T) {
	if len(ccDefaultTools) != 20 {
		t.Errorf("ccDefaultTools count = %d, want 20", len(ccDefaultTools))
	}
}

func TestCCDecoyToolsShape(t *testing.T) {
	if len(ccDecoyTools) != 20 {
		t.Fatalf("ccDecoyTools count = %d, want 20", len(ccDecoyTools))
	}
	for _, dt := range ccDecoyTools {
		if dt.description != "This tool is currently unavailable." {
			t.Errorf("decoy %q description = %q, want unavailable", dt.name, dt.description)
		}
		if dt.inputSchema["type"] != "object" {
			t.Errorf("decoy %q schema type = %v, want object", dt.name, dt.inputSchema["type"])
		}
		props, ok := dt.inputSchema["properties"].(map[string]any)
		if !ok || len(props) != 0 {
			t.Errorf("decoy %q schema properties = %v, want empty", dt.name, dt.inputSchema["properties"])
		}
	}
}

func TestClaudeThinkingSignaturePinned(t *testing.T) {
	wantLen := 1068
	wantSHA256 := "4286c50f0ffa43a05f50f43790a4ae30ab18f1bbc6a362df317d0d3d0144d0b3"

	if len(defaultThinkingClaudeSignature) != wantLen {
		t.Errorf("signature len = %d, want %d", len(defaultThinkingClaudeSignature), wantLen)
	}
	h := sha256.Sum256([]byte(defaultThinkingClaudeSignature))
	got := hex.EncodeToString(h[:])
	if got != wantSHA256 {
		t.Errorf("signature sha256 = %s, want %s", got, wantSHA256)
	}
}
