package usage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// UsageReader supplies the store-level reads used by StatsService.
type UsageReader interface {
	LoadDailyRange(maxDays int) ([]*store.UsageDailyRow, error)
	RangeRequestLogs(sinceISO, untilISO string) ([]*store.RequestLogEntry, error)
	ListRecentRequestLogs(limit int) ([]*store.RequestLogEntry, error)
}

// NameSource resolves display names for connection, provider, and API key IDs.
type NameSource interface {
	ConnectionName(id string) string
	ProviderName(id string) string
	APIKeyName(key string) string
}

// Stats is the usage stats payload.
type Stats struct {
	TotalRequests         int64                     `json:"total_requests"`
	TotalPromptTokens     int64                     `json:"total_prompt_tokens"`
	TotalCompletionTokens int64                     `json:"total_completion_tokens"`
	TotalCost             float64                   `json:"total_cost"`
	ByProvider            map[string]*ProviderStat  `json:"by_provider"`
	ByModel               map[string]*ModelStat     `json:"by_model"`
	ByAccount             map[string]*AccountStat   `json:"by_account"`
	ByAPIKey              map[string]*APIKeyStat    `json:"by_api_key"`
	ByEndpoint            map[string]*EndpointStat  `json:"by_endpoint"`
	Last10Minutes         []Bucket                  `json:"last_10_minutes"`
	Pending               map[string]int64          `json:"pending"`
	ActiveRequests        []ActiveRequest           `json:"active_requests"`
	RecentRequests        []RecentRequest           `json:"recent_requests"`
	ErrorProvider         string                    `json:"error_provider"`
}

// ProviderStat aggregates usage for a single provider.
type ProviderStat struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
}

// ModelStat aggregates usage for a single model/provider tuple.
type ModelStat struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
	RawModel         string  `json:"raw_model"`
	Provider         string  `json:"provider"`
	LastUsed         string  `json:"last_used"`
}

// AccountStat aggregates usage for a single account/model/provider tuple.
type AccountStat struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
	RawModel         string  `json:"raw_model"`
	Provider         string  `json:"provider"`
	ConnectionID     string  `json:"connection_id"`
	AccountName      string  `json:"account_name"`
	LastUsed         string  `json:"last_used"`
}

// APIKeyStat aggregates usage for a single API key/model/provider tuple.
type APIKeyStat struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
	RawModel         string  `json:"raw_model"`
	Provider         string  `json:"provider"`
	APIKey           string  `json:"api_key"`
	KeyName          string  `json:"key_name"`
	APIKeyKey        string  `json:"api_key_key"`
	LastUsed         string  `json:"last_used"`
}

// EndpointStat aggregates usage for a single endpoint/model/provider tuple.
type EndpointStat struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
	Endpoint         string  `json:"endpoint"`
	RawModel         string  `json:"raw_model"`
	Provider         string  `json:"provider"`
	LastUsed         string  `json:"last_used"`
}

// Bucket is a one-minute or hourly usage bucket.
type Bucket struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
	Label            string  `json:"label,omitempty"`
	Tokens           int64   `json:"tokens,omitempty"`
}

// ActiveRequest is a currently in-flight request group.
type ActiveRequest struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Count    int64  `json:"count"`
}

// StatsService builds usage statistics from daily rollups and live logs.
type StatsService struct {
	reader   UsageReader
	names    NameSource
	tracker  *Tracker
	ring     *Ring
	clock    func() time.Time
}

// NewStatsService creates a StatsService with injected dependencies.
func NewStatsService(reader UsageReader, names NameSource, tracker *Tracker, ring *Ring, clock func() time.Time) *StatsService {
	return &StatsService{
		reader:  reader,
		names:   names,
		tracker: tracker,
		ring:    ring,
		clock:   clock,
	}
}

