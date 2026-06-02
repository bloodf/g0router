package rtk

import (
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/rtk/filters"
)

func CompressRequest(req providers.ChatRequest) providers.ChatRequest {
	messages := make([]providers.Message, len(req.Messages))
	copy(messages, req.Messages)
	req.Messages = messages

	for i := range req.Messages {
		content, ok := req.Messages[i].Content.(string)
		if !ok {
			continue
		}
		req.Messages[i].Content = compressContent(content)
	}

	return req
}

func compressContent(content string) string {
	switch DetectFormat(content) {
	case FormatGitDiff:
		return filters.SmartTruncate(filters.GitDiff(content))
	case FormatGitStatus:
		return filters.SmartTruncate(filters.GitStatus(content))
	case FormatGrep:
		return filters.SmartTruncate(filters.Grep(content))
	case FormatFind:
		return filters.SmartTruncate(filters.Find(content))
	case FormatLS:
		return filters.SmartTruncate(filters.LS(content))
	case FormatTree:
		return filters.SmartTruncate(filters.Tree(content))
	case FormatBuildOutput:
		return filters.SmartTruncate(filters.BuildOutput(content))
	case FormatLog:
		return filters.SmartTruncate(filters.DedupLog(content))
	case FormatReadNumbered:
		return filters.SmartTruncate(filters.ReadNumbered(content))
	case FormatSearchList:
		return filters.SmartTruncate(filters.SearchList(content))
	default:
		return filters.SmartTruncate(content)
	}
}
