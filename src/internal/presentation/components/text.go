package components

import (
	"strings"

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

// CommandHeaderHeight is the number of lines a CommandHeader occupies:
// (blank, title, divider, blank).
const CommandHeaderHeight = 4

// CommandHeader renders the canonical j subcommand header used by every CLI
// command and TUI. Indented by one space; bold title on the left; optional
// muted context right-aligned; thin ─ divider underneath.
//
// Output (4 lines):
//
//	(blank line)
//	 j install                                       darwin · arm64
//	 ──────────────────────────────────────────────────────────────────
//	(blank line)
//
// command:  command path or action label, e.g. "j install" or "install autologin"
// context:  optional right-aligned info, e.g. "self: mac-mini · homelab"
// width:    target terminal width
func CommandHeader(command, context string, width int) string {
	if width < 20 {
		width = 80
	}

	const indent = " "
	title := theme.SectionTitle.Render(command)
	left := indent + title
	leftW := VisibleLen(left)

	titleLine := left
	if context != "" {
		right := theme.Muted.Render(context)
		rightW := VisibleLen(right)
		gap := width - leftW - rightW - 1 // -1 for trailing space
		if gap < 1 {
			gap = 1
		}
		titleLine = left + strings.Repeat(" ", gap) + right
	}

	dividerW := width - 1
	if dividerW < 4 {
		dividerW = 4
	}
	divider := indent + theme.Muted.Render(strings.Repeat("─", dividerW))

	return "\n" + titleLine + "\n" + divider + "\n\n"
}
