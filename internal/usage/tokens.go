package usage

// TokenSet is the normalized token breakdown used for cost calculation.
type TokenSet struct {
	Prompt        int64
	Completion    int64
	Cached        int64
	Reasoning     int64
	CacheCreation int64
}

// NormalizeTokens converts a token map from an API response into a TokenSet.
// It accepts both canonical and synonym field names:
//   - prompt_tokens / input_tokens
//   - completion_tokens / output_tokens
//   - cached_tokens / cache_read_input_tokens
//   - reasoning_tokens
//   - cache_creation_input_tokens
func NormalizeTokens(tokens map[string]int64) TokenSet {
	if tokens == nil {
		return TokenSet{}
	}

	get := func(keys ...string) int64 {
		for _, k := range keys {
			if val, ok := tokens[k]; ok {
				return val
			}
		}
		return 0
	}

	return TokenSet{
		Prompt:        get("prompt_tokens", "input_tokens"),
		Completion:    get("completion_tokens", "output_tokens"),
		Cached:        get("cached_tokens", "cache_read_input_tokens"),
		Reasoning:     get("reasoning_tokens"),
		CacheCreation: get("cache_creation_input_tokens"),
	}
}
