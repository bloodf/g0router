package usage

import (
	"fmt"
	"time"
)

// RecentLogs returns the last N request log rows formatted as display strings.
func (s *StatsService) RecentLogs(limit int) []string {
	rows, err := s.reader.ListRecentRequestLogs(limit)
	if err != nil {
		return []string{}
	}

	out := make([]string, 0, len(rows))
	for _, r := range rows {
		ts := "-"
		if t, err := time.Parse(time.RFC3339, r.Timestamp); err == nil {
			ts = t.Format("02-01-2006 15:04:05")
		}

		model := orDash(r.Model)
		provider := orDashUpper(r.Provider)

		account := "-"
		if r.ConnectionID != "" {
			account = s.names.ConnectionName(r.ConnectionID)
			if account == "" {
				if len(r.ConnectionID) > 8 {
					account = r.ConnectionID[:8]
				} else {
					account = r.ConnectionID
				}
			}
		}

		sent := tokenString(r.PromptTokens, r.Tokens, "prompt_tokens")
		received := tokenString(r.CompletionTokens, r.Tokens, "completion_tokens")
		status := orDash(r.Status)

		out = append(out, fmt.Sprintf("%s | %s | %s | %s | %s | %s | %s", ts, model, provider, account, sent, received, status))
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func orDashUpper(s string) string {
	if s == "" {
		return "-"
	}
	return upper(s)
}

func upper(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c = c - 'a' + 'A'
		}
		out[i] = c
	}
	return string(out)
}

func tokenString(col int64, tokens map[string]int64, key string) string {
	if col != 0 {
		return fmt.Sprintf("%d", col)
	}
	if tokens != nil {
		if v, ok := tokens[key]; ok && v != 0 {
			return fmt.Sprintf("%d", v)
		}
	}
	return "-"
}
