package translation

import "strings"

// Format identifies a wire-format dialect.
type Format string

// Format identifiers (PAR-TRANS-002), verbatim from 9router's formats.js.
const (
	FormatOpenAI          Format = "openai"
	FormatOpenAIResponses Format = "openai-responses"
	FormatOpenAIResponse  Format = "openai-response"
	FormatClaude          Format = "claude"
	FormatGemini          Format = "gemini"
	FormatGeminiCLI       Format = "gemini-cli"
	FormatVertex          Format = "vertex"
	FormatCodex           Format = "codex"
	FormatAntigravity     Format = "antigravity"
	FormatKiro            Format = "kiro"
	FormatCursor          Format = "cursor"
	FormatOllama          Format = "ollama"
	FormatCommandCode     Format = "commandcode"
)

// DetectFormatByEndpoint returns the source format for a request URL pathname
// plus body hint, mirroring 9router's detectFormatByEndpoint. It returns the
// empty Format when no endpoint-specific format is known.
func DetectFormatByEndpoint(path string, hasInputArray bool) Format {
	if strings.Contains(path, "/v1/responses") {
		return FormatOpenAIResponses
	}
	if strings.Contains(path, "/v1/messages") {
		return FormatClaude
	}
	if strings.Contains(path, "/v1/chat/completions") && hasInputArray {
		return FormatOpenAI
	}
	return ""
}
