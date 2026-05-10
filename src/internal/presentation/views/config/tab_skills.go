package configview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/skill"
)

// skillEntry is one row in the Skills tab — a (repo, name) pair plus the
// derived install state. Built by refreshSkillSections from the static
// favourites list and the live `skill list` output.
type skillEntry struct {
	Repo string
	Name string
}

// skillSection groups skillEntries under a collapsible header.
type skillSection struct {
	Title     string
	Items     []skillEntry
	Collapsed bool
}

// installedCount counts how many of the section's skills are currently
// installed in the user's skills directory.
func (s skillSection) installedCount(installed map[string]bool) (n, total int) {
	total = len(s.Items)
	for _, e := range s.Items {
		if installed[e.Name] {
			n++
		}
	}
	return
}

// skillsAvailable reports whether the underlying `skills` CLI is installed.
// When false the Skills tab renders a "not installed" placeholder instead of
// a list — saves a confusing empty pane.
func skillsAvailable() bool {
	return skill.IsInstalled()
}

// refreshSkillSections rebuilds skillSections + skillsInstalled from the
// static favourites list and the live `skills list` output. Called on
// startup and after each install/uninstall completes.
func (m *Model) refreshSkillSections() {
	if !skillsAvailable() {
		m.skillSections = nil
		m.skillsInstalled = nil
		return
	}

	installed := skill.ListInstalled()
	m.skillsInstalled = make(map[string]bool, len(installed))
	for _, name := range installed {
		m.skillsInstalled[name] = true
	}

	m.skillSections = []skillSection{
		{Title: "Studio", Items: skillsToEntries(config.GetStudioSkills())},
		{Title: "Community", Items: skillsToEntries(config.GetCommunitySkills())},
	}

	// "Installed" section: anything currently installed that isn't already
	// pinned as a Studio or Community favourite. Repo column blank — we
	// only know the name from `skills list`.
	var others []skillEntry
	for _, name := range installed {
		if !config.IsFavoriteSkill("", name) {
			others = append(others, skillEntry{Name: name})
		}
	}
	if len(others) > 0 {
		m.skillSections = append(m.skillSections, skillSection{Title: "Installed", Items: others})
	}

	m.clampSkillCursor()
}

func skillsToEntries(in []config.Skill) []skillEntry {
	out := make([]skillEntry, len(in))
	for i, s := range in {
		out[i] = skillEntry{Repo: s.Repo, Name: s.Skill}
	}
	return out
}

// currentSkill returns the entry under the skill cursor, or zero if invalid.
func (m Model) currentSkill() (skillEntry, bool) {
	if m.skillCursor.section < 0 || m.skillCursor.section >= len(m.skillSections) {
		return skillEntry{}, false
	}
	sec := m.skillSections[m.skillCursor.section]
	if m.skillCursor.item < 0 || m.skillCursor.item >= len(sec.Items) {
		return skillEntry{}, false
	}
	return sec.Items[m.skillCursor.item], true
}

func (m Model) skillInstalled(e skillEntry) bool {
	return m.skillsInstalled[e.Name]
}

// clampSkillCursor mirrors clampCursor for the skills tab.
func (m *Model) clampSkillCursor() {
	if len(m.skillSections) == 0 {
		m.skillCursor = cursorPos{section: -1, item: -1}
		return
	}
	if m.skillCursor.section < 0 {
		m.skillCursor.section = 0
	}
	if m.skillCursor.section >= len(m.skillSections) {
		m.skillCursor.section = len(m.skillSections) - 1
	}
	sec := m.skillSections[m.skillCursor.section]
	if len(sec.Items) == 0 {
		m.skillCursor = firstSkillCursor(m.skillSections)
		return
	}
	if m.skillCursor.item >= len(sec.Items) {
		m.skillCursor.item = len(sec.Items) - 1
	}
	if m.skillCursor.item < 0 {
		m.skillCursor.item = 0
	}
}

func firstSkillCursor(sections []skillSection) cursorPos {
	for s := range sections {
		if len(sections[s].Items) > 0 {
			return cursorPos{section: s, item: 0}
		}
	}
	return cursorPos{section: -1, item: -1}
}

func (m *Model) moveSkillCursorUp() {
	if m.skillCursor.section < 0 {
		return
	}
	if !m.skillSections[m.skillCursor.section].Collapsed && m.skillCursor.item > 0 {
		m.skillCursor.item--
		return
	}
	for s := m.skillCursor.section - 1; s >= 0; s-- {
		if m.skillSections[s].Collapsed || len(m.skillSections[s].Items) == 0 {
			continue
		}
		m.skillCursor.section = s
		m.skillCursor.item = len(m.skillSections[s].Items) - 1
		return
	}
}

