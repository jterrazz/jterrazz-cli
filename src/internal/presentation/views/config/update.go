package configview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// keymap groups every key the model reacts to so the view can render hints
// from a single source of truth.
type keymap struct {
	Up        key.Binding
	Down      key.Binding
	TabPrev   key.Binding // ← / shift+tab — previous tab
	TabNext   key.Binding // → — next tab
	Toggle    key.Binding // tab — collapse/expand current section (Configuration tab)
	Details   key.Binding // space — toggle inline detail panel (Configuration tab)
	Install   key.Binding // i
	Uninstall key.Binding // u
	Quit      key.Binding // q / esc
	Cancel    key.Binding // ctrl+c (always quits)
}

var keys = keymap{
	Up:        key.NewBinding(key.WithKeys("up", "k")),
	Down:      key.NewBinding(key.WithKeys("down", "j")),
	TabPrev:   key.NewBinding(key.WithKeys("left", "shift+tab")),
	TabNext:   key.NewBinding(key.WithKeys("right")),
	Toggle:    key.NewBinding(key.WithKeys("tab")),
	Details:   key.NewBinding(key.WithKeys(" ")),
	Install:   key.NewBinding(key.WithKeys("i")),
	Uninstall: key.NewBinding(key.WithKeys("u")),
	Quit:      key.NewBinding(key.WithKeys("q", "esc")),
	Cancel:    key.NewBinding(key.WithKeys("ctrl+c")),
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case actionDoneMsg:
		m.busy = false
		m.busyAction = ""
		if msg.err != nil {
			m.lastErr = msg.err
			m.lastResult = fmt.Sprintf("Failed to %s %s: %s", msg.verb, msg.scriptName, msg.err)
		} else {
			m.lastErr = nil
			m.lastResult = fmt.Sprintf("%sed %s", capitalize(msg.verb), msg.scriptName)
		}
		// Refresh both caches — actionDoneMsg doesn't carry the tab origin
		// and the lookups are cheap. Configuration tab's rebuildSections
		// also clamps its cursor.
		m.rebuildSections()
		m.refreshSkillSections()
		return m, nil
	}

	// Modal owns key handling while it's up. Route everything to huh, then
	// react to its terminal state (completed → run install, aborted → close).
	if m.modalActive() {
		return m.updateModal(msg)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Ctrl+C always quits, even when busy.
		if key.Matches(keyMsg, keys.Cancel) {
			return m, tea.Quit
		}
		// Block other keys while an action is running.
		if m.busy {
			return m, nil
		}
		return m.handleKey(keyMsg)
	}
	return m, nil
}

// updateModal forwards the message to the huh form and reacts when the form
// reaches a terminal state. On completion, runs the closure stashed by
// buildModal/buildFormModal. On abort, just closes the modal.
func (m Model) updateModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	switch m.form.State {
	case huh.StateCompleted:
		onComplete := m.formOnComplete
		m.closeModal()
		m.lastResult = ""
		m.lastErr = nil
		if onComplete == nil {
			return m, nil
		}
		return m, onComplete()

	case huh.StateAborted:
		m.closeModal()
		return m, nil
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Number keys 1..N jump to a tab directly. Handled before the per-tab
	// keys so a stray "1" never lands on a list item.
	if s := msg.String(); len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		idx := int(s[0] - '1')
		if idx < len(TabLabels) {
			m.tabs.SetActive(idx)
			m.lastResult = ""
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.TabPrev):
		m.tabs.Prev()
		m.lastResult = ""
		return m, nil

	case key.Matches(msg, keys.TabNext):
		m.tabs.Next()
		m.lastResult = ""
		return m, nil

	case key.Matches(msg, keys.Up):
		m.cursorUpForActiveTab()
		return m, nil

	case key.Matches(msg, keys.Down):
		m.cursorDownForActiveTab()
		return m, nil

	case key.Matches(msg, keys.Toggle):
		m.toggleSectionForActiveTab()
		return m, nil

	case key.Matches(msg, keys.Details):
		s := m.currentScript()
		if s != nil {
			m.expanded[s.Name] = !m.expanded[s.Name]
		}
		return m, nil

	case key.Matches(msg, keys.Install):
		return m.installForActiveTab()

	case key.Matches(msg, keys.Uninstall):
		return m.uninstallForActiveTab()
	}
	return m, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Per-tab dispatch helpers — each tab owns its own cursor + actions.
