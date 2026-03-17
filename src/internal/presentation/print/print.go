package print

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// =============================================================================
// Basic Print Functions
// =============================================================================

// Line prints a line
func Line(s string) {
	fmt.Println(s)
}

// Linef prints a formatted line
func Linef(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

// Empty prints an empty line
func Empty() {
	fmt.Println()
}

// =============================================================================
// Message Print Functions
// =============================================================================

// Error prints an error message
func Error(msg string) {
	fmt.Printf("%s %s\n", theme.Danger.Render("Error:"), msg)
}

// Warning prints a warning message
func Warning(msg string) {
	fmt.Printf("%s %s\n", theme.Warning.Render("Warning:"), msg)
}

// Success prints a success message with checkmark
func Success(msg string) {
	fmt.Printf("%s %s\n", theme.Success.Render(theme.IconCheck), msg)
}

// Info prints an info message in cyan
func Info(msg string) {
	fmt.Println(theme.Special.Render(msg))
}

// Dim prints a dimmed/muted message
func Dim(msg string) {
	fmt.Println(theme.Muted.Render(msg))
}

// =============================================================================
// Action Print Functions
// =============================================================================

// Action prints an action being performed (e.g., "🔄 Updating...")
func Action(emoji, msg string) {
	fmt.Println(theme.Special.Render(emoji + " " + msg))
}

// Done prints a completion message
func Done(msg string) {
	fmt.Println(theme.Success.Render("✅ " + msg))
}

// Installing prints an installing message
func Installing(name string) {
	fmt.Printf(components.PageIndent+"📥 Installing %s...\n", name)
}

// InstallingVia prints an installing message with method
func InstallingVia(name, method string) {
	fmt.Printf(components.PageIndent+"📥 Installing %s (via %s)...\n", name, method)
}

// =============================================================================
// Section Print Functions
// =============================================================================

// Title prints a styled title
func Title(title string) {
	style := theme.Title.MarginTop(1)
	fmt.Println(style.Render(title))
}

// Section prints a section header
func Section(title string) {
	fmt.Println(theme.Section.Render(title))
}

// SubSection prints a subsection header
func SubSection(title string) {
	fmt.Println(theme.SubSection.Render(title))
}

// Category prints a category header (dimmed)
func Category(name string) {
	fmt.Println(theme.Muted.Render(name))
}

// =============================================================================
// Status Row Functions
// =============================================================================

// Row prints a status row (icon + label + detail)
func Row(ok bool, label, detail string) {
	icon := components.Badge(ok)
	if detail != "" {
		fmt.Printf(components.PageIndent+"%s %-14s %s\n", icon, label, theme.Muted.Render(detail))
	} else {
		fmt.Printf(components.PageIndent+"%s %s\n", icon, label)
	}
}

// =============================================================================
// Usage Print Functions
// =============================================================================

// Usage prints usage instructions
func Usage(lines ...string) {
	for _, line := range lines {
		fmt.Println(theme.Muted.Render(line))
	}
}

// =============================================================================
// Color Helpers (return styled string for use with fmt)
// =============================================================================

var (
	cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSpecial))
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSuccess))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDanger))
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorWarning))
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorMuted))
)

// Cyan returns cyan-colored text
func Cyan(s string) string {
	return cyan.Render(s)
}

// Green returns green-colored text
func Green(s string) string {
	return green.Render(s)
}

// Red returns red-colored text
func Red(s string) string {
	return red.Render(s)
}

// Yellow returns yellow-colored text
func Yellow(s string) string {
	return yellow.Render(s)
}

// Dimmed returns dimmed/muted text
func Dimmed(s string) string {
	return dim.Render(s)
}