// Stats returns usage statistics for the given period.
// Supported periods: today, 24h, 7d, 30d, 60d, all.
func (s *StatsService) Stats(period string) (Stats, error) {
	switch period {
	case "today", "24h", "7d", "30d", "60d", "all":
		// ok
	default:
		return Stats{}, fmt.Errorf("invalid period %q", period)
	}

	stats := Stats{
		ByProvider:    make(map[string]*ProviderStat),
		ByModel:       make(map[string]*ModelStat),
		ByAccount:     make(map[string]*AccountStat),
		ByAPIKey:      make(map[string]*APIKeyStat),
		ByEndpoint:    make(map[string]*EndpointStat),
		Pending:       make(map[string]int64),
		ActiveRequests: make([]ActiveRequest, 0),
		RecentRequests: make([]RecentRequest, 0),
	}

	stats.Last10Minutes = s.last10MinuteBuckets()

	now := s.clock().UTC()
	if period == "today" || period == "24h" {
		var cutoff time.Time
		if period == "today" {
			cutoff = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		} else {
			cutoff = now.Add(-24 * time.Hour)
		}
		logs, err := s.reader.RangeRequestLogs(cutoff.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			return Stats{}, fmt.Errorf("range request logs: %w", err)
		}
		s.aggregateLive(&stats, logs)
	} else {
		periodDays := map[string]int{"7d": 7, "30d": 30, "60d": 60}
		maxDays := periodDays[period]
		dayRows, err := s.reader.LoadDailyRange(maxDays)
		if err != nil {
			return Stats{}, fmt.Errorf("load daily range: %w", err)
		}
		s.aggregateDaily(&stats, dayRows)

		var cutoff time.Time
		if maxDays > 0 {
			cutoff = now.AddDate(0, 0, -maxDays)
		} else {
			cutoff = time.Time{}
		}
		overlayRows, err := s.reader.RangeRequestLogs(cutoff.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			return Stats{}, fmt.Errorf("overlay request logs: %w", err)
		}
		s.overlayLastUsed(&stats, overlayRows)
	}

	stats.TotalRequests = sumProviderRequests(stats.ByProvider)

	s.fillTrackerFields(&stats)
	s.fillRecentRequests(&stats)

	return stats, nil
}

func (s *StatsService) aggregateDaily(stats *Stats, rows []*store.UsageDailyRow) {
	for _, row := range rows {
		var day map[string]any
		if err := json.Unmarshal([]byte(row.Data), &day); err != nil {
			continue
		}

		stats.TotalPromptTokens += int64(toFloat64(day["promptTokens"]))
		stats.TotalCompletionTokens += int64(toFloat64(day["completionTokens"]))
		stats.TotalCost += toFloat64(day["cost"])

		for prov, v := range day["byProvider"].(map[string]any) {
			addCounterToProvider(stats.ByProvider, prov, v.(map[string]any))
		}

		for _, v := range day["byModel"].(map[string]any) {
			m := v.(map[string]any)
			rawModel := stringValue(m, "rawModel")
			provider := stringValue(m, "provider")
			statsKey := modelStatsKey(rawModel, provider)
			lastUsed := row.DateKey
			s.addOrUpdateModel(stats.ByModel, statsKey, m, rawModel, provider, lastUsed)
		}

		for connID, v := range day["byAccount"].(map[string]any) {
			a := v.(map[string]any)
			rawModel := stringValue(a, "rawModel")
			provider := stringValue(a, "provider")
			accountName := s.names.ConnectionName(connID)
			if accountName == "" {
				accountName = accountFallback(connID)
			}
			statsKey := accountStatsKey(rawModel, provider, accountName)
			lastUsed := row.DateKey
			s.addOrUpdateAccount(stats.ByAccount, statsKey, a, rawModel, provider, connID, accountName, lastUsed)
		}

		for key, v := range day["byApiKey"].(map[string]any) {
			ak := v.(map[string]any)
			rawModel := stringValue(ak, "rawModel")
			provider := stringValue(ak, "provider")
			apiKeyVal := stringValue(ak, "apiKey")
			keyName := s.names.APIKeyName(apiKeyVal)
			if keyName == "" {
				keyName = apiKeyNameFallback(apiKeyVal)
			}
			apiKeyKey := apiKeyVal
			if apiKeyKey == "" {
				apiKeyKey = "local-no-key"
			}
			lastUsed := row.DateKey
			s.addOrUpdateAPIKey(stats.ByAPIKey, key, ak, rawModel, provider, apiKeyVal, keyName, apiKeyKey, lastUsed)
		}

		for key, v := range day["byEndpoint"].(map[string]any) {
			ep := v.(map[string]any)
			endpoint := stringValue(ep, "endpoint")
			rawModel := stringValue(ep, "rawModel")
			provider := stringValue(ep, "provider")
			lastUsed := row.DateKey
			s.addOrUpdateEndpoint(stats.ByEndpoint, key, ep, endpoint, rawModel, provider, lastUsed)
		}
	}
}

