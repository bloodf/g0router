package admin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// adminNameSource resolves display names for the stats service.
type adminNameSource struct {
	connCache *usage.ConnNameCache
	store     *store.Store
}

func (s *adminNameSource) ConnectionName(id string) string {
	if s.connCache == nil {
		return ""
	}
	return s.connCache.Get()[id]
}

func (s *adminNameSource) ProviderName(id string) string {
	if s.store == nil || id == "" {
		return id
	}
	p, err := s.store.GetProvider(id)
	if err != nil || p == nil {
		return id
	}
	if p.Name != "" {
		return p.Name
	}
	return id
}

func (s *adminNameSource) APIKeyName(key string) string {
	if s.store == nil || key == "" {
		return ""
	}
	rec, err := s.store.GetAPIKeyByKey(key)
	if err != nil || rec == nil {
		return ""
	}
	if rec.Name != "" {
		return rec.Name
	}
	return ""
}

// connInfoLister adapts store connections to usage.ConnInfo.
func connInfoLister(st *store.Store) func() ([]usage.ConnInfo, error) {
	return func() ([]usage.ConnInfo, error) {
		conns, err := st.ListConnections()
		if err != nil {
			return nil, fmt.Errorf("list connections: %w", err)
		}
		out := make([]usage.ConnInfo, 0, len(conns))
		for _, c := range conns {
			out = append(out, usage.ConnInfo{ID: c.ID, Name: c.Name})
		}
		return out, nil
	}
}

func realTimerFactory(d time.Duration, fn func()) func() {
	t := time.AfterFunc(d, fn)
	return func() { t.Stop() }
}

// BuildUsageServices constructs the production StatsService and pricing Resolver.
func BuildUsageServices(st *store.Store) (*usage.StatsService, *usage.Resolver) {
	resolver := usage.NewResolver(st, func() int64 { return time.Now().UnixMilli() })

	events := usage.NewEvents()
	tracker := usage.NewTracker(func() time.Time { return time.Now() }, realTimerFactory, events)
	ring := usage.NewRing(50)
	_ = ring.Init(func() ([]*store.RequestLogEntry, error) { return st.ListRecentRequestLogs(50) })

	connCache := usage.NewConnNameCache(connInfoLister(st), 30*time.Second, func() time.Time { return time.Now() })
	nameSrc := &adminNameSource{connCache: connCache, store: st}

	stats := usage.NewStatsService(st, nameSrc, tracker, ring, func() time.Time { return time.Now() })
	return stats, resolver
}

// GetUsageStats handles GET /api/usage/stats.
func (h *Handlers) GetUsageStats(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "all"
	}
	if !validUsagePeriod(period, true) {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid period")
		return
	}

	stats, err := h.stats.Stats(period)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "usage stats")
		return
	}
	writeData(ctx, fasthttp.StatusOK, stats)
}

// GetUsageChart handles GET /api/usage/chart.
func (h *Handlers) GetUsageChart(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "7d"
	}
	if !validUsagePeriod(period, false) {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid period")
		return
	}

	buckets, err := h.stats.Chart(period)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "chart data")
		return
	}
	writeData(ctx, fasthttp.StatusOK, buckets)
}

// GetUsageRequestLogs handles GET /api/usage/request-logs and /api/usage/logs.
func (h *Handlers) GetUsageRequestLogs(ctx *fasthttp.RequestCtx) {
	logs := h.stats.RecentLogs(200)
	writeData(ctx, fasthttp.StatusOK, logs)
}

// GetRequestDetails handles GET /api/usage/request-details.
func (h *Handlers) GetRequestDetails(ctx *fasthttp.RequestCtx) {
	page, pageSize, ok := parsePagination(ctx)
	if !ok {
		return
	}

	filter := store.RequestDetailsFilter{
		Page:       page,
		PageSize:   pageSize,
		Provider:   string(ctx.QueryArgs().Peek("provider")),
		Model:      string(ctx.QueryArgs().Peek("model")),
		ConnectionID: string(ctx.QueryArgs().Peek("connectionId")),
		Status:     string(ctx.QueryArgs().Peek("status")),
		StartDate:  string(ctx.QueryArgs().Peek("startDate")),
		EndDate:    string(ctx.QueryArgs().Peek("endDate")),
	}

	rows, pagination, err := h.store.QueryRequestDetails(filter)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "request details")
		return
	}

	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"data":       rows,
		"pagination": pagination,
	})
}

func parsePagination(ctx *fasthttp.RequestCtx) (int, int, bool) {
	pageStr := string(ctx.QueryArgs().Peek("page"))
	pageSizeStr := string(ctx.QueryArgs().Peek("pageSize"))

	page := 1
	if pageStr != "" {
		if n, err := strconv.Atoi(pageStr); err == nil {
			page = n
		}
	}
	pageSize := 20
	if pageSizeStr != "" {
		if n, err := strconv.Atoi(pageSizeStr); err == nil {
			pageSize = n
		}
	}

	if page < 1 {
		writeError(ctx, fasthttp.StatusBadRequest, "page must be >= 1")
		return 0, 0, false
	}
	if pageSize < 1 || pageSize > 100 {
		writeError(ctx, fasthttp.StatusBadRequest, "pageSize must be between 1 and 100")
		return 0, 0, false
	}
	return page, pageSize, true
}

func validUsagePeriod(period string, allowAll bool) bool {
	valid := []string{"today", "24h", "7d", "30d", "60d"}
	if allowAll {
		valid = append(valid, "all")
	}
	for _, v := range valid {
		if period == v {
			return true
		}
	}
	return false
}
