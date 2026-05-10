package configview

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// Model is the bubbletea model backing `j config`.
//
// State machine:
//   - sections: ordered, role-filtered groups of scripts
//   - cursor: which section + item is highlighted
//   - expanded: per-script id, is the inline detail panel open?
//   - busy / lastResult: action lifecycle
type Model struct {
	sections []Section
	cursor   cursorPos
	expanded map[string]bool

	selfAlias string
	selfRole  config.Role

	width, height int

	busy       bool
	busyAction string

	lastResult string
	lastErr    error
}

// cursorPos identifies the highlighted item: section index + item index
// within that section's Scripts slice.
type cursorPos struct {
	section int
	item    int
}

// NewModel constructs a fresh Model with all sections expanded and the
// cursor on the first item.
func NewModel() Model {
	alias, m, ok := config.SelfMachine()
	role := config.Role("")
	if ok {
		role = m.Role
	}
	if !ok {
		alias = "(unregistered)"
	}

	sections := buildSections(role)
	return Model{
		sections:  sections,
		cursor:    firstItemCursor(sections),
		expanded:  map[string]bool{},
		selfAlias: alias,
		selfRole:  role,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// firstItemCursor returns the position of the first item across all sections,
// or {-1, -1} if there are no items.
func firstItemCursor(sections []Section) cursorPos {
	for s := range sections {
		if len(sections[s].Scripts) > 0 {
			return cursorPos{section: s, item: 0}
		}
	}
	return cursorPos{section: -1, item: -1}
}

// currentScript returns the script under the cursor, or nil if the cursor
// isn't on a valid item.
func (m Model) currentScript() *config.Script {
	if m.cursor.section < 0 || m.cursor.section >= len(m.sections) {
		return nil
	}
	sec := m.sections[m.cursor.section]
	if m.cursor.item < 0 || m.cursor.item >= len(sec.Scripts) {
		return nil
	}
	return sec.Scripts[m.cursor.item]
}

// isInstalled reports the current install state of a script via its CheckFn.
// Scripts without a CheckFn are treated as "not installed" — they have no
// observable state, so the only verb that makes sense is install.
func isInstalled(s *config.Script) bool {
	if s == nil || s.CheckFn == nil {
		return false
	}
	return s.CheckFn().Installed
}

// rebuildSections re-runs buildSections (after an install/uninstall changed
// state) and clamps the cursor onto a valid item.
func (m *Model) rebuildSections() {
	m.sections = buildSections(m.selfRole)
	m.clampCursor()
}

// clampCursor ensures (m.cursor.section, m.cursor.item) point at a real
// script. If the cursor falls off the end (because items moved or vanished),
// snaps it to the closest valid position.
func (m *Model) clampCursor() {
	if len(m.sections) == 0 {
		m.cursor = cursorPos{section: -1, item: -1}
		return
	}
	if m.cursor.section < 0 {
		m.cursor.section = 0
	}
	if m.cursor.section >= len(m.sections) {
		m.cursor.section = len(m.sections) - 1
	}
	sec := m.sections[m.cursor.section]
	if len(sec.Scripts) == 0 {
		// section emptied; jump to the next non-empty one
		m.cursor = firstItemCursor(m.sections)
		return
	}
	if m.cursor.item >= len(sec.Scripts) {
		m.cursor.item = len(sec.Scripts) - 1
	}
	if m.cursor.item < 0 {
		m.cursor.item = 0
	}
}

// fnExecCommand adapts a Go func() error into a tea.ExecCommand so install
// and uninstall actions can release the terminal (sudo, prompts, key
// generation all need raw TTY access).
type fnExecCommand struct {
	fn func() error
}

func (f *fnExecCommand) Run() error           { return f.fn() }
func (*fnExecCommand) SetStdin(io.Reader)     {}
func (*fnExecCommand) SetStdout(io.Writer)    {}
func (*fnExecCommand) SetStderr(io.Writer)    {}

// actionDoneMsg is dispatched when a tea.Exec'd install/uninstall completes.
type actionDoneMsg struct {
	scriptName string
	verb       string // "install" or "uninstall"
	err        error
}

// runAction returns a tea.Cmd that releases the terminal, runs fn, then
// emits an actionDoneMsg. Used for both install and uninstall paths.
func runAction(scriptName, verb string, fn func() error) tea.Cmd {
	return tea.Exec(&fnExecCommand{fn: fn}, func(err error) tea.Msg {
		return actionDoneMsg{scriptName: scriptName, verb: verb, err: err}
	})
}

// Run starts the TUI. Use RunOrExit for the standard CLI entry point.
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunOrExit runs the TUI and exits with status 1 on error.
func RunOrExit() {
	if err := Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
