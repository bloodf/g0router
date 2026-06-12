package usage

// Pricing holds per-model rates in dollars per 1M tokens.
type Pricing struct {
	Input         float64
	Output        float64
	Cached        float64
	Reasoning     float64
	CacheCreation float64
}

// ModelPricing is the canonical, provider-agnostic model price table.
var ModelPricing = map[string]Pricing{
	// === Anthropic / Claude ===
	"claude-opus-4-6":              {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 25.00, CacheCreation: 6.25},
	"claude-opus-4-5-20251101":     {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 25.00, CacheCreation: 6.25},
	"claude-sonnet-4-6":            {Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 15.00, CacheCreation: 3.75},
	"claude-sonnet-4-5-20250929":   {Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 15.00, CacheCreation: 3.75},
	"claude-haiku-4-5-20251001":    {Input: 1.00, Output: 5.00, Cached: 0.10, Reasoning: 5.00, CacheCreation: 1.25},
	"claude-sonnet-4-20250514":     {Input: 3.00, Output: 15.00, Cached: 1.50, Reasoning: 15.00, CacheCreation: 3.00},
	"claude-opus-4-20250514":       {Input: 15.00, Output: 25.00, Cached: 7.50, Reasoning: 112.50, CacheCreation: 15.00},
	"claude-3-5-sonnet-20241022":   {Input: 3.00, Output: 15.00, Cached: 1.50, Reasoning: 15.00, CacheCreation: 3.00},
	"claude-haiku-4.5":             {Input: 0.50, Output: 2.50, Cached: 0.05, Reasoning: 3.75, CacheCreation: 0.50},
	"claude-opus-4.1":              {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 37.50, CacheCreation: 5.00},
	"claude-opus-4.5":              {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 37.50, CacheCreation: 5.00},
	"claude-opus-4.6":              {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 37.50, CacheCreation: 5.00},
	"claude-sonnet-4":              {Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 22.50, CacheCreation: 3.00},
	"claude-sonnet-4.5":            {Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 22.50, CacheCreation: 3.00},
	"claude-sonnet-4.6":            {Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 22.50, CacheCreation: 3.00},
	"claude-opus-4-5-thinking":     {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 37.50, CacheCreation: 5.00},
	"claude-opus-4-6-thinking":     {Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 37.50, CacheCreation: 5.00},

	// === OpenAI / GPT ===
	"gpt-3.5-turbo":                {Input: 0.50, Output: 1.50, Cached: 0.25, Reasoning: 2.25, CacheCreation: 0.50},
	"gpt-4":                        {Input: 2.50, Output: 10.00, Cached: 1.25, Reasoning: 15.00, CacheCreation: 2.50},
	"gpt-4-turbo":                  {Input: 10.00, Output: 30.00, Cached: 5.00, Reasoning: 45.00, CacheCreation: 10.00},
	"gpt-4o":                       {Input: 2.50, Output: 10.00, Cached: 1.25, Reasoning: 15.00, CacheCreation: 2.50},
	"gpt-4o-mini":                  {Input: 0.15, Output: 0.60, Cached: 0.075, Reasoning: 0.90, CacheCreation: 0.15},
	"gpt-4.1":                      {Input: 2.50, Output: 10.00, Cached: 1.25, Reasoning: 15.00, CacheCreation: 2.50},
	"gpt-5":                        {Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00},
	"gpt-5-mini":                   {Input: 0.75, Output: 3.00, Cached: 0.375, Reasoning: 4.50, CacheCreation: 0.75},
	"gpt-5-codex":                  {Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00},
	"gpt-5.1":                      {Input: 4.00, Output: 16.00, Cached: 2.00, Reasoning: 24.00, CacheCreation: 4.00},
	"gpt-5.1-codex":                {Input: 4.00, Output: 16.00, Cached: 2.00, Reasoning: 24.00, CacheCreation: 4.00},
	"gpt-5.1-codex-mini":           {Input: 1.50, Output: 6.00, Cached: 0.75, Reasoning: 9.00, CacheCreation: 1.50},
	"gpt-5.1-codex-mini-high":      {Input: 2.00, Output: 8.00, Cached: 1.00, Reasoning: 12.00, CacheCreation: 2.00},
	"gpt-5.1-codex-max":            {Input: 8.00, Output: 32.00, Cached: 4.00, Reasoning: 48.00, CacheCreation: 8.00},
	"gpt-5.2":                      {Input: 5.00, Output: 20.00, Cached: 2.50, Reasoning: 30.00, CacheCreation: 5.00},
	"gpt-5.2-codex":                {Input: 5.00, Output: 20.00, Cached: 2.50, Reasoning: 30.00, CacheCreation: 5.00},
	"gpt-5.3-codex":                {Input: 6.00, Output: 24.00, Cached: 3.00, Reasoning: 36.00, CacheCreation: 6.00},
	"gpt-5.3-codex-xhigh":          {Input: 10.00, Output: 40.00, Cached: 5.00, Reasoning: 60.00, CacheCreation: 10.00},
	"gpt-5.3-codex-high":           {Input: 8.00, Output: 32.00, Cached: 4.00, Reasoning: 48.00, CacheCreation: 8.00},
	"gpt-5.3-codex-low":            {Input: 4.00, Output: 16.00, Cached: 2.00, Reasoning: 24.00, CacheCreation: 4.00},
	"gpt-5.3-codex-none":           {Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00},
	"gpt-5.3-codex-spark":          {Input: 3.00, Output: 12.00, Cached: 0.30, Reasoning: 12.00, CacheCreation: 3.00},
	"o1":                           {Input: 15.00, Output: 60.00, Cached: 7.50, Reasoning: 90.00, CacheCreation: 15.00},
	"o1-mini":                      {Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00},

	// === Gemini ===
	"gemini-3-flash-preview":       {Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50},
	"gemini-3-pro-preview":         {Input: 2.00, Output: 12.00, Cached: 0.25, Reasoning: 18.00, CacheCreation: 2.00},
	"gemini-3.1-pro-low":           {Input: 2.00, Output: 12.00, Cached: 0.25, Reasoning: 18.00, CacheCreation: 2.00},
	"gemini-3.1-pro-high":          {Input: 4.00, Output: 18.00, Cached: 0.50, Reasoning: 27.00, CacheCreation: 4.00},
	"gemini-pro-agent":             {Input: 4.00, Output: 18.00, Cached: 0.50, Reasoning: 27.00, CacheCreation: 4.00},
	"gemini-3-flash-agent":         {Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50},
	"gemini-3.5-flash-low":         {Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50},
	"gemini-3.5-flash-extra-low":   {Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50},
	"gemini-3-flash":               {Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50},
	"gemini-2.5-pro":               {Input: 2.00, Output: 12.00, Cached: 0.25, Reasoning: 18.00, CacheCreation: 2.00},
	"gemini-2.5-flash":             {Input: 0.30, Output: 2.50, Cached: 0.03, Reasoning: 3.75, CacheCreation: 0.30},
	"gemini-2.5-flash-lite":        {Input: 0.15, Output: 1.25, Cached: 0.015, Reasoning: 1.875, CacheCreation: 0.15},

	// === Qwen ===
	"qwen3-coder-plus":             {Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00},
	"qwen3-coder-flash":            {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},

	// === Kimi ===
	"kimi-k2":                      {Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00},
	"kimi-k2-thinking":             {Input: 1.50, Output: 6.00, Cached: 0.75, Reasoning: 9.00, CacheCreation: 1.50},
	"kimi-k2.5":                    {Input: 1.20, Output: 4.80, Cached: 0.60, Reasoning: 7.20, CacheCreation: 1.20},
	"kimi-k2.5-thinking":           {Input: 1.80, Output: 7.20, Cached: 0.90, Reasoning: 10.80, CacheCreation: 1.80},
	"kimi-latest":                  {Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00},

	// === DeepSeek ===
	"deepseek-chat":                {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
	"deepseek-reasoner":            {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
	"deepseek-r1":                  {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
	"deepseek-v3.2-chat":           {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
	"deepseek-v3.2-reasoner":       {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
	"deepseek-v4-flash":            {Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14},
	"deepseek-v4-pro":              {Input: 0.435, Output: 0.87, Cached: 0.003625, Reasoning: 0.87, CacheCreation: 0.435},

	// === GLM ===
	"glm-4.6":                      {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},
	"glm-4.6v":                     {Input: 0.75, Output: 3.00, Cached: 0.375, Reasoning: 4.50, CacheCreation: 0.75},
	"glm-4.7":                      {Input: 0.75, Output: 3.00, Cached: 0.375, Reasoning: 4.50, CacheCreation: 0.75},
	"glm-5":                        {Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00},

	// === MiniMax ===
	"MiniMax-M3":                   {Input: 0.30, Output: 1.20, Cached: 0.06, Reasoning: 1.80, CacheCreation: 0.30},
	"MiniMax-M2.1":                 {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},
	"MiniMax-M2.5":                 {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},
	"MiniMax-M2.7":                 {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},
	"minimax-m2.1":                 {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},
	"minimax-m2.5":                 {Input: 0.60, Output: 2.40, Cached: 0.30, Reasoning: 3.60, CacheCreation: 0.60},

	// === Grok ===
	"grok-code-fast-1":             {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},

	// === OpenRouter fallback ===
	"auto":                         {Input: 2.00, Output: 8.00, Cached: 1.00, Reasoning: 12.00, CacheCreation: 2.00},

	// === Misc ===
	"oswe-vscode-prime":            {Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00},
	"gpt-oss-120b-medium":          {Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50},
	"vision-model":                 {Input: 1.50, Output: 6.00, Cached: 0.75, Reasoning: 9.00, CacheCreation: 1.50},
	"coder-model":                  {Input: 1.50, Output: 6.00, Cached: 0.75, Reasoning: 9.00, CacheCreation: 1.50},
}

// ProviderPricing holds provider-specific overrides that differ from ModelPricing.
var ProviderPricing = map[string]map[string]Pricing{
	"gh": {
		"gpt-5.3-codex": {Input: 1.75, Output: 14.00, Cached: 0.175, Reasoning: 14.00, CacheCreation: 1.75},
	},
}

// PatternPrice pairs a glob pattern with its pricing. First match wins.
type PatternPrice struct {
	Pattern string
	Pricing Pricing
}

// PatternPricing is the ordered pattern fallback table.
var PatternPricing = []PatternPrice{
	// --- Codex variants ---
	{Pattern: "*-codex-xhigh", Pricing: Pricing{Input: 10.00, Output: 40.00, Cached: 5.00, Reasoning: 60.00, CacheCreation: 10.00}},
	{Pattern: "*-codex-high", Pricing: Pricing{Input: 8.00, Output: 32.00, Cached: 4.00, Reasoning: 48.00, CacheCreation: 8.00}},
	{Pattern: "*-codex-max", Pricing: Pricing{Input: 8.00, Output: 32.00, Cached: 4.00, Reasoning: 48.00, CacheCreation: 8.00}},
	{Pattern: "*-codex-mini-*", Pricing: Pricing{Input: 1.50, Output: 6.00, Cached: 0.75, Reasoning: 9.00, CacheCreation: 1.50}},
	{Pattern: "*-codex-mini", Pricing: Pricing{Input: 1.50, Output: 6.00, Cached: 0.75, Reasoning: 9.00, CacheCreation: 1.50}},
	{Pattern: "*-codex-low", Pricing: Pricing{Input: 4.00, Output: 16.00, Cached: 2.00, Reasoning: 24.00, CacheCreation: 4.00}},
	{Pattern: "*-codex-none", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00}},
	{Pattern: "*-codex-spark", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 0.30, Reasoning: 12.00, CacheCreation: 3.00}},
	{Pattern: "codex-*", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00}},
	{Pattern: "*-codex", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00}},

	// --- Claude ---
	{Pattern: "claude-opus-*", Pricing: Pricing{Input: 5.00, Output: 25.00, Cached: 0.50, Reasoning: 25.00, CacheCreation: 6.25}},
	{Pattern: "claude-sonnet-*", Pricing: Pricing{Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 15.00, CacheCreation: 3.75}},
	{Pattern: "claude-haiku-*", Pricing: Pricing{Input: 1.00, Output: 5.00, Cached: 0.10, Reasoning: 5.00, CacheCreation: 1.25}},
	{Pattern: "claude-*", Pricing: Pricing{Input: 3.00, Output: 15.00, Cached: 0.30, Reasoning: 15.00, CacheCreation: 3.75}},

	// --- Gemini (specific first, generic last) ---
	{Pattern: "gemini-*-flash-lite", Pricing: Pricing{Input: 0.15, Output: 1.25, Cached: 0.015, Reasoning: 1.875, CacheCreation: 0.15}},
	{Pattern: "gemini-*-flash", Pricing: Pricing{Input: 0.30, Output: 2.50, Cached: 0.03, Reasoning: 3.75, CacheCreation: 0.30}},
	{Pattern: "gemini-*-pro", Pricing: Pricing{Input: 2.00, Output: 12.00, Cached: 0.25, Reasoning: 18.00, CacheCreation: 2.00}},
	{Pattern: "gemini-3-*", Pricing: Pricing{Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50}},
	{Pattern: "gemini-2.5-*", Pricing: Pricing{Input: 0.30, Output: 2.50, Cached: 0.03, Reasoning: 3.75, CacheCreation: 0.30}},
	{Pattern: "gemini-*", Pricing: Pricing{Input: 0.50, Output: 3.00, Cached: 0.03, Reasoning: 4.50, CacheCreation: 0.50}},

	// --- GPT (specific first, generic last) ---
	{Pattern: "gpt-5.3-*", Pricing: Pricing{Input: 6.00, Output: 24.00, Cached: 3.00, Reasoning: 36.00, CacheCreation: 6.00}},
	{Pattern: "gpt-5.2-*", Pricing: Pricing{Input: 5.00, Output: 20.00, Cached: 2.50, Reasoning: 30.00, CacheCreation: 5.00}},
	{Pattern: "gpt-5.1-*", Pricing: Pricing{Input: 4.00, Output: 16.00, Cached: 2.00, Reasoning: 24.00, CacheCreation: 4.00}},
	{Pattern: "gpt-5-*", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00}},
	{Pattern: "gpt-5*", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00}},
	{Pattern: "gpt-4o-*", Pricing: Pricing{Input: 0.15, Output: 0.60, Cached: 0.075, Reasoning: 0.90, CacheCreation: 0.15}},
	{Pattern: "gpt-4o", Pricing: Pricing{Input: 2.50, Output: 10.00, Cached: 1.25, Reasoning: 15.00, CacheCreation: 2.50}},
	{Pattern: "gpt-4*", Pricing: Pricing{Input: 2.50, Output: 10.00, Cached: 1.25, Reasoning: 15.00, CacheCreation: 2.50}},

	// --- o1 / o-series ---
	{Pattern: "o1-*", Pricing: Pricing{Input: 3.00, Output: 12.00, Cached: 1.50, Reasoning: 18.00, CacheCreation: 3.00}},
	{Pattern: "o1", Pricing: Pricing{Input: 15.00, Output: 60.00, Cached: 7.50, Reasoning: 90.00, CacheCreation: 15.00}},
	{Pattern: "o3-*", Pricing: Pricing{Input: 10.00, Output: 40.00, Cached: 5.00, Reasoning: 60.00, CacheCreation: 10.00}},
	{Pattern: "o4-*", Pricing: Pricing{Input: 2.00, Output: 8.00, Cached: 1.00, Reasoning: 12.00, CacheCreation: 2.00}},

	// --- Qwen ---
	{Pattern: "qwen3-coder-*", Pricing: Pricing{Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00}},
	{Pattern: "qwen*-coder-*", Pricing: Pricing{Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00}},
	{Pattern: "qwen*", Pricing: Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}},

	// --- Kimi ---
	{Pattern: "kimi-*-thinking", Pricing: Pricing{Input: 1.80, Output: 7.20, Cached: 0.90, Reasoning: 10.80, CacheCreation: 1.80}},
	{Pattern: "kimi-k2*", Pricing: Pricing{Input: 1.20, Output: 4.80, Cached: 0.60, Reasoning: 7.20, CacheCreation: 1.20}},
	{Pattern: "kimi-*", Pricing: Pricing{Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00}},

	// --- DeepSeek ---
	{Pattern: "deepseek-*reasoner*", Pricing: Pricing{Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14}},
	{Pattern: "deepseek-r*", Pricing: Pricing{Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14}},
	{Pattern: "deepseek-v*", Pricing: Pricing{Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14}},
	{Pattern: "deepseek-*", Pricing: Pricing{Input: 0.14, Output: 0.28, Cached: 0.0028, Reasoning: 0.28, CacheCreation: 0.14}},

	// --- GLM ---
	{Pattern: "glm-5*", Pricing: Pricing{Input: 1.00, Output: 4.00, Cached: 0.50, Reasoning: 6.00, CacheCreation: 1.00}},
	{Pattern: "glm-4*", Pricing: Pricing{Input: 0.75, Output: 3.00, Cached: 0.375, Reasoning: 4.50, CacheCreation: 0.75}},
	{Pattern: "glm-*", Pricing: Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}},

	// --- MiniMax ---
	{Pattern: "MiniMax-*", Pricing: Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}},
	{Pattern: "minimax-*", Pricing: Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}},

	// --- Grok ---
	{Pattern: "grok-code-*", Pricing: Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}},
	{Pattern: "grok-*", Pricing: Pricing{Input: 0.50, Output: 2.00, Cached: 0.25, Reasoning: 3.00, CacheCreation: 0.50}},
}
