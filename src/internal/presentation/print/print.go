package print

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// =============================================================================
// Basic Print Functions
// =============================================================================

func Line(s string)                    { fmt.Println(s) }
func Linef(format string, args ...any) { fmt.Printf(format+"\n", args...) }
func Empty()                           { fmt.Println() }

// =============================================================================
// Message Print Functions
// =============================================================================

func Error(msg string)   { fmt.Printf("%s %s\n", theme.Danger.Render("Error:"), msg) }
func Warning(msg string) { fmt.Printf("%s %s\n", theme.Warning.Render("Warning:"), msg) }
func Success(msg string) { fmt.Printf("%s %s\n", theme.Success.Render(theme.IconCheck), msg) }
func Info(msg string)    { fmt.Println(theme.Special.Render(msg)) }
func Dim(msg string)     { fmt.Println(theme.Muted.Render(msg)) }

// =============================================================================
// Action Print Functions
// =============================================================================

func Action(emoji, msg string) { fmt.Println(theme.Special.Render(emoji + " " + msg)) }
func Done(msg string)          { fmt.Println(theme.Success.Render("✅ " + msg)) }

func Installing(name string) {
	fmt.Printf(components.PageIndent+"📥 Installing %s...\n", name)
}

// =============================================================================
// Role chip (used in header contexts)
// =============================================================================

// roleClientStyle / roleServerStyle colour the role chip so it pops without
// shouting. Cool for client, warm for server.
var (
	roleClientStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafd7"))
	roleServerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fd75f"))
)

// RenderRole returns the role string with a subtle colour for client/server.
// Falls back to plain text for unknown roles. Lives here (rather than in
// commands or config) so TUIs and CLI commands share the same rendering
// without pulling in a config dependency on the print package.
func RenderRole(role string) string {
	switch role {
	case "client":
		return roleClientStyle.Render(role)
	case "server":
		return roleServerStyle.Render(role)
	}
	return role
}

// MutedText returns text in the muted style — handy for "(unregistered)" or
// other small bits of context that should fade into the background.
func MutedText(s string) string { return theme.Muted.Render(s) }

// =============================================================================
// Header — canonical command header used by every j subcommand and TUI
// =============================================================================

// Header prints the canonical command header to stdout. Use for CLI commands
// (non-TUI). For TUIs that need to embed the header in their View string,
// use RenderHeader.
//
// command:  command path or action label, e.g. "j install", "install autologin".
//           Always lowercase, never decorated.
// context:  optional right-aligned info, e.g. "self: mac-mini · server".
//           Pass "" to omit (no placeholder rendered).
//
// Output:
//
//	(blank line)
//	 j install                                       darwin · arm64
//	 ──────────────────────────────────────────────────────────────────
//	(blank line)
func Header(command, context string) {
	fmt.Print(RenderHeader(command, context, termWidth()))
}

// RenderHeader returns the canonical header as a string, sized to the given
// width. Use from inside bubbletea View() functions or anywhere you need the
// rendered string instead of stdout output. Thin wrapper over
// components.CommandHeader.
func RenderHeader(command, context string, width int) string {
	return components.CommandHeader(command, context, width)
}

// Category prints a category header (dimmed)
func Category(name string) {
	fmt.Println(theme.Muted.Render(name))
}

func termWidth() int {
	if w, _ := strconv.Atoi(os.Getenv("COLUMNS")); w > 0 {
		return w
	}
	out, err := exec.Command("tput", "cols").Output()
	if err == nil {
		if w, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && w > 0 {
			return w
		}
	}
	return 80
}

// =============================================================================
// Status Row
// =============================================================================

func Row(ok bool, label, detail string) {
	icon := components.Badge(ok)
	if detail != "" {
		fmt.Printf(components.PageIndent+"%s %-14s %s\n", icon, label, theme.Muted.Render(detail))
	} else {
		fmt.Printf(components.PageIndent+"%s %s\n", icon, label)
	}
}

// =============================================================================
// Usage
// =============================================================================

func Usage(lines ...string) {
	for _, line := range lines {
		fmt.Println(theme.Muted.Render(line))
	}
}

// =============================================================================
// Color Helpers (return styled string for use with fmt)
// =============================================================================

var (
	cyan  = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSpecial))
	green = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSuccess))
	dim   = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorMuted))
)

func Cyan(s string) string    { return cyan.Render(s) }
func Green(s string) string   { return green.Render(s) }
func Dimmed(s string) string  { return dim.Render(s) }
