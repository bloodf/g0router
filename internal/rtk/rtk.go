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
		if req.Messages[i].Role == "tool" {
			content, ok := req.Messages[i].Content.(string)
			if ok {
				req.Messages[i].Content = compressContent(content)
				continue
			}
		}
		req.Messages[i].Content = compressToolResultBlocks(req.Messages[i].Content)
	}

	return req
}

func compressToolResultBlocks(content any) any {
	switch blocks := content.(type) {
	case []map[string]any:
		return compressAnyBlocks(blocks)
	case []map[string]string:
		return compressStringBlocks(blocks)
	case []any:
		return compressMixedBlocks(blocks)
	default:
		return content
	}
}

func compressAnyBlocks(blocks []map[string]any) []map[string]any {
	compressed := make([]map[string]any, len(blocks))
	for i, block := range blocks {
		compressed[i] = copyAnyBlock(block)
		if compressed[i]["type"] != "tool_result" {
			continue
		}
		content, ok := compressed[i]["content"].(string)
		if ok {
			compressed[i]["content"] = compressContent(content)
		}
	}
	return compressed
}

func compressStringBlocks(blocks []map[string]string) []map[string]string {
	compressed := make([]map[string]string, len(blocks))
	for i, block := range blocks {
		compressed[i] = copyStringBlock(block)
		if compressed[i]["type"] == "tool_result" {
			compressed[i]["content"] = compressContent(compressed[i]["content"])
		}
	}
	return compressed
}

func compressMixedBlocks(blocks []any) []any {
	compressed := make([]any, len(blocks))
	copy(compressed, blocks)
	for i, block := range blocks {
		typedBlock, ok := block.(map[string]any)
		if !ok || typedBlock["type"] != "tool_result" {
			continue
		}
		next := copyAnyBlock(typedBlock)
		content, ok := next["content"].(string)
		if ok {
			next["content"] = compressContent(content)
		}
		compressed[i] = next
	}
	return compressed
}

func copyAnyBlock(block map[string]any) map[string]any {
	copied := make(map[string]any, len(block))
	for key, value := range block {
		copied[key] = value
	}
	return copied
}

func copyStringBlock(block map[string]string) map[string]string {
	copied := make(map[string]string, len(block))
	for key, value := range block {
		copied[key] = value
	}
	return copied
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
