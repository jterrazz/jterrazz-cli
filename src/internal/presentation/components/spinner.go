package components

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// SpinnerFPS is the animation speed for all spinners.
const SpinnerFPS = 80 * time.Millisecond

// NewSpinnerModel creates a styled bubbles spinner.Model. Used by every TUI
// that needs a loader — j status' load progress, j config's busy state.
func NewSpinnerModel() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: theme.BrailleSpinner,
		FPS:    SpinnerFPS,
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSpinner))
	return s
}