func (m *Model) moveSkillCursorDown() {
	if m.skillCursor.section < 0 {
		return
	}
	sec := m.skillSections[m.skillCursor.section]
	if !sec.Collapsed && m.skillCursor.item+1 < len(sec.Items) {
		m.skillCursor.item++
		return
	}
	for s := m.skillCursor.section + 1; s < len(m.skillSections); s++ {
		if m.skillSections[s].Collapsed || len(m.skillSections[s].Items) == 0 {
			continue
		}
		m.skillCursor.section = s
		m.skillCursor.item = 0
		return
	}
}

func (m *Model) toggleCurrentSkillSection() {
	if m.skillCursor.section < 0 || m.skillCursor.section >= len(m.skillSections) {
		return
	}
	m.skillSections[m.skillCursor.section].Collapsed = !m.skillSections[m.skillCursor.section].Collapsed
}

// skillStartInstall queues an install for the current entry. Refuses if the
// item is already installed or has no repo (Installed-section orphans).
func (m Model) skillStartInstall() (tea.Model, tea.Cmd) {
	e, ok := m.currentSkill()
	if !ok || e.Repo == "" {
		return m, nil
	}
	if m.skillInstalled(e) {
		return m, nil
	}
	m.busy = true
	m.busyAction = "install " + e.Name
	m.lastResult = ""
	m.lastErr = nil
	repo, name := e.Repo, e.Name
	return m, runAction(name, "install", func() error { return skill.Install(repo, name) })
}

// skillStartUninstall queues an uninstall for the current entry.
func (m Model) skillStartUninstall() (tea.Model, tea.Cmd) {
	e, ok := m.currentSkill()
	if !ok {
		return m, nil
	}
	if !m.skillInstalled(e) {
		return m, nil
	}
	m.busy = true
	m.busyAction = "uninstall " + e.Name
	m.lastResult = ""
	m.lastErr = nil
	name := e.Name
	return m, runAction(name, "uninstall", func() error { return skill.Remove(name) })
}

// renderSkillsBody renders the Skills tab content. Mirrors the layout of
// the Configuration tab (collapsible sections, cursor row, install state
// icons) so the two tabs feel like the same TUI.
func (m Model) renderSkillsBody() string {
	if !skillsAvailable() {
		return contextStyle.Render(" The `skills` CLI isn't installed. Run: npm install -g skills")
	}
	if len(m.skillSections) == 0 {
		return contextStyle.Render(" No skills available.")
	}
	var b strings.Builder
	for sIdx, section := range m.skillSections {
		if sIdx > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.renderSkillSectionHeader(section))
		b.WriteString("\n")
		if section.Collapsed {
			continue
		}
		for iIdx, entry := range section.Items {
			b.WriteString(m.renderSkillItem(entry, sIdx, iIdx))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) renderSkillSectionHeader(s skillSection) string {
	caret := "▾"
	if s.Collapsed {
		caret = "▸"
	}
	installed, total := s.installedCount(m.skillsInstalled)
	count := fmt.Sprintf("%d/%d", installed, total)
	return fmt.Sprintf(" %s %s   %s",
		dividerStyle.Render(caret),
		sectionHeaderStyle.Render(s.Title),
		sectionCountStyle.Render(count),
	)
}

func (m Model) renderSkillItem(e skillEntry, sectionIdx, itemIdx int) string {
	isCursor := m.tabs.Active == tabSkills && m.skillCursor.section == sectionIdx && m.skillCursor.item == itemIdx

	icon := iconMissing
	iconStyle := stateMissingStyle
	if m.busy && isCursor {
		icon = iconBusy
	} else if m.skillInstalled(e) {
		icon = iconInstalled
		iconStyle = stateInstalledStyle
	}

	cursorMark := "  "
	if isCursor {
		cursorMark = cursorStyle.Render("▶ ")
	}

	nameStyle := itemNameStyle
	if !m.skillInstalled(e) {
		nameStyle = itemNameMutedStyle
	}

	row := fmt.Sprintf(" %s%s %s",
		cursorMark,
		iconStyle.Render(icon),
		nameStyle.Render(e.Name),
	)
	if isCursor {
		row = cursorRowStyle.Render(padToWidth(row, m.contentWidth()))
	}
	return row
}
