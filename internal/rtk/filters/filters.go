package filters

import (
	"fmt"
	"strings"
)

const smartTruncateLimit = 4096

type Filter func(input string) string

type NamedFilter struct {
	Name   string
	Filter Filter
}

func NamedFilters() []NamedFilter {
	return []NamedFilter{
		{Name: "git_diff", Filter: GitDiff},
		{Name: "git_status", Filter: GitStatus},
		{Name: "grep", Filter: Grep},
		{Name: "find", Filter: Find},
		{Name: "ls", Filter: LS},
		{Name: "tree", Filter: Tree},
		{Name: "build_output", Filter: BuildOutput},
		{Name: "dedup_log", Filter: DedupLog},
		{Name: "smart_truncate", Filter: SmartTruncate},
		{Name: "read_numbered", Filter: ReadNumbered},
		{Name: "search_list", Filter: SearchList},
	}
}

func GitDiff(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "index ") {
			continue
		}
		if strings.HasPrefix(line, " ") {
			out = append(out, " "+strings.TrimSpace(line))
			continue
		}
		out = append(out, strings.TrimRight(line, " \t"))
	}
	return strings.Join(out, "\n")
}

func GitStatus(input string) string {
	var out []string
	inUntracked := false
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.HasPrefix(trimmed, "On branch "):
			out = append(out, "branch "+strings.TrimPrefix(trimmed, "On branch "))
			inUntracked = false
		case strings.HasPrefix(trimmed, "Untracked files:"):
			inUntracked = true
		case strings.HasPrefix(trimmed, "modified:"):
			out = append(out, "M "+strings.TrimSpace(strings.TrimPrefix(trimmed, "modified:")))
			inUntracked = false
		case strings.HasPrefix(trimmed, "deleted:"):
			out = append(out, "D "+strings.TrimSpace(strings.TrimPrefix(trimmed, "deleted:")))
			inUntracked = false
		case strings.HasPrefix(trimmed, "new file:"):
			out = append(out, "A "+strings.TrimSpace(strings.TrimPrefix(trimmed, "new file:")))
			inUntracked = false
		case inUntracked && !strings.Contains(trimmed, "use \"git add"):
			out = append(out, "?? "+trimmed)
		}
	}
	return strings.Join(out, "\n")
}

func Grep(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(parts) != 3 {
			continue
		}
		out = append(out, parts[0]+":"+parts[1]+" "+strings.TrimSpace(parts[2]))
	}
	return strings.Join(out, "\n")
}

func Find(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		path := strings.TrimSpace(line)
		if path == "" || path == "." || path == "./" {
			continue
		}
		out = append(out, strings.TrimPrefix(path, "./"))
	}
	return strings.Join(out, "\n")
}

func LS(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "total ") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 9 {
			out = append(out, trimmed)
			continue
		}
		name := strings.Join(fields[8:], " ")
		if strings.HasPrefix(fields[0], "d") {
			name += "/"
		}
		out = append(out, name)
	}
	return strings.Join(out, "\n")
}

func Tree(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		name := treeName(line)
		if name != "" {
			out = append(out, name)
		}
	}
	return strings.Join(out, "\n")
}

func BuildOutput(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "ok ") || strings.HasPrefix(trimmed, "? ") {
			continue
		}
		out = append(out, strings.Join(strings.Fields(trimmed), " "))
	}
	return strings.Join(out, "\n")
}

func DedupLog(input string) string {
	seen := make(map[string]bool)
	var out []string
	for _, line := range strings.Split(input, "\n") {
		message := logMessage(strings.TrimSpace(line))
		if message == "" || seen[message] {
			continue
		}
		seen[message] = true
		out = append(out, message)
	}
	return strings.Join(out, "\n")
}

func SmartTruncate(input string) string {
	if len(input) <= smartTruncateLimit {
		return input
	}
	headLen := smartTruncateLimit / 2
	tailLen := smartTruncateLimit / 2
	omitted := len(input) - headLen - tailLen
	return input[:headLen] + fmt.Sprintf("\n... truncated %d bytes ...\n", omitted) + input[len(input)-tailLen:]
}

func ReadNumbered(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		i := 0
		for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
			i++
		}
		if i > 0 {
			trimmed = strings.TrimLeft(trimmed[i:], " \t")
		}
		out = append(out, strings.TrimRight(trimmed, " \t"))
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

func SearchList(input string) string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			out = append(out, strings.TrimSpace(trimmed[2:]))
		}
	}
	return strings.Join(out, "\n")
}

func treeName(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, "│|`-─├└ ")
	if strings.HasPrefix(trimmed, "── ") {
		trimmed = strings.TrimPrefix(trimmed, "── ")
	}
	return strings.TrimSpace(trimmed)
}

func logMessage(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	if strings.Contains(fields[0], "T") && len(fields) > 1 {
		return strings.Join(fields[1:], " ")
	}
	return strings.Join(fields, " ")
}
