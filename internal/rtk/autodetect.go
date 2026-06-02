package rtk

import (
	"encoding/json"
	"strings"
)

const detectSampleBytes = 1024

func DetectFormat(input string) ContentFormat {
	sample := input
	if len(sample) > detectSampleBytes {
		sample = sample[:detectSampleBytes]
	}

	trimmed := strings.TrimSpace(sample)
	if trimmed == "" {
		return FormatUnknown
	}

	lines := nonEmptyLines(trimmed)

	switch {
	case isJSON(trimmed):
		return FormatJSON
	case isGitDiff(trimmed):
		return FormatGitDiff
	case isGitStatus(trimmed):
		return FormatGitStatus
	case isBuildOutput(trimmed):
		return FormatBuildOutput
	case isReadNumbered(trimmed):
		return FormatReadNumbered
	case isTree(trimmed):
		return FormatTree
	case isLS(trimmed):
		return FormatLS
	case isFind(lines):
		return FormatFind
	case isGrep(trimmed):
		return FormatGrep
	case isHTML(trimmed):
		return FormatHTML
	case isXML(trimmed):
		return FormatXML
	case isMarkdown(trimmed):
		return FormatMarkdown
	case isLog(trimmed):
		return FormatLog
	case isSearchList(lines):
		return FormatSearchList
	default:
		return FormatPlainText
	}
}

func isJSON(input string) bool {
	if !strings.HasPrefix(input, "{") && !strings.HasPrefix(input, "[") {
		return false
	}
	var value any
	return json.Unmarshal([]byte(input), &value) == nil
}

func isGitDiff(input string) bool {
	return strings.HasPrefix(input, "diff --git ") ||
		(strings.Contains(input, "\n--- ") && strings.Contains(input, "\n+++ ") && strings.Contains(input, "\n@@"))
}

func isGitStatus(input string) bool {
	return strings.HasPrefix(input, "On branch ") ||
		strings.HasPrefix(input, "## ") ||
		strings.Contains(input, "Changes not staged for commit:") ||
		strings.Contains(input, "Untracked files:")
}

func isBuildOutput(input string) bool {
	return strings.Contains(input, "\nFAIL\t") ||
		strings.HasPrefix(input, "# github.com/") ||
		strings.Contains(input, "undefined:") ||
		strings.Contains(input, "[build failed]")
}

func isReadNumbered(input string) bool {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimLeft(line, " \t")
		if line == "" || !isDigit(line[0]) {
			continue
		}
		for i := 1; i < len(line); i++ {
			if line[i] == ' ' || line[i] == '\t' {
				return true
			}
			if !isDigit(line[i]) {
				break
			}
		}
	}
	return false
}

func isTree(input string) bool {
	return strings.Contains(input, "├──") ||
		strings.Contains(input, "└──") ||
		strings.Contains(input, "|-- ") ||
		strings.Contains(input, "`-- ")
}

func isLS(input string) bool {
	return strings.HasPrefix(input, "total ") ||
		strings.HasPrefix(input, "drwx") ||
		strings.HasPrefix(input, "-rw") ||
		strings.Contains(input, "\ndrwx") ||
		strings.Contains(input, "\n-rw")
}

func isFind(lines []string) bool {
	if len(lines) < 2 {
		return false
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "./") && !strings.HasPrefix(line, "/") {
			return false
		}
	}
	return true
}

func isGrep(input string) bool {
	for _, line := range strings.Split(input, "\n") {
		first := strings.IndexByte(line, ':')
		if first <= 0 || first == len(line)-1 {
			continue
		}
		rest := line[first+1:]
		second := strings.IndexByte(rest, ':')
		if second <= 0 {
			continue
		}
		lineNumber := rest[:second]
		if allDigits(lineNumber) {
			return true
		}
	}
	return false
}

func isHTML(input string) bool {
	lower := strings.ToLower(input)
	return strings.HasPrefix(lower, "<!doctype html") ||
		strings.HasPrefix(lower, "<html") ||
		strings.Contains(lower, "<body")
}

func isXML(input string) bool {
	return strings.HasPrefix(input, "<?xml ")
}

func isMarkdown(input string) bool {
	return strings.HasPrefix(input, "# ") ||
		strings.Contains(input, "\n## ") ||
		strings.Contains(input, "\n- ") ||
		strings.Contains(input, "\n```")
}

func isLog(input string) bool {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[INFO]") || strings.HasPrefix(line, "[WARN]") || strings.HasPrefix(line, "[ERROR]") {
			return true
		}
		if strings.HasPrefix(line, "INFO ") || strings.HasPrefix(line, "WARN ") || strings.HasPrefix(line, "ERROR ") {
			return true
		}
		if len(line) >= len("2006-01-02T") && allDigits(line[:4]) && line[4] == '-' && line[7] == '-' {
			return true
		}
	}
	return false
}

func isSearchList(lines []string) bool {
	if len(lines) < 2 {
		return false
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			return false
		}
	}
	return true
}

func nonEmptyLines(input string) []string {
	rawLines := strings.Split(input, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func allDigits(input string) bool {
	if input == "" {
		return false
	}
	for i := 0; i < len(input); i++ {
		if !isDigit(input[i]) {
			return false
		}
	}
	return true
}

func isDigit(value byte) bool {
	return value >= '0' && value <= '9'
}
