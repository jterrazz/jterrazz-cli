package configview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// keymap groups every key the model reacts to so the view can render hints
// from a single source of truth.
type keymap struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding // tab — collapse/expand current section
	Details   key.Binding // space — toggle inline detail panel
	Install   key.Binding // i
	Uninstall key.Binding // u
	Quit      key.Binding // q / esc
	Cancel    key.Binding // ctrl+c (always quits)
}

var keys = keymap{
	Up:        key.NewBinding(key.WithKeys("up", "k")),
	Down:      key.NewBinding(key.WithKeys("down", "j")),
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
		m.rebuildSections()
		return m, nil

	case tea.KeyMsg:
		// Ctrl+C always quits, even when busy.
		if key.Matches(msg, keys.Cancel) {
			return m, tea.Quit
		}
		// Block other keys while an action is running.
		if m.busy {
			return m, nil
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Up):
		m.moveCursorUp()
		return m, nil

	case key.Matches(msg, keys.Down):
		m.moveCursorDown()
		return m, nil

	case key.Matches(msg, keys.Toggle):
		m.toggleCurrentSection()
		return m, nil

	case key.Matches(msg, keys.Details):
		s := m.currentScript()
		if s != nil {
			m.expanded[s.Name] = !m.expanded[s.Name]
		}
		return m, nil

	case key.Matches(msg, keys.Install):
		return m.startInstall()

	case key.Matches(msg, keys.Uninstall):
		return m.startUninstall()
	}
	return m, nil
}

// startInstall fires the install action for the current item, if applicable.
// Refuses if already installed or no InstallFn.
func (m Model) startInstall() (tea.Model, tea.Cmd) {
	s := m.currentScript()
	if s == nil || s.InstallFn == nil {
		return m, nil
	}
	if isInstalled(s) {
		return m, nil
	}
	m.busy = true
	m.busyAction = "install " + s.Name
	m.lastResult = ""
	m.lastErr = nil
	return m, runAction(s.Name, "install", s.InstallFn)
}

// startUninstall fires the uninstall action for the current item, if applicable.
// Refuses unless the item is currently installed AND has an UninstallFn.
func (m Model) startUninstall() (tea.Model, tea.Cmd) {
	s := m.currentScript()
	if s == nil || s.UninstallFn == nil {
		return m, nil
	}
	if !isInstalled(s) {
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
