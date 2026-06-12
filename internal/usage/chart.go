package usage

import (
	"encoding/json"
	"fmt"
	"time"
)

// Chart returns bucketed usage data for the given period.
// Supports today, 24h, 7d, 30d, 60d.
func (s *StatsService) Chart(period string) ([]Bucket, error) {
	switch period {
	case "today", "24h", "7d", "30d", "60d":
		// ok
	default:
		return nil, fmt.Errorf("invalid period %q", period)
	}

	now := s.clock().UTC()

	if period == "today" {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return s.hourlyBuckets(start, start.Add(24*time.Hour), now, false), nil
	}

	if period == "24h" {
		start := now.Add(-24 * time.Hour)
		return s.hourlyBuckets(start, now, now, true), nil
	}

	bucketCount := map[string]int{"7d": 7, "30d": 30, "60d": 60}[period]
	return s.dailyBuckets(now, bucketCount), nil
}

func (s *StatsService) hourlyBuckets(start, end, now time.Time, clamp bool) []Bucket {
	bucketCount := 24
	buckets := make([]Bucket, bucketCount)
	for i := 0; i < bucketCount; i++ {
		buckets[i].Label = start.Add(time.Duration(i) * time.Hour).Format("15:04")
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
		if ts.Before(start) || ts.Equal(end) || ts.After(end) {
			if !clamp {
				continue
			}
			if ts.Before(start) || ts.After(now) {
				continue
			}
		}
		idx := int(ts.Sub(start).Hours())
		if clamp {
			if idx < 0 {
				idx = 0
			}
			if idx >= bucketCount {
				idx = bucketCount - 1
			}
		}
		if idx < 0 || idx >= bucketCount {
			continue
		}
		b := &buckets[idx]
		b.Tokens += r.PromptTokens + r.CompletionTokens
		b.Cost += r.Cost
	}
	return buckets
}

func (s *StatsService) dailyBuckets(now time.Time, bucketCount int) []Bucket {
	buckets := make([]Bucket, bucketCount)
	dayMap := make(map[string]map[string]any)

	dayRows, err := s.reader.LoadDailyRange(bucketCount, now)
	if err == nil {
		for _, r := range dayRows {
			var day map[string]any
			if err := json.Unmarshal([]byte(r.Data), &day); err == nil {
				dayMap[r.DateKey] = day
			}
		}
	}

	for i := 0; i < bucketCount; i++ {
		d := now.AddDate(0, 0, -(bucketCount - 1 - i))
		dateKey := d.Format("2006-01-02")
		day := dayMap[dateKey]
		tokens := int64(0)
		cost := 0.0
		if day != nil {
			tokens = int64(toFloat64(day["promptTokens"])) + int64(toFloat64(day["completionTokens"]))
			cost = toFloat64(day["cost"])
		}
		buckets[i] = Bucket{
			Label: d.Format("Jan 2"),
			Tokens: tokens,
			Cost:  cost,
		}
	}
	return buckets
}
