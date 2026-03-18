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
// Section Print Functions
// =============================================================================

// SectionDivider prints a section divider matching the status TUI style
func SectionDivider(title string) {
	w := termWidth()
	line := theme.SectionBorder.Render(strings.Repeat("━", w))
	label := " " + theme.SectionTitle.Render(strings.ToUpper(title))
	fmt.Println()
	fmt.Println(line)
	fmt.Println(label)
	fmt.Println(line)
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
