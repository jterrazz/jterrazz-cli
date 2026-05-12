package tool

import (
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// Output Parsers - Extract version strings from command output
// =============================================================================

// stripAnsi removes ANSI escape codes (colors, formatting) from a string
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// TrimVersion strips "v" prefix and whitespace
func TrimVersion(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(StripAnsi(s), "v"))
}

// parseFirstLineField extracts a field from the first line of output
// fieldIndex is 0-based, stripV optionally removes "v" prefix
func parseFirstLineField(s string, fieldIndex int, stripV bool) string {
	s = StripAnsi(s)
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) > fieldIndex {
			v := parts[fieldIndex]
			if stripV {
				v = strings.TrimPrefix(v, "v")
			}
			return v
		}
	}
	return ""
}

// ParseGhVersion parses "gh version 2.67.0 (2025-01-01)\n..." -> "2.67.0"
func ParseGhVersion(s string) string {
	return parseFirstLineField(s, 2, false)
}

// ParseBrewVersion parses "Homebrew 4.2.0\n..." -> "4.2.0"
func ParseBrewVersion(s string) string {
	return parseFirstLineField(s, 1, false)
}

// ParseGitVersion parses "git version 2.39.0 (Apple Git-145)" -> "2.39.0"
func ParseGitVersion(s string) string {
	s = StripAnsi(s)
	v := strings.TrimPrefix(strings.TrimSpace(s), "git version ")
	if idx := strings.Index(v, " ("); idx != -1 {
		v = v[:idx]
	}
	return v
}

// ParseTmuxVersion parses "tmux 3.6a" -> "3.6a"
func ParseTmuxVersion(s string) string {
	return parseFirstLineField(s, 1, false)
}

// ParseTailscaleVersion parses "1.86.2\ntailscale commit: ..." -> "1.86.2"
func ParseTailscaleVersion(s string) string {
	return parseFirstLineField(s, 0, false)
}

// ParseGoVersion parses "go version go1.23.4 darwin/arm64" -> "1.23.4"
func ParseGoVersion(s string) string {
	s = StripAnsi(s)
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		return strings.TrimPrefix(parts[2], "go")
	}
	return ""
}

// ParseJavaVersion parses 'openjdk version "21.0.1"...' -> "21.0.1"
func ParseJavaVersion(s string) string {
	s = StripAnsi(s)
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "version") {
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start != -1 && end != -1 && end > start {
				return line[start+1 : end]
			}
		}
	}
	return ""
}

// ParsePythonVersion parses "Python 3.12.0" -> "3.12.0"
func ParsePythonVersion(s string) string {
	s = StripAnsi(s)
	return strings.TrimPrefix(strings.TrimSpace(s), "Python ")
}

// ParseRustVersion parses "rustc 1.80.1 (....)" -> "1.80.1"
func ParseRustVersion(s string) string {
	return parseFirstLineField(s, 1, false)
}

// ParseTerraformVersion parses "Terraform v1.5.7\n..." -> "1.5.7"
func ParseTerraformVersion(s string) string {
	return parseFirstLineField(s, 1, true)
}

// ParseAnsibleVersion parses "ansible [core 2.15.0]" -> "2.15.0"
func ParseAnsibleVersion(s string) string {
	s = StripAnsi(s)
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		line := lines[0]
		if strings.Contains(line, "[core") {
			start := strings.Index(line, "[core ")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 {
				return line[start+6 : end]
			}
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

// ParseMultipassVersion parses "multipass 1.12.0+mac\n..." -> "1.12.0+mac"
func ParseMultipassVersion(s string) string {
	return parseFirstLineField(s, 1, false)
}

// ParseCodexVersion parses "codex 0.1.0" -> "0.1.0" or "0.1.0" -> "0.1.0"
func ParseCodexVersion(s string) string {
	v := parseFirstLineField(s, 1, false)
	if v == "" {
		// Fallback: return first field if only one field present
		return parseFirstLineField(s, 0, false)
	}
	return v
}

// ParseMoleVersion parses "\nMole version 1.14.5\n..." -> "1.14.5"
func ParseMoleVersion(s string) string {
	s = StripAnsi(s)
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "Mole version") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}

// ParseClaudeVersion parses "2.0.76 (Claude Code)" -> "2.0.76"
func ParseClaudeVersion(s string) string {
	return parseFirstLineField(s, 0, false)
}

// ParseHermesVersion parses "Hermes Agent v0.13.0 (2026.5.7)\n..." -> "0.13.0"
func ParseHermesVersion(s string) string {
	return parseFirstLineField(s, 2, true)
}

// ParseAnsibleLintVersion parses "ansible-lint 25.12.2 using..." -> "25.12.2"
func ParseAnsibleLintVersion(s string) string {
	return parseFirstLineField(s, 1, false)
}

// ParsePulumiVersion parses "v3.100.0" -> "3.100.0"
func ParsePulumiVersion(s string) string {
	s = StripAnsi(s)
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

// ParseEasVersion parses "eas-cli/16.32.0 darwin-arm64 node-v24.11.1" -> "16.32.0"
// Handles multi-line output with upgrade notices
func ParseEasVersion(s string) string {
	s = StripAnsi(s)
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "eas-cli/") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				return strings.TrimPrefix(parts[0], "eas-cli/")
			}
		}
	}
	return ""
}


// =============================================================================
// Formatters
// =============================================================================

// FormatBytes formats bytes into human-readable format (KB, MB, GB, etc.)
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// =============================================================================
// Slice Helpers
// =============================================================================

// FilterStrings returns elements from slice that are not in exclude
func FilterStrings(slice, exclude []string) []string {
	excludeSet := make(map[string]bool)
	for _, s := range exclude {
		excludeSet[s] = true
	}

	var result []string
	for _, s := range slice {
		if !excludeSet[s] {
			result = append(result, s)
		}
	}
	return result
}
