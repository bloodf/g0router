package translation

import (
	"fmt"
	"strings"
)

const defaultThinkingGeminiCLISignature = "CiQBjz1rX/AlslZWMe5RgBt4Tv9j4+YNZTTez+JH2/+5oAlICygKXgGPPWtf7/Sux9eLYap/bmYAdPqFThLXj+l7o0DLu/hdgU98MA9ZrlRDNHXx+T0tuY8AcnjPZbiDyOq2bE11Fjhsk6p5axqayaapC/Pt9GczcgIQf1z15WTxCeKWAPYKYQGPPWtfDYj0nlNFNoTlU39RC91Z16xFKJ2MLEmkm+NvimsoOJ6be3g2BssNPtJ/9BKDXRA5cVs17tBeeW72lH8TMB5999udtxHM2SiUsnWsrHlfVuGSCpNQQ+5REw8HNvEKkgEBjz1rXzBNWrqZGbjun55K+vgYPBhJO2qZ67uRWXUA5/qcU12U/mbi5XoA3swoxYE8LEXfZvFFC9WG/W28QNCA0Qd4Trk/WkWiAwZmB8a84Fs14rkv3wqyxwFavPkJorqurAfd2XzGiFy0sB0ITCOPYi1HzDGV5WfXk6b9k+jT66/RuzGa8EcSOWo/QtC3Bkhgowo4AY89a1/f/tw8A02zjIoK7JVDAbf8W4UfmbApJJhwXIiGtu1M0JItObx7g2reYqT+HHL2Q/R4VDc="

func openaiToGeminiCLIRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	gemini, err := openaiToGeminiBase(model, body, stream, defaultThinkingGeminiCLISignature, credentials)
	if err != nil {
		return nil, fmt.Errorf("openaiToGeminiCLIRequest: %w", err)
	}

	if genConfig, ok := gemini["generationConfig"].(map[string]any); ok {
		// Add thinking config for CLI from reasoning_effort
		if rawEffort, ok := body["reasoning_effort"]; ok && rawEffort != nil {
			if effort, ok := rawEffort.(string); ok && effort != "" {
				budgetMap := map[string]int{
					"low":    1024,
					"medium": 8192,
					"high":   32768,
				}
				budget := budgetMap[strings.ToLower(effort)]
				if budget == 0 {
					budget = 8192
				}
				// "include_thoughts" is snake_case in the frozen ref
				// (openai-to-gemini.js:239,247) — verbatim parity, not a typo.
				genConfig["thinkingConfig"] = map[string]any{
					"thinkingBudget":   float64(budget),
					"include_thoughts": true,
				}
			}
		}

		// Thinking config from Claude format
		if rawThinking, ok := body["thinking"]; ok && rawThinking != nil {
			if thinking, ok := rawThinking.(map[string]any); ok {
				if t, ok := thinking["type"].(string); ok && t == "enabled" {
					if budget, ok := thinking["budget_tokens"]; ok && budget != nil {
						// snake_case per frozen ref openai-to-gemini.js:247.
						genConfig["thinkingConfig"] = map[string]any{
							"thinkingBudget":   budget,
							"include_thoughts": true,
						}
					}
				}
			}
		}
	}

	// Clean schema for tools
	if tools, ok := gemini["tools"].([]any); ok && len(tools) > 0 {
		if tool0, ok := tools[0].(map[string]any); ok {
			if fdList, ok := tool0["functionDeclarations"].([]any); ok {
				for _, fd := range fdList {
					fdm, ok := fd.(map[string]any)
					if !ok {
						continue
					}
					if params, ok := fdm["parameters"].(map[string]any); ok {
						cleanJSONSchemaForGemini(params)
					}
				}
			}
		}
	}

	return gemini, nil
}
