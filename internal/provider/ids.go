package provider

import "strings"

func CanonicalProviderID(provider string) string {
	switch normalizedProviderID(provider) {
	case "codex":
		return "openai"
	case "github":
		return "github-copilot"
	default:
		return normalizedProviderID(provider)
	}
}

func ProviderAliases(provider string) []string {
	canonical := CanonicalProviderID(provider)
	switch canonical {
	case "openai":
		return []string{"openai", "codex"}
	case "github-copilot":
		return []string{"github-copilot", "github"}
	default:
		return []string{canonical}
	}
}

func normalizedProviderID(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
