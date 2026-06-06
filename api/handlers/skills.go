package handlers

import "github.com/valyala/fasthttp"

type skillItem struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// Skills returns the embedded static skills catalog.
func Skills(ctx *fasthttp.RequestCtx) {
	catalog := []skillItem{
		{Name: "9router", Category: "gateway", Description: "AI gateway with OpenAI-compatible REST", URL: "https://github.com/bloodf/g0router"},
		{Name: "agent-browser", Category: "automation", Description: "Browser automation for AI agents", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-chat", Category: "chat", Description: "Chat and code generation via gateway", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-image", Category: "image", Description: "Image generation via gateway", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-stt", Category: "audio", Description: "Speech-to-text via gateway", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-tts", Category: "audio", Description: "Text-to-speech via gateway", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-web-search", Category: "search", Description: "Web search via gateway", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-web-fetch", Category: "search", Description: "Fetch URL content via gateway", URL: "https://github.com/bloodf/g0router"},
		{Name: "9router-embeddings", Category: "embeddings", Description: "Vector embeddings via gateway", URL: "https://github.com/bloodf/g0router"},
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[skillItem]{Data: catalog})
}
