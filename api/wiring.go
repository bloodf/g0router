package api

import (
	"github.com/bloodf/g0router"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/metrics"
	"github.com/bloodf/g0router/internal/notify"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/bloodf/g0router/internal/traffic"
	"github.com/valyala/fasthttp"
)

func NewServer(config ServerConfig) *Server {
	uiFS, err := g0router.UI()
	srv := &Server{
		config:                    config,
		uiFS:                      uiFS,
		uiErr:                     err,
		limiter:                   ratelimit.NewLimiter(),
		loginRateLimiter:          auth.NewLoginRateLimiter(),
		metrics:                   metrics.NewCollector(),
		trafficBroker:             traffic.NewBroker(256),
		logRetentionInterval:      logRetentionInterval,
		connectionRefreshInterval: connectionRefreshInterval,
		tunnelHealthInterval:      tunnelHealthInterval,
		proxyPoolHealthInterval:   proxyPoolHealthInterval,
		notifiedStale:             make(map[string]bool),
		stopCh:                    make(chan struct{}),
	}
	srv.runRetention = srv.runLogRetentionOnce
	srv.runConnectionRefresh = srv.runConnectionRefreshOnce
	srv.notifierFor = func(url string) notify.Notifier { return notify.NewNotifier(url, nil) }
	if refresher, ok := config.InferenceEngine.(ConnectionRefresher); ok {
		srv.connRefresher = refresher
	}
	srv.server = &fasthttp.Server{
		Handler: srv.handle,
	}
	return srv
}
