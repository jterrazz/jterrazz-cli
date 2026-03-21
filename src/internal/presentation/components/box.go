package components

import (
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// BoxStyle defines the border style for a box
type BoxStyle int

const (
	BoxRounded BoxStyle = iota // ╭─╮ │ │ ╰─╯
	BoxThick                   // ┏━┓ ┃ ┃ ┗━┛
)

// =============================================================================
// Section Header Box (thick borders)
// =============================================================================

// SectionHeader renders a section divider with horizontal lines
func SectionHeader(title string, width int) string {
	if width < 10 {
		width = 10
	}
	line := theme.SectionBorder.Render(strings.Repeat("━", width))
	label := " " + theme.SectionTitle.Render(strings.ToUpper(title))
	return line + "\n" + label + "\n" + line + "\n"
}

// =============================================================================
// Subsection Box (rounded borders with title)
// =============================================================================

// SubsectionBox renders a subsection with rounded borders and embedded title
// ╭─ Title ────────────────────────────────────────────────────────────────╮
// │ content line 1                                                         │
// │ content line 2                                                         │
// ╰────────────────────────────────────────────────────────────────────────╯
func SubsectionBox(title string, lines []string, width int) string {
	innerWidth := width - 4 // account for border + padding
	if innerWidth < 20 {
		innerWidth = 20
	}

	borderStyle := theme.SectionBorder

	// Build top border with title: ╭─ Title ─────────────────╮
	// Total width = innerWidth + 2 (for borders ╭ and ╮)
	// Left part: ╭─ (2 chars)
	// Title part: title
	// Right part: ─...─╮ (remaining chars)
	totalBorderChars := innerWidth + 2 // total horizontal space including corners
	leftPart := 2                      // "─ " after ╭
	rightPart := 2                     // " ─" before ╮
	titleSpace := len(title)           // title text
	remainingDashes := totalBorderChars - leftPart - titleSpace - rightPart + 1
	if remainingDashes < 1 {
		remainingDashes = 1
	}

	top := borderStyle.Render(theme.BoxRoundedTopLeft+theme.BoxRoundedHorizontal+" ") +
		theme.SubSection.Render(title) +
		borderStyle.Render(" "+strings.Repeat(theme.BoxRoundedHorizontal, remainingDashes)+theme.BoxRoundedTopRight)

	bottom := borderStyle.Render(theme.BoxRoundedBottomLeft + strings.Repeat(theme.BoxRoundedHorizontal, innerWidth+2) + theme.BoxRoundedBottomRight)

	// Pad content lines
	var paddedLines []string
	for _, line := range lines {
		paddedLines = append(paddedLines, padBoxLine(line, innerWidth))
	}

	return top + "\n" + strings.Join(paddedLines, "\n") + "\n" + bottom
}

// SubsectionBoxWithSeparator renders a subsection box with a horizontal separator
// between topLines and bottomLines.
// ╭─ Title ────────────────────────────────────────────────────────────────╮
// │ top line 1                                                             │
// │ top line 2                                                             │
// ├────────────────────────────────────────────────────────────────────────┤
// │ bottom line 1                                                          │
// │ bottom line 2                                                          │
// ╰────────────────────────────────────────────────────────────────────────╯
func SubsectionBoxWithSeparator(title string, topLines []string, bottomLines []string, width int) string {
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	borderStyle := theme.SectionBorder

	// Build top border with title (same as SubsectionBox)
	totalBorderChars := innerWidth + 2
	leftPart := 2
	rightPart := 2
	titleSpace := len(title)
	remainingDashes := totalBorderChars - leftPart - titleSpace - rightPart + 1
	if remainingDashes < 1 {
		remainingDashes = 1
	}

	top := borderStyle.Render(theme.BoxRoundedTopLeft+theme.BoxRoundedHorizontal+" ") +
		theme.SubSection.Render(title) +
		borderStyle.Render(" "+strings.Repeat(theme.BoxRoundedHorizontal, remainingDashes)+theme.BoxRoundedTopRight)

	separator := borderStyle.Render(theme.BoxRoundedTeeLeft + strings.Repeat(theme.BoxRoundedHorizontal, innerWidth+2) + theme.BoxRoundedTeeRight)

	bottom := borderStyle.Render(theme.BoxRoundedBottomLeft + strings.Repeat(theme.BoxRoundedHorizontal, innerWidth+2) + theme.BoxRoundedBottomRight)

	var paddedTop []string
	for _, line := range topLines {
		paddedTop = append(paddedTop, padBoxLine(line, innerWidth))
	}
	var paddedBottom []string
	for _, line := range bottomLines {
		paddedBottom = append(paddedBottom, padBoxLine(line, innerWidth))
	}

	result := top + "\n" + strings.Join(paddedTop, "\n") + "\n" + separator + "\n" + strings.Join(paddedBottom, "\n") + "\n" + bottom
	return result
}

// =============================================================================
// Helpers
// =============================================================================

// padBoxLine pads or truncates a line to fit inside a box with borders
func padBoxLine(line string, innerWidth int) string {
	borderStyle := theme.SectionBorder
	visLen := VisibleLen(line)
	if visLen > innerWidth {
		line = truncateAnsi(line, innerWidth)
		visLen = innerWidth
	}
	padding := innerWidth - visLen
	return borderStyle.Render(theme.BoxRoundedVertical+" ") + line + strings.Repeat(" ", padding) + borderStyle.Render(" "+theme.BoxRoundedVertical)
}

// truncateAnsi truncates a string with ANSI escape codes to a visible width
func truncateAnsi(s string, maxWidth int) string {
	var result strings.Builder
	visible := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if visible >= maxWidth {
			break
		}
		result.WriteRune(r)
		visible++
	}
	// Reset styling after truncation
	result.WriteString("\x1b[0m")
	return result.String()
}

// VisibleLen returns the visible length of a string, stripping ANSI escape codes
func VisibleLen(s string) int {
	inEscape := false
	length := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		length++
	}
	return length
}