// ─────────────────────────────────────────────────────────────────────────────

func (m *Model) cursorUpForActiveTab() {
	switch m.tabs.Active {
	case tabConfiguration:
		m.moveCursorUp()
	case tabSkills:
		m.moveSkillCursorUp()
	}
}

func (m *Model) cursorDownForActiveTab() {
	switch m.tabs.Active {
	case tabConfiguration:
		m.moveCursorDown()
	case tabSkills:
		m.moveSkillCursorDown()
	}
}

func (m *Model) toggleSectionForActiveTab() {
	switch m.tabs.Active {
	case tabConfiguration:
		m.toggleCurrentSection()
	case tabSkills:
		m.toggleCurrentSkillSection()
	}
}

func (m Model) installForActiveTab() (tea.Model, tea.Cmd) {
	switch m.tabs.Active {
	case tabConfiguration:
		return m.startInstall()
	case tabSkills:
		return m.skillStartInstall()
	case tabRemote:
		return m.remoteStartConfigure()
	}
	return m, nil
}

func (m Model) uninstallForActiveTab() (tea.Model, tea.Cmd) {
	switch m.tabs.Active {
	case tabConfiguration:
		return m.startUninstall()
	case tabSkills:
		return m.skillStartUninstall()
	}
	return m, nil
}

// startInstall fires the install action for the current item, if applicable.
// Refuses if already installed or no InstallFn. When the script declares
// Inputs, opens the modal first to collect them; the install runs after the
// form completes (see updateModal).
func (m Model) startInstall() (tea.Model, tea.Cmd) {
	s := m.currentScript()
	if s == nil || s.InstallFn == nil {
		return m, nil
	}
	if m.installed(s) {
		return m, nil
	}
	if len(s.Inputs) > 0 {
		m.buildModal(s)
		return m, m.form.Init()
	}
	m.busy = true
	m.busyAction = "install " + s.Name
	m.lastResult = ""
	m.lastErr = nil
	install := s.InstallFn
	return m, runAction(s.Name, "install", func() error { return install(config.InputValues{}) })
}

// startUninstall fires the uninstall action for the current item, if applicable.
// Refuses unless the item is currently installed AND has an UninstallFn.
func (m Model) startUninstall() (tea.Model, tea.Cmd) {
	s := m.currentScript()
	if s == nil || s.UninstallFn == nil {
		return m, nil
	}
	if !m.installed(s) {
		return m, nil
	}
	m.busy = true
	m.busyAction = "uninstall " + s.Name
	m.lastResult = ""
	m.lastErr = nil
	return m, runAction(s.Name, "uninstall", s.UninstallFn)
}

// toggleCurrentSection collapses or expands the section the cursor is in.
// When collapsing the current section, the cursor stays on its first item so
// expanding restores the same view.
func (m *Model) toggleCurrentSection() {
	if m.cursor.section < 0 || m.cursor.section >= len(m.sections) {
		return
	}
	m.sections[m.cursor.section].Collapsed = !m.sections[m.cursor.section].Collapsed
}

// moveCursorUp moves to the previous visible item, skipping over collapsed
// sections. No-op at the very top.
func (m *Model) moveCursorUp() {
	if m.cursor.section < 0 {
		return
	}
	// Try previous item in current section.
	if !m.sections[m.cursor.section].Collapsed && m.cursor.item > 0 {
		m.cursor.item--
		return
	}
	// Walk back through prior sections to find the last visible item.
	for s := m.cursor.section - 1; s >= 0; s-- {
		if m.sections[s].Collapsed || len(m.sections[s].Scripts) == 0 {
			continue
		}
		m.cursor.section = s
		m.cursor.item = len(m.sections[s].Scripts) - 1
		return
	}
}

// moveCursorDown moves to the next visible item, skipping over collapsed
// sections. No-op at the very bottom.
func (m *Model) moveCursorDown() {
	if m.cursor.section < 0 {
		return
	}
	sec := m.sections[m.cursor.section]
	// Try next item in current section.
	if !sec.Collapsed && m.cursor.item+1 < len(sec.Scripts) {
		m.cursor.item++
		return
	}
	// Walk forward to find the first visible item in a later section.
	for s := m.cursor.section + 1; s < len(m.sections); s++ {
		if m.sections[s].Collapsed || len(m.sections[s].Scripts) == 0 {
			continue
		}
		m.cursor.section = s
		m.cursor.item = 0
		return
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
