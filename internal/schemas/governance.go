package schemas

// VirtualKey defines a virtual API key with provider routing rules.
type VirtualKey struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	TeamID          string           `json:"team_id,omitempty"`
	ProviderConfigs []ProviderConfig `json:"provider_configs"`
	Budget          *Budget          `json:"budget,omitempty"`
	RateLimitRPM    *int             `json:"rate_limit_rpm,omitempty"`
	RateLimit       *RateLimit       `json:"rate_limit,omitempty"`
}

// RateLimit is the dual-dimension token/request rate limit attached to a virtual
// key (bf-gov-3, D1/D3). Each dimension has its own max and reset period; a zero
// max disables that dimension. The reset period accepts the calendar words
// daily/weekly/monthly or a rolling-duration token (1h/1d/1M).
type RateLimit struct {
	TokenMax           int64  `json:"token_max,omitempty"`
	TokenResetPeriod   string `json:"token_reset_period,omitempty"`
	RequestMax         int64  `json:"request_max,omitempty"`
	RequestResetPeriod string `json:"request_reset_period,omitempty"`
}

// ProviderConfig binds a provider to allowed models, keys, and weight.
type ProviderConfig struct {
	Provider          string    `json:"provider"`
	AllowedModels     []string  `json:"allowed_models"`
	BlacklistedModels BlackList `json:"blacklisted_models,omitempty"`
	KeyIDs            []string  `json:"key_ids"`
	AllowAllKeys      bool      `json:"allow_all_keys"`
	Weight            *float64  `json:"weight,omitempty"`
}

// Budget tracks spend against a limit.
type Budget struct {
	Limit  float64 `json:"limit"`
	Period string  `json:"period"`
	Used   float64 `json:"used"`
}
