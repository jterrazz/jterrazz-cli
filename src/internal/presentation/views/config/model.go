package configview

import (
	"fmt"
	"io"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
)

// TabLabels enumerates the j config tabs in display order. Index in this
// slice maps to Model.tabs.Active.
var TabLabels = []string{"Configuration", "Skills", "Remote"}

// Tab indices, kept as named constants so view code reads cleanly.
const (
	tabConfiguration = 0
	tabSkills        = 1
	tabRemote        = 2
)

// Model is the bubbletea model backing `j config`.
//
// Hosts three tabs:
//   - Configuration: install/uninstall items (sections + cursor + modal)
//   - Skills: install/uninstall AI agent skills
//   - Remote: configure Tailscale endpoint
//
// Per-tab state lives as named fields on the Model rather than a sum type,
// so each tab's render and update functions read directly from what they
// need without unwrapping.
type Model struct {
	tabs components.Tabs

	// ── Configuration tab ────────────────────────────────────────────
	sections []Section
	cursor   cursorPos
	expanded map[string]bool

	// checkCache stores the result of each Script's CheckFn so we don't
	// fork+exec a sub-process (defaults, pmset, systemsetup, gh auth
	// status, …) on every render. Refreshed at startup and after each
	// install/uninstall completes — never during navigation.
	checkCache map[string]config.CheckResult

	// ── Skills tab ───────────────────────────────────────────────────
	skillSections   []skillSection
	skillCursor     cursorPos
	skillsInstalled map[string]bool

	// ── Cross-tab ────────────────────────────────────────────────────
	selfAlias string
	selfRole  config.Role

	width, height int

	busy       bool
	busyAction string

	lastResult string
	lastErr    error

	// Modal state — non-nil while a huh form is up. Used both for the
	// install-with-inputs flow (formScript set, bindings auto-collected)
	// and for generic forms like the Remote-tab reconfigure (formScript
	// nil, bindings owned by the caller via closure).
	form           *huh.Form
	formScript     *config.Script
	formBindings   []*string
	formTitle      string         // header label, e.g. "install autologin"
	formHelp       string         // optional sub-text shown above the form
	formOnComplete func() tea.Cmd // fires when the user submits
}

// cursorPos identifies the highlighted item: section index + item index
// within that section's Scripts slice.
type cursorPos struct {
	section int
	item    int
}

// NewModel constructs a fresh Model with all sections expanded, the
// cursor on the first item, and a fresh check cache.
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
	model := Model{
		tabs:       components.Tabs{Labels: TabLabels, Active: tabConfiguration},
		sections:   sections,
		cursor:     firstItemCursor(sections),
		expanded:   map[string]bool{},
		checkCache: map[string]config.CheckResult{},
		selfAlias:  alias,
		selfRole:   role,
	}
	model.refreshCheckCache()
	model.refreshSkillSections()
	model.skillCursor = firstSkillCursor(model.skillSections)
	return model
}

// refreshCheckCache invokes every Script's CheckFn once (in parallel) and
// stores the results. Called at startup and after each install/uninstall
// lifecycle. CheckFn implementations typically fork+exec macOS commands
// (10-50ms each), so we deliberately avoid calling them on every render
// frame and fan them out across goroutines on the rare invalidations.
func (m *Model) refreshCheckCache() {
	if m.checkCache == nil {
		m.checkCache = map[string]config.CheckResult{}
	}

	type result struct {
		name string
		cr   config.CheckResult
	}
	var pending []*config.Script
	for _, sec := range m.sections {
		for _, s := range sec.Scripts {
			if s.CheckFn != nil {
				pending = append(pending, s)
			}
		}
	}
	if len(pending) == 0 {
		return
	}
	results := make([]result, len(pending))
	var wg sync.WaitGroup
	wg.Add(len(pending))
	for i, s := range pending {
		go func(i int, s *config.Script) {
			defer wg.Done()
			results[i] = result{name: s.Name, cr: s.CheckFn()}
		}(i, s)
	}
	wg.Wait()
	for _, r := range results {
		m.checkCache[r.name] = r.cr
	}
}

// cachedCheck returns the cached CheckResult for s, falling back to a
// live call if the script isn't in the cache (shouldn't normally happen).
func (m Model) cachedCheck(s *config.Script) config.CheckResult {
	if s == nil {
		return config.CheckResult{}
	}
	if r, ok := m.checkCache[s.Name]; ok {
		return r
	}
	if s.CheckFn != nil {
		return s.CheckFn()
	}
	return config.CheckResult{}
}

// installed reports whether the script is currently installed, reading
// from the cache.
func (m Model) installed(s *config.Script) bool {
	return m.cachedCheck(s).Installed
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

// sectionInstalledCount returns how many of the section's scripts are
// currently installed, reading from the check cache. Mirrors the semantic
// of Section.installedCount but never re-runs CheckFn during render.
func (m Model) sectionInstalledCount(sec Section) (installed, total int) {
	for _, sc := range sec.Scripts {
		if sc.CheckFn == nil {
			continue
		}
		total++
		if m.installed(sc) {
			installed++
		}
	}
	return
}

// rebuildSections re-runs buildSections (after an install/uninstall changed
// state), refreshes the check cache, and clamps the cursor onto a valid item.
func (m *Model) rebuildSections() {
	m.sections = buildSections(m.selfRole)
	m.refreshCheckCache()
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
