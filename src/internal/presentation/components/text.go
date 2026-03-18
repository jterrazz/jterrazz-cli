package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// =============================================================================
// Styled Text Helpers
// =============================================================================

// Muted renders text in muted style
func Muted(text string) string {
	return theme.Muted.Render(text)
}

// =============================================================================
// Layout Constants
// =============================================================================

// ColumnSeparator is the standard separator between columns
const ColumnSeparator = "  "

// RenderDescription renders a description in muted style with column separator
func RenderDescription(text string) string {
	if text == "" {
		return ""
	}
	return theme.Muted.Render(ColumnSeparator + text)
}

// PageIndent is the prefix for page-level headers and titles
const PageIndent = " "

// =============================================================================
// Page Header
// =============================================================================

// PageHeaderHeight returns the number of lines used by the page header
func PageHeaderHeight(hasSubtitle bool) int {
	if hasSubtitle {
		return 5
	}
	return 4
}

// PageHeader renders a page header with title and optional subtitle
func PageHeader(title string, subtitle string) string {
	var lines []string
	lines = append(lines, "") // Top padding
	lines = append(lines, PageIndent+theme.SectionTitle.Render(strings.ToUpper(title)))
	if subtitle != "" {
		lines = append(lines, PageIndent+theme.Muted.Render(subtitle))
	}
	lines = append(lines, "") // Bottom padding
	return lipgloss.JoinVertical(lipgloss.Left, lines...) + "\n"
}