func (s *StatsService) aggregateLive(stats *Stats, rows []*store.RequestLogEntry) {
	for _, r := range rows {
		stats.TotalPromptTokens += r.PromptTokens
		stats.TotalCompletionTokens += r.CompletionTokens
		stats.TotalCost += r.Cost

		provider := r.Provider
		providerDisplay := s.names.ProviderName(provider)

		addToProvider(stats.ByProvider, provider)

		modelKey := modelStatsKey(r.Model, provider)
		s.addOrUpdateModel(stats.ByModel, modelKey, nil, r.Model, providerDisplay, r.Timestamp)

		if r.ConnectionID != "" {
			accountName := s.names.ConnectionName(r.ConnectionID)
			if accountName == "" {
				accountName = accountFallback(r.ConnectionID)
			}
			accountKey := accountStatsKey(r.Model, provider, accountName)
			s.addOrUpdateAccount(stats.ByAccount, accountKey, nil, r.Model, providerDisplay, r.ConnectionID, accountName, r.Timestamp)
		}

		apiKeyVal := r.APIKey
		if apiKeyVal != "" {
			keyName := s.names.APIKeyName(apiKeyVal)
			if keyName == "" {
				keyName = apiKeyNameFallback(apiKeyVal)
			}
			akKey := fmt.Sprintf("%s|%s|%s", apiKeyVal, r.Model, providerOrUnknown(provider))
			s.addOrUpdateAPIKey(stats.ByAPIKey, akKey, nil, r.Model, providerDisplay, apiKeyVal, keyName, apiKeyVal, r.Timestamp)
		} else {
			keyName := "Local (No API Key)"
			s.addOrUpdateAPIKey(stats.ByAPIKey, "local-no-key", nil, r.Model, providerDisplay, "", keyName, "local-no-key", r.Timestamp)
		}

		endpoint := r.Endpoint
		if endpoint == "" {
			endpoint = "Unknown"
		}
		epKey := fmt.Sprintf("%s|%s|%s", endpoint, r.Model, providerOrUnknown(provider))
		s.addOrUpdateEndpoint(stats.ByEndpoint, epKey, nil, endpoint, r.Model, providerDisplay, r.Timestamp)
	}
}

func (s *StatsService) overlayLastUsed(stats *Stats, rows []*store.RequestLogEntry) {
	for _, r := range rows {
		ts := r.Timestamp
		modelKey := modelStatsKey(r.Model, r.Provider)
		if m, ok := stats.ByModel[modelKey]; ok && ts > m.LastUsed {
			m.LastUsed = ts
		}

		if r.ConnectionID != "" {
			accountName := s.names.ConnectionName(r.ConnectionID)
			if accountName == "" {
				accountName = accountFallback(r.ConnectionID)
			}
			accountKey := accountStatsKey(r.Model, r.Provider, accountName)
			if a, ok := stats.ByAccount[accountKey]; ok && ts > a.LastUsed {
				a.LastUsed = ts
			}
		}

		apiKeyKey := apiKeyStatsKey(r.APIKey, r.Model, r.Provider)
		if k, ok := stats.ByAPIKey[apiKeyKey]; ok && ts > k.LastUsed {
			k.LastUsed = ts
		}

		endpoint := r.Endpoint
		if endpoint == "" {
			endpoint = "Unknown"
		}
		epKey := fmt.Sprintf("%s|%s|%s", endpoint, r.Model, providerOrUnknown(r.Provider))
		if e, ok := stats.ByEndpoint[epKey]; ok && ts > e.LastUsed {
			e.LastUsed = ts
		}
	}
}

