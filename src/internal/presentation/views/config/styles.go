package configview

import "github.com/charmbracelet/lipgloss"

// Style palette for the j config TUI. Kept in one file so the look can be
// re-themed by editing constants here without touching layout code.
var (
	colorPrimary    = lipgloss.AdaptiveColor{Light: "#005f87", Dark: "#5fafd7"}
	colorSuccess    = lipgloss.AdaptiveColor{Light: "#2e8540", Dark: "#5fd75f"}
	colorMuted      = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#7a7a7a"}
	colorMutedDim   = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#5a5a5a"}
	colorInverseBg  = lipgloss.AdaptiveColor{Light: "#dadada", Dark: "#3a3a3a"}
	colorInverseFg  = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
	colorAccentText = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccentText)

	contextStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	dividerStyle = lipgloss.NewStyle().
			Foreground(colorMutedDim)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccentText)

	sectionCountStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	itemNameStyle = lipgloss.NewStyle().
			Foreground(colorAccentText)

	itemNameMutedStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	stateInstalledStyle = lipgloss.NewStyle().
				Foreground(colorSuccess)

	stateMissingStyle = lipgloss.NewStyle().
				Foreground(colorMutedDim)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	cursorRowStyle = lipgloss.NewStyle().
			Background(colorInverseBg).
			Foreground(colorInverseFg)

	detailFrameStyle = lipgloss.NewStyle().
				Foreground(colorMutedDim).
				PaddingLeft(2)

	detailTextStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	footerKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	footerLabelStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	footerSepStyle = lipgloss.NewStyle().
			Foreground(colorMutedDim)

	resultOkStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	resultErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))
)
