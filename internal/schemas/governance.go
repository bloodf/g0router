package schemas

// VirtualKey defines a virtual API key with provider routing rules.
type VirtualKey struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	TeamID          string           `json:"team_id,omitempty"`
	ProviderConfigs []ProviderConfig `json:"provider_configs"`
	Budget          *Budget          `json:"budget,omitempty"`
	RateLimitRPM    *int             `json:"rate_limit_rpm,omitempty"`
}

// ProviderConfig binds a provider to allowed models, keys, and weight.
type ProviderConfig struct {
	Provider      string   `json:"provider"`
	AllowedModels []string `json:"allowed_models"`
	KeyIDs        []string `json:"key_ids"`
	AllowAllKeys  bool     `json:"allow_all_keys"`
	Weight        *float64 `json:"weight,omitempty"`
}

// Budget tracks spend against a limit.
type Budget struct {
	Limit  float64 `json:"limit"`
	Period string  `json:"period"`
	Used   float64 `json:"used"`
}