func (s *StatsService) last10MinuteBuckets() []Bucket {
	now := s.clock().UTC()
	currentMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.UTC)
	start := currentMinute.Add(-9 * time.Minute)

	buckets := make([]Bucket, 10)
	for i := 0; i < 10; i++ {
		buckets[i].Label = start.Add(time.Duration(i) * time.Minute).Format("15:04")
	}

	rows, err := s.reader.RangeRequestLogs(start.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return buckets
	}

	for _, r := range rows {
		ts, err := time.Parse(time.RFC3339, r.Timestamp)
		if err != nil {
			continue
		}
		minute := time.Date(ts.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), 0, 0, time.UTC)
		idx := int(minute.Sub(start) / time.Minute)
		if idx < 0 || idx >= 10 {
			continue
		}
		b := &buckets[idx]
		b.Requests++
		b.PromptTokens += r.PromptTokens
		b.CompletionTokens += r.CompletionTokens
		b.Cost += r.Cost
	}
	return buckets
}

func (s *StatsService) fillTrackerFields(stats *Stats) {
	if s.tracker == nil {
		return
	}
	stats.ErrorProvider = s.tracker.LastErrorProvider()
}

func (s *StatsService) fillRecentRequests(stats *Stats) {
	if s.ring == nil {
		return
	}
	entries := s.ring.Snapshot()
	recent := make([]RecentRequest, 0, len(entries))
	for _, e := range entries {
		recent = append(recent, RecentRequest{
			Timestamp:        e.Timestamp,
			Model:            e.Model,
			Provider:         e.Provider,
			PromptTokens:     e.PromptTokens,
			CompletionTokens: e.CompletionTokens,
			Status:           e.Status,
		})
	}
	stats.RecentRequests = DedupeRecent(recent)
}

func addCounterToProvider(target map[string]*ProviderStat, key string, src map[string]any) {
	p := target[key]
	if p == nil {
		p = &ProviderStat{}
		target[key] = p
	}
	p.Requests += int64(toFloat64(src["requests"]))
	p.PromptTokens += int64(toFloat64(src["promptTokens"]))
	p.CompletionTokens += int64(toFloat64(src["completionTokens"]))
	p.Cost += toFloat64(src["cost"])
}

func addToProvider(target map[string]*ProviderStat, provider string) {
	p := target[provider]
	if p == nil {
		p = &ProviderStat{}
		target[provider] = p
	}
	p.Requests++
}

func (s *StatsService) addOrUpdateModel(target map[string]*ModelStat, key string, src map[string]any, rawModel, provider, lastUsed string) {
	m := target[key]
	if m == nil {
		m = &ModelStat{RawModel: rawModel, Provider: s.names.ProviderName(provider), LastUsed: lastUsed}
		target[key] = m
	}
	if src != nil {
		m.Requests += int64(toFloat64(src["requests"]))
		m.PromptTokens += int64(toFloat64(src["promptTokens"]))
		m.CompletionTokens += int64(toFloat64(src["completionTokens"]))
		m.Cost += toFloat64(src["cost"])
	} else {
		m.Requests++
	}
	if lastUsed > m.LastUsed {
		m.LastUsed = lastUsed
	}
}

