package configview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// State icons for each item row.
const (
	iconInstalled = "✓"
	iconMissing   = "✗"
	iconBusy      = "…"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.modalActive() {
		return m.renderModal()
	}
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")
	b.WriteString(m.renderBody())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")
	b.WriteString(m.renderFooter())
	b.WriteString("\n")
	return b.String()
}

// renderModal renders the input-collection form. We frame huh's output with
// the same header so the user keeps context, and add a footer hint.
func (m Model) renderModal() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" install " + m.formScript.Name))
	b.WriteString("\n\n")
	if m.formScript.Help != "" {
		b.WriteString(detailTextStyle.Render(" " + wrapText(m.formScript.Help, m.contentWidth()-2)))
		b.WriteString("\n\n")
	}
	b.WriteString(m.form.View())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")
	b.WriteString(footerLabelStyle.Render(" enter confirm   esc cancel"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderHeader() string {
	left := titleStyle.Render("j config")
	right := contextStyle.Render(fmt.Sprintf("self: %s · %s", m.selfAlias, m.roleLabel()))
	return joinLeftRight(left, right, m.contentWidth())
}

func (m Model) roleLabel() string {
	if m.selfRole == "" {
		return "no role"
	}
	return string(m.selfRole)
}

func (m Model) renderDivider() string {
	w := m.contentWidth()
	if w <= 0 {
		w = 80
	}
	return dividerStyle.Render(strings.Repeat("─", w))
}

func (m Model) renderBody() string {
	if len(m.sections) == 0 {
		return contextStyle.Render("  No configurable items for the current role.")
	}
	var b strings.Builder
	for sIdx, section := range m.sections {
		if sIdx > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.renderSectionHeader(section))
		b.WriteString("\n")
		if section.Collapsed {
			continue
		}
		for iIdx, script := range section.Scripts {
			b.WriteString(m.renderItem(script, sIdx, iIdx))
			b.WriteString("\n")
			if m.expanded[script.Name] {
				b.WriteString(m.renderDetail(script))
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func (m Model) renderSectionHeader(s Section) string {
	caret := "▾"
	if s.Collapsed {
		caret = "▸"
	}
	installed, total := s.installedCount()
	var count string
	if total > 0 {
		count = fmt.Sprintf("%d/%d", installed, total)
	} else {
		count = fmt.Sprintf("%d", len(s.Scripts))
	}
	name := sectionHeaderStyle.Render(string(s.Category))
	return fmt.Sprintf(" %s %s   %s",
		dividerStyle.Render(caret),
		name,
		sectionCountStyle.Render(count),
	)
}

func (m Model) renderItem(s *config.Script, sectionIdx, itemIdx int) string {
	isCursor := m.cursor.section == sectionIdx && m.cursor.item == itemIdx

	icon := iconMissing
	iconStyle := stateMissingStyle
	if m.busy && m.cursor.section == sectionIdx && m.cursor.item == itemIdx {
		icon = iconBusy
		iconStyle = stateMissingStyle
	} else if isInstalled(s) {
		icon = iconInstalled
		iconStyle = stateInstalledStyle
	}

	cursorMark := "  "
	if isCursor {
		cursorMark = cursorStyle.Render("▶ ")
	}

	nameStyle := itemNameStyle
	if !isInstalled(s) {
		nameStyle = itemNameMutedStyle
	}

	row := fmt.Sprintf(" %s%s %s",
		cursorMark,
		iconStyle.Render(icon),
		nameStyle.Render(s.Name),
	)
	if isCursor {
		row = cursorRowStyle.Render(padToWidth(row, m.contentWidth()))
	}
	return row
}

func (m Model) renderDetail(s *config.Script) string {
	if s.Help == "" {
		return detailFrameStyle.Render("│ (no description)")
	}
	wrapped := wrapText(s.Help, m.contentWidth()-6)
	var lines []string
	lines = append(lines, "│")
	for _, line := range strings.Split(wrapped, "\n") {
		lines = append(lines, "│ "+detailTextStyle.Render(line))
	}
	lines = append(lines, "│")
	return detailFrameStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderFooter() string {
	if m.busy {
		return contextStyle.Render(" " + iconBusy + " " + m.busyAction + "…")
	}

	s := m.currentScript()
	if s == nil {
		if m.lastResult != "" {
			return m.renderResult()
		}
		return contextStyle.Render(" no item selected")
	}

	var hints []string
	if !isInstalled(s) && s.InstallFn != nil {
		hints = append(hints, footerKey("i", "install"))
	}
	if isInstalled(s) && s.UninstallFn != nil {
		hints = append(hints, footerKey("u", "uninstall"))
	}
	detailLabel := "details"
	if m.expanded[s.Name] {
		detailLabel = "close"
	}
	hints = append(hints, footerKey("space", detailLabel))

	prefix := footerLabelStyle.Render(" ▶ " + s.Name + "  ")
	keys := strings.Join(hints, footerSepStyle.Render("   "))

	footer := prefix + keys
	if m.lastResult != "" {
		footer = m.renderResult() + "\n" + footer
	}
	return footer
}

func (m Model) renderResult() string {
	if m.lastErr != nil {
		return resultErrStyle.Render(" ✗ " + m.lastResult)
	}
	return resultOkStyle.Render(" ✓ " + m.lastResult)
}

func footerKey(k, label string) string {
	return footerKeyStyle.Render(k) + " " + footerLabelStyle.Render(label)
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 80
	}
	return m.width
}

// joinLeftRight stretches content so left is at the start and right is at
// the far edge of the row, separated by spaces.
func joinLeftRight(left, right string, totalWidth int) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := totalWidth - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// padToWidth right-pads text with spaces so the visible width matches `w`.
// Useful for cursor row highlighting that needs to extend to the edge.
func padToWidth(text string, w int) string {
	current := lipgloss.Width(text)
	if current >= w {
		return text
	}
	return text + strings.Repeat(" ", w-current)
}

// wrapText word-wraps `s` to lines no wider than `width`. Preserves any
// pre-existing newlines.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		var line strings.Builder
		for _, w := range words {
			if line.Len() == 0 {
				line.WriteString(w)
				continue
			}
			if line.Len()+1+len(w) > width {
				out = append(out, line.String())
				line.Reset()
				line.WriteString(w)
				continue
			}
			line.WriteString(" ")
			line.WriteString(w)
		}
		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}
	return strings.Join(out, "\n")
}
