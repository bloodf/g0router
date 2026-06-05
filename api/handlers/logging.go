package handlers

import (
	"log"

	"github.com/valyala/fasthttp"
)

func Logs(ctx *fasthttp.RequestCtx, usageStore UsageStore) {
	if usageStore == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "usage store unavailable")
		return
	}

	filter, err := parseUsageFilter(ctx)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	entries, err := usageStore.GetUsage(filter)
	if err != nil {
		log.Printf("get logs: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get logs")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, usageListResponse{
		Object: "list",
		Data:   usageLogResponses(entries),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}