func (s *StatsService) addOrUpdateAccount(target map[string]*AccountStat, key string, src map[string]any, rawModel, provider, connectionID, accountName, lastUsed string) {
	a := target[key]
	if a == nil {
		a = &AccountStat{
			RawModel:     rawModel,
			Provider:     s.names.ProviderName(provider),
			ConnectionID: connectionID,
			AccountName:  accountName,
			LastUsed:     lastUsed,
		}
		target[key] = a
	}
	if src != nil {
		a.Requests += int64(toFloat64(src["requests"]))
		a.PromptTokens += int64(toFloat64(src["promptTokens"]))
		a.CompletionTokens += int64(toFloat64(src["completionTokens"]))
		a.Cost += toFloat64(src["cost"])
	} else {
		a.Requests++
	}
	if lastUsed > a.LastUsed {
		a.LastUsed = lastUsed
	}
}

func (s *StatsService) addOrUpdateAPIKey(target map[string]*APIKeyStat, key string, src map[string]any, rawModel, provider, apiKey, keyName, apiKeyKey, lastUsed string) {
	k := target[key]
	if k == nil {
		k = &APIKeyStat{
			RawModel:  rawModel,
			Provider:  s.names.ProviderName(provider),
			APIKey:    apiKey,
			KeyName:   keyName,
			APIKeyKey: apiKeyKey,
			LastUsed:  lastUsed,
		}
		target[key] = k
	}
	if src != nil {
		k.Requests += int64(toFloat64(src["requests"]))
		k.PromptTokens += int64(toFloat64(src["promptTokens"]))
		k.CompletionTokens += int64(toFloat64(src["completionTokens"]))
		k.Cost += toFloat64(src["cost"])
	} else {
		k.Requests++
	}
	if lastUsed > k.LastUsed {
		k.LastUsed = lastUsed
	}
}

func (s *StatsService) addOrUpdateEndpoint(target map[string]*EndpointStat, key string, src map[string]any, endpoint, rawModel, provider, lastUsed string) {
	e := target[key]
	if e == nil {
		e = &EndpointStat{
			Endpoint: endpoint,
			RawModel: rawModel,
			Provider: s.names.ProviderName(provider),
			LastUsed: lastUsed,
		}
		target[key] = e
	}
	if src != nil {
		e.Requests += int64(toFloat64(src["requests"]))
		e.PromptTokens += int64(toFloat64(src["promptTokens"]))
		e.CompletionTokens += int64(toFloat64(src["completionTokens"]))
		e.Cost += toFloat64(src["cost"])
	} else {
		e.Requests++
	}
	if lastUsed > e.LastUsed {
		e.LastUsed = lastUsed
	}
}

func sumProviderRequests(byProvider map[string]*ProviderStat) int64 {
	var sum int64
	for _, p := range byProvider {
		sum += p.Requests
	}
	return sum
}

func modelStatsKey(rawModel, provider string) string {
	if provider != "" {
		return fmt.Sprintf("%s (%s)", rawModel, provider)
	}
	return rawModel
}

func accountStatsKey(rawModel, provider, accountName string) string {
	return fmt.Sprintf("%s (%s - %s)", rawModel, provider, accountName)
}

func apiKeyStatsKey(apiKey, model, provider string) string {
	if apiKey == "" {
		return "local-no-key"
	}
	return fmt.Sprintf("%s|%s|%s", apiKey, model, providerOrUnknown(provider))
}

func accountFallback(connID string) string {
	if len(connID) > 8 {
		return fmt.Sprintf("Account %s...", connID[:8])
	}
	return fmt.Sprintf("Account %s...", connID)
}

func apiKeyNameFallback(apiKey string) string {
	if apiKey == "" {
		return "Local (No API Key)"
	}
	if len(apiKey) > 8 {
		return apiKey[:8] + "..."
	}
	return apiKey + "..."
}

func providerOrUnknown(p string) string {
	if p == "" {
		return "unknown"
	}
	return p
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	}
	return 0
}
