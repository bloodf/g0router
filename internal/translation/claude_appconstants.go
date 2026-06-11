package translation

const claudeToolSuffix = "_ide"

// ccDefaultTools is the set of Claude Code native tool names (PAR-TRANS-022).
// Client tools matching these names are skipped from renaming.
var ccDefaultTools = map[string]struct{}{
	"Task":           {},
	"TaskOutput":     {},
	"TaskStop":       {},
	"TaskCreate":     {},
	"TaskGet":        {},
	"TaskUpdate":     {},
	"TaskList":       {},
	"Bash":           {},
	"Glob":           {},
	"Grep":           {},
	"Read":           {},
	"Edit":           {},
	"Write":          {},
	"NotebookEdit":   {},
	"WebFetch":       {},
	"WebSearch":      {},
	"AskUserQuestion": {},
	"Skill":          {},
	"EnterPlanMode":  {},
	"ExitPlanMode":   {},
}

type decoyTool struct {
	name        string
	description string
	inputSchema map[string]any
}

// ccDecoyTools are Claude Code native tool names injected as unavailable
// decoys after client tools (PAR-TRANS-022).
var ccDecoyTools = []decoyTool{
	{"Task", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"TaskOutput", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"TaskStop", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"TaskCreate", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"TaskGet", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"TaskUpdate", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"TaskList", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Bash", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Glob", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Grep", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Read", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Edit", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Write", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"NotebookEdit", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"WebFetch", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"WebSearch", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"AskUserQuestion", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"Skill", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"EnterPlanMode", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
	{"ExitPlanMode", "This tool is currently unavailable.", map[string]any{"type": "object", "properties": map[string]any{}}},
}
