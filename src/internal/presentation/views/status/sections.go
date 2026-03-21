package status

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/status"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

const minColumnWidth = 44

// ─────────────────────────────────────────────────────────────────────────────
// Main renderer
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderContent() string {
	var b strings.Builder
	w := m.width
	if w < minColumnWidth {
		w = minColumnWidth
	}

	sections := m.groupBySection()

	// ── SYSTEM ───────────────────────────────────────────────────────
	b.WriteString(sectionDivider("SYSTEM", w))
	b.WriteString(m.renderActivity(sections, w))

	// ── ENVIRONMENT ──────────────────────────────────────────────────
	b.WriteString(sectionDivider("ENVIRONMENT", w))
	b.WriteString(m.renderEnvironment(sections, w))

	// ── WORKSPACE ────────────────────────────────────────────────────
	b.WriteString(sectionDivider("WORKSPACE", w))
	b.WriteString(m.renderWorkspace(sections, w))

	// ── SETUP ────────────────────────────────────────────────────────
	b.WriteString(sectionDivider("SETUP", w))
	b.WriteString(m.renderSetup(sections, w))

	// ── SOFTWARE ─────────────────────────────────────────────────────
	b.WriteString(sectionDivider("SOFTWARE", w))
	b.WriteString(m.renderTools(sections, w))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Section divider
// ─────────────────────────────────────────────────────────────────────────────

func sectionDivider(title string, width int) string {
	return components.SectionHeader(title, width)
}

// ─────────────────────────────────────────────────────────────────────────────
// ACTIVITY — sparklines + CPU/Memory columns
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderActivity(sections sectionMap, w int) string {
	var b strings.Builder

	// Activity box: System column | Sparklines column
	innerW := w - 4 // box border + padding

	// System info (left column)
	sysItems := m.getSubsectionItems(sections, "Environment", "Health")
	var sysRows []string
	for _, item := range sysItems {
		if item.Kind == status.KindProcess {
			for _, p := range item.Processes {
				sysRows = append(sysRows, " "+padName(p.Name, 12)+theme.Muted.Render(p.Value))
			}
		}
	}

	// Sparklines (right column)
	sysColW := innerW / 4
	if sysColW < 24 {
		sysColW = 24
	}
	sparkColW := innerW - sysColW - 1 // -1 for divider

	// All sparkline rows use: " LABEL " (7) + padding + graph + " VALUE " (8)
	const slLabelW = 7  // " CPU  " or " NET ↓"
	const slValueW = 8  // " 109%  " or "  3.4K "
	graphW := sparkColW * 40 / 100
	if graphW < 15 {
		graphW = 15
	}
	if graphW > sparkHistorySize {
		graphW = sparkHistorySize
	}
	slPad := sparkColW - slLabelW - graphW - slValueW
	if slPad < 0 {
		slPad = 0
	}

	// Helper to build a sparkline row
	buildSparkRow := func(label string, history []float64, colored bool) string {
		row := fmt.Sprintf(" %-6s", label)
		if len(history) > 0 {
			var spark string
			if colored {
				spark = renderColoredSparkline(history, graphW)
			} else {
				spark = renderAutoSparkline(history, graphW)
			}
			current := history[len(history)-1]
			var val string
			if colored {
				val = fmt.Sprintf("%5.0f%% ", current)
			} else {
				val = formatBytesPerSec(current) + " "
			}
			row += strings.Repeat(" ", slPad) + spark + theme.Muted.Render(" "+val)
		} else {
			row += strings.Repeat(" ", slPad+graphW) + theme.Muted.Render("      - ")
		}
		return row
	}

	sparkLines := []string{
		buildSparkRow("CPU", m.cpuHistory, true),
		buildSparkRow("GPU", m.gpuHistory, true),
		buildSparkRow("NET ↓", m.netRxHistory, false),
		buildSparkRow("NET ↑", m.netTxHistory, false),
	}

	var activityRows []string
	activityRows = append(activityRows, "") // breathing room

	if innerW >= minColWidthResponsive*2 {
		// Wide: 2 asymmetric columns (system ~25% | graphs ~75%)
		divider := theme.SectionBorder.Render("│")

		activityRows = append(activityRows,
			padTo(" "+theme.SubSection.Render("HEALTH"), sysColW)+divider+" "+theme.SubSection.Render("GRAPHS"))
		activityRows = append(activityRows, padTo("", sysColW)+divider)

		maxR := len(sysRows)
		if len(sparkLines) > maxR {
			maxR = len(sparkLines)
		}
		for i := 0; i < maxR; i++ {
			left := getOr(sysRows, i, "")
			right := getOr(sparkLines, i, "")
			activityRows = append(activityRows, padTo(left, sysColW)+divider+right)
		}
	} else {
		// Narrow: stacked
		activityRows = append(activityRows, " "+theme.SubSection.Render("HEALTH"))
		activityRows = append(activityRows, "")
		activityRows = append(activityRows, sysRows...)
		activityRows = append(activityRows, "")
		activityRows = append(activityRows, " "+theme.SubSection.Render("GRAPHS"))
		activityRows = append(activityRows, "")
		activityRows = append(activityRows, sparkLines...)
	}

	activityRows = append(activityRows, "") // breathing room

	b.WriteString(components.SubsectionBox("System", activityRows, w))
	b.WriteString("\n")

	// CPU | MEMORY in a box
	cpuItems := m.getSubsectionItems(sections, "System", "CPU")
	memItems := m.getSubsectionItems(sections, "System", "Memory")

	barColW := innerW / 2
	cpuRows := m.renderProcessBars(cpuItems, barColW-2)
	memRows := m.renderProcessBars(memItems, barColW-2)

	procContent := renderColumnsRaw([]namedColumn{
		{"CPU", cpuRows},
		{"Memory", memRows},
	}, innerW)

	var boxRows []string
	boxRows = append(boxRows, "") // breathing room
	boxRows = append(boxRows, strings.Split(strings.TrimRight(procContent, "\n"), "\n")...)
	boxRows = append(boxRows, "") // breathing room

	b.WriteString(components.SubsectionBox("Processes", boxRows, w))
	b.WriteString("\n")

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// ENVIRONMENT — Network | Services | System (3-column)
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderEnvironment(sections sectionMap, w int) string {
	numCols := 2
	if w < minColWidthResponsive*2 {
		numCols = 1
	}
	colWidth := w / numCols

	// Network box
	netItems := m.getSubsectionItems(sections, "Environment", "Network")
	netInnerW := colWidth - 4
	var netRows []string
	for _, item := range netItems {
		if item.Kind == status.KindNetwork && item.Loaded && item.Available {
			icon := ""
			if (item.Name == "vpn" || item.Name == "dns") && item.Style == "success" {
				icon = components.Badge(true) + " "
			} else if item.Name == "vpn" || item.Name == "dns" {
				icon = theme.Warning.Render("⚠") + " "
			}
			left := " " + item.Name
			right := icon + theme.Muted.Render(item.Value)
			gap := netInnerW - components.VisibleLen(left) - components.VisibleLen(right)
			if gap < 1 {
				gap = 1
			}
			netRows = append(netRows, left+strings.Repeat(" ", gap)+right)
		} else if item.Kind == status.KindNetwork && !item.Loaded {
			netRows = append(netRows, " "+item.Name+" "+m.loadingIndicator())
		}
	}

	// Tailscale box
	tsItems := m.getSubsectionItems(sections, "Environment", "Tailscale")
	tsInnerW := colWidth - 4
	var tsBox string
	for _, item := range tsItems {
		if item.Kind == status.KindTailscale && item.Loaded && item.Available && item.TailscaleStatus != nil {
			st := item.TailscaleStatus

			// Top rows: status, ip, exit node
			var tsTopRows []string

			// Status row — connected is neutral (mesh only, not privacy)
			statusVal := strings.ToLower(st.BackendState)
			if st.Mode != "" {
				statusVal += " (" + string(st.Mode) + ")"
			}
			left := " status"
			right := theme.Muted.Render(statusVal)
			gap := tsInnerW - components.VisibleLen(left) - components.VisibleLen(right)
			if gap < 1 {
				gap = 1
			}
			tsTopRows = append(tsTopRows, left+strings.Repeat(" ", gap)+right)

			// IP row
			if st.IP != "" {
				left = " ip"
				right = theme.Muted.Render(st.IP)
				gap = tsInnerW - components.VisibleLen(left) - components.VisibleLen(right)
				if gap < 1 {
					gap = 1
				}
				tsTopRows = append(tsTopRows, left+strings.Repeat(" ", gap)+right)
			}

			// Exit node row — ✓ only when active (real VPN), ⚠ when none (traffic exposed)
			exitVal := "none"
			exitIcon := theme.Warning.Render("⚠") + " "
			if st.ExitNode != "" {
				exitVal = st.ExitNode
				exitIcon = components.Badge(true) + " "
			}
			left = " exit node"
			right = exitIcon + theme.Muted.Render(exitVal)
			gap = tsInnerW - components.VisibleLen(left) - components.VisibleLen(right)
			if gap < 1 {
				gap = 1
			}
			tsTopRows = append(tsTopRows, left+strings.Repeat(" ", gap)+right)

			// Peer rows
			var tsPeerRows []string
			if len(st.Peers) > 0 {
				for _, peer := range st.Peers {
					peerName := " " + peer.Name
					peerIP := peer.IP
					onlineLabel := "offline"
					onlineIcon := ""
					if peer.Online {
						onlineLabel = "online"
						onlineIcon = " " + components.Badge(true)
					}

					// Format: " name          100.98.133.10 ✓ online"
					rightPart := theme.Muted.Render(peerIP) + onlineIcon + " " + theme.Muted.Render(onlineLabel)
					gap = tsInnerW - components.VisibleLen(peerName) - components.VisibleLen(rightPart)
					if gap < 1 {
						gap = 1
					}
					tsPeerRows = append(tsPeerRows, peerName+strings.Repeat(" ", gap)+rightPart)
				}
			} else {
				tsPeerRows = append(tsPeerRows, " "+theme.Muted.Render("no peers"))
			}

			tsBox = components.SubsectionBoxWithSeparator("Tailscale", tsTopRows, tsPeerRows, colWidth)
		} else if item.Kind == status.KindTailscale && !item.Loaded {
			tsBox = components.SubsectionBox("Tailscale", []string{" " + m.loadingIndicator()}, colWidth)
		} else if item.Kind == status.KindTailscale && item.Loaded && !item.Available {
			tsBox = components.SubsectionBox("Tailscale", []string{" " + theme.Muted.Render("not available")}, colWidth)
		}
	}

	// Services box (Containers + Ports)
	svcItems := m.getSubsectionItems(sections, "Environment", "Services")
	svcInnerW := colWidth - 4
	var svcRows []string
	for _, item := range svcItems {
		if item.Kind == status.KindProcess {
			for _, p := range item.Processes {
				prefix := " "
				if item.Name == "Containers" {
					prefix = " " + theme.ServiceRunning.Render("●") + " "
				}
				left := prefix + p.Name
				right := theme.Muted.Render(p.Value)
				leftLen := components.VisibleLen(left)
				rightLen := components.VisibleLen(right)
				gap := svcInnerW - leftLen - rightLen
				if gap < 1 {
					gap = 1
				}
				svcRows = append(svcRows, left+strings.Repeat(" ", gap)+right)
			}
		}
	}
	if len(svcRows) == 0 {
		svcRows = append(svcRows, " "+theme.Muted.Render("none"))
	}

	var boxes []string
	if len(netRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Network", netRows, colWidth))
	}
	if tsBox != "" {
		boxes = append(boxes, tsBox)
	}
	if len(svcRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Services", svcRows, colWidth))
	}

	if len(boxes) == 0 {
		return ""
	}

	if numCols > len(boxes) {
		numCols = len(boxes)
	}

	columns := make([][]string, numCols)
	colHeights := make([]int, numCols)
	for _, box := range boxes {
		shortest := 0
		for i := 1; i < numCols; i++ {
			if colHeights[i] < colHeights[shortest] {
				shortest = i
			}
		}
		columns[shortest] = append(columns[shortest], box)
		colHeights[shortest] += strings.Count(box, "\n") + 1
	}

	var renderedCols []string
	for _, col := range columns {
		renderedCols = append(renderedCols, lipgloss.JoinVertical(lipgloss.Left, col...))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...) + "\n"
}

// ─────────────────────────────────────────────────────────────────────────────
// WORKSPACE — Git | Disk (2-column)
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderWorkspace(sections sectionMap, w int) string {
	numCols := 2
	if w < minColWidthResponsive*2 {
		numCols = 1
	}
	colWidth := w / numCols
	innerW := colWidth - 4

	// Repositories box (tree-style)
	repoItems := m.getSubsectionItems(sections, "Workspace", "Repositories")
	repoInnerW := colWidth - 4
	var repoRows []string
	for _, item := range repoItems {
		if item.Kind == status.KindRepository {
			if !item.Loaded {
				repoRows = append(repoRows, " "+m.loadingIndicator())
			} else if !item.Available {
				repoRows = append(repoRows, " "+theme.Muted.Render("no repositories found"))
			} else {
				// Find max repo name width for alignment
				maxNameW := 0
				for _, group := range item.ProjectGroups {
					for _, repo := range group.Repos {
						if len(repo.Name) > maxNameW {
							maxNameW = len(repo.Name)
						}
					}
				}

				for gi, group := range item.ProjectGroups {
					if gi > 0 {
						repoRows = append(repoRows, "") // spacing between groups
					}
					// Project header
					repoRows = append(repoRows, " "+theme.SubSection.Render(group.Prefix+"/"))

					for ri, repo := range group.Repos {
						// Tree connector
						connector := "├─"
						if ri == len(group.Repos)-1 {
							connector = "└─"
						}

						branchStyle := theme.Muted

						// Status
						var statusStr string
						if repo.Clean {
							statusStr = theme.Success.Render("clean")
						} else {
							statusStr = theme.Warning.Render(fmt.Sprintf("%d changed", repo.ChangeCount))
						}

						left := "   " + theme.Muted.Render(connector) + " " + fmt.Sprintf("%-*s", maxNameW, repo.Name)
						right := statusStr + "  " + branchStyle.Render(repo.Branch)
						gap := repoInnerW - components.VisibleLen(left) - components.VisibleLen(right)
						if gap < 1 {
							gap = 1
						}
						repoRows = append(repoRows, left+strings.Repeat(" ", gap)+right)
					}
				}
			}
		}
	}

	// Disk box — only show bars after all loaded (cached sort + max)
	diskItems := m.getSubsectionItems(sections, "Workspace", "Disk")

	diskReady := len(m.diskSortOrder) > 0 && m.diskMaxSize > 0
	if diskReady {
		// Apply cached sort order
		orderMap := make(map[string]int, len(m.diskSortOrder))
		for i, id := range m.diskSortOrder {
			orderMap[id] = i
		}
		sort.SliceStable(diskItems, func(i, j int) bool {
			return orderMap[diskItems[i].ID] < orderMap[diskItems[j].ID]
		})
	}

	barWidth := 15
	valueWidth := 9
	nameWidth := innerW - barWidth - valueWidth - 3
	if nameWidth < 10 {
		nameWidth = 10
	}
	// Simpler width when no bars
	simpleNameWidth := innerW - valueWidth - 2
	if simpleNameWidth < 10 {
		simpleNameWidth = 10
	}

	var diskRows []string
	for _, item := range diskItems {
		if item.Kind != status.KindCache {
			continue
		}
		if !item.Loaded {
			diskRows = append(diskRows, " "+padName(item.Name, simpleNameWidth)+m.loadingIndicator())
		} else if item.Available {
			if diskReady {
				// Stable bars with cached max
				size := parseDisplaySize(item.Value)
				ratio := size / m.diskMaxSize
				if ratio > 1.0 {
					ratio = 1.0
				}
				bar := renderBar(ratio, barWidth)
				diskRows = append(diskRows, " "+fmt.Sprintf("%-*s", nameWidth, item.Name)+" "+bar+" "+theme.Muted.Render(fmt.Sprintf("%*s", valueWidth, item.Value)))
			} else {
				// Loading phase: just show name + value, no bars
				diskRows = append(diskRows, " "+fmt.Sprintf("%-*s", simpleNameWidth, item.Name)+" "+theme.Muted.Render(fmt.Sprintf("%*s", valueWidth, item.Value)))
			}
		}
	}

	var boxes []string
	if len(repoRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Repositories", repoRows, colWidth))
	}
	if len(diskRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Disk Usage", diskRows, colWidth))
	}

	if len(boxes) == 0 {
		return ""
	}

	if numCols > len(boxes) {
		numCols = len(boxes)
	}

	columns := make([][]string, numCols)
	colHeights := make([]int, numCols)
	for _, box := range boxes {
		shortest := 0
		for i := 1; i < numCols; i++ {
			if colHeights[i] < colHeights[shortest] {
				shortest = i
			}
		}
		columns[shortest] = append(columns[shortest], box)
		colHeights[shortest] += strings.Count(box, "\n") + 1
	}

	var renderedCols []string
	for _, col := range columns {
		renderedCols = append(renderedCols, lipgloss.JoinVertical(lipgloss.Left, col...))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...) + "\n"
}

// ─────────────────────────────────────────────────────────────────────────────
// SETUP — boxes in masonry
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderSetup(sections sectionMap, w int) string {
	numCols := 3
	if w < minColWidthResponsive*2 {
		numCols = 1
	} else if w < minColWidthResponsive*3 {
		numCols = 2
	}
	colWidth := w / numCols

	// Build check rows helper
	buildCheckRows := func(items []status.Item) []string {
		var rows []string
		for _, item := range items {
			if !item.Loaded {
				rows = append(rows, " "+m.loadingIndicator()+" "+item.Name)
			} else {
				ok := item.Installed
				if item.Kind == status.KindSecurity || item.Kind == status.KindIdentity {
					ok = item.Installed == item.GoodWhen
				}
				badge := components.Badge(ok)
				rows = append(rows, " "+badge+" "+item.Name)
			}
		}
		return rows
	}

	// Setup box
	setupRows := buildCheckRows(m.getSubsectionItems(sections, "Setup", "Setup"))

	// Security box (from Environment/System — KindSecurity items)
	var secItems []status.Item
	for _, item := range m.getSubsectionItems(sections, "Environment", "Health") {
		if item.Kind == status.KindSecurity {
			secItems = append(secItems, item)
		}
	}
	secRows := buildCheckRows(secItems)

	// Identity box
	idRows := buildCheckRows(m.getSubsectionItems(sections, "Setup", "Identity"))

	var boxes []string
	if len(setupRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Setup", setupRows, colWidth))
	}
	if len(secRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Security", secRows, colWidth))
	}
	if len(idRows) > 0 {
		boxes = append(boxes, components.SubsectionBox("Identity", idRows, colWidth))
	}

	if len(boxes) == 0 {
		return ""
	}

	if numCols > len(boxes) {
		numCols = len(boxes)
	}

	columns := make([][]string, numCols)
	colHeights := make([]int, numCols)
	for _, box := range boxes {
		shortest := 0
		for i := 1; i < numCols; i++ {
			if colHeights[i] < colHeights[shortest] {
				shortest = i
			}
		}
		columns[shortest] = append(columns[shortest], box)
		colHeights[shortest] += strings.Count(box, "\n") + 1
	}

	var renderedCols []string
	for _, col := range columns {
		renderedCols = append(renderedCols, lipgloss.JoinVertical(lipgloss.Left, col...))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...) + "\n"
}

// ─────────────────────────────────────────────────────────────────────────────
// TOOLS — column layout
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderTools(sections sectionMap, w int) string {
	subsections, ok := sections["Tools"]
	if !ok {
		return ""
	}

	// 3 groups: Core (foundation), CLI (command-line workflow), Apps (visual applications)
	type toolGroup struct {
		label      string
		categories []string
	}
	groups := []toolGroup{
		{"Core", []string{"Package Managers", "Runtimes"}},
		{"CLI", []string{"Terminal & Git", "DevOps", "AI"}},
		{"Apps", []string{"GUI Apps", "Mac App Store"}},
	}

	// Determine column count
	numCols := 3
	if w < minColWidthResponsive*2 {
		numCols = 1
	} else if w < minColWidthResponsive*3 {
		numCols = 2
	}

	colWidth := w / numCols

	// Build boxes for a set of categories
	buildBoxes := func(categories []string) []string {
		var boxes []string
		for _, sub := range categories {
			items, ok := subsections[sub]
			if !ok || len(items) == 0 {
				continue
			}
			nameW := 0
			for _, item := range items {
				if len(item.Name) > nameW {
					nameW = len(item.Name)
				}
			}
			innerW := colWidth - 4
			var rows []string
			for _, item := range items {
				rows = append(rows, m.renderToolRowCompact(item, nameW, innerW))
			}
			boxes = append(boxes, components.SubsectionBox(sub, rows, colWidth))
		}
		return boxes
	}

	// Masonry layout for a set of boxes
	masonryLayout := func(boxes []string) string {
		if len(boxes) == 0 {
			return ""
		}
		nc := numCols
		if nc > len(boxes) {
			nc = len(boxes)
		}
		columns := make([][]string, nc)
		colHeights := make([]int, nc)
		for _, box := range boxes {
			shortest := 0
			for i := 1; i < nc; i++ {
				if colHeights[i] < colHeights[shortest] {
					shortest = i
				}
			}
			columns[shortest] = append(columns[shortest], box)
			colHeights[shortest] += strings.Count(box, "\n") + 1
		}
		var renderedCols []string
		for _, col := range columns {
			renderedCols = append(renderedCols, lipgloss.JoinVertical(lipgloss.Left, col...))
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...) + "\n"
	}

	var b strings.Builder
	for _, group := range groups {
		boxes := buildBoxes(group.categories)
		if len(boxes) == 0 {
			continue
		}
		// Dotted separator: Label · · · · · · · · · · · · · · ·
		label := " " + group.label + " "
		labelLen := len(label)
		remaining := w - labelLen
		if remaining < 2 {
			remaining = 2
		}
		dots := remaining / 2
		separator := theme.Muted.Render("·") + theme.SubSection.Render(label) +
			theme.Muted.Render(strings.Repeat(" ·", dots))
		b.WriteString(separator + "\n")
		b.WriteString(masonryLayout(boxes))
	}

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Rendering helpers
// ─────────────────────────────────────────────────────────────────────────────

// renderToolRowCompact renders a tool row with right-aligned version
func (m Model) renderToolRowCompact(item status.Item, nameW int, rowW int) string {
	if !item.Loaded {
		return " " + m.loadingIndicator() + " " + fmt.Sprintf("%-*s", nameW, item.Name)
	}

	badge := components.Badge(item.Installed)
	name := fmt.Sprintf("%-*s", nameW, item.Name)

	// Clean up version
	version := item.Version
	if strings.HasPrefix(version, item.Name+" ") {
		version = strings.TrimPrefix(version, item.Name+" ")
	}
	if strings.HasPrefix(version, item.Name+"/") {
		version = strings.TrimPrefix(version, item.Name+"/")
	}
	if len(version) > 16 {
		version = version[:13] + "..."
	}

	// Left part: " ✓ name"
	left := " " + badge + " " + name

	// Right part: status icon + method + version
	right := ""
	if item.Status == "running" {
		right += theme.ServiceRunning.Render(theme.IconServiceOn) + " "
	} else if item.Status == "stopped" {
		right += theme.Muted.Render("○") + " "
	}

	if version != "" {
		right += theme.Muted.Render(version)
	}

	// Show install method when there's enough space
	method := item.Method
	if method != "" {
		// Minimum space: left + gap(1) + method + space(1) + version
		minNeeded := components.VisibleLen(left) + 1 + len(method) + 1 + components.VisibleLen(right)
		if minNeeded <= rowW {
			right += " " + theme.Muted.Render(method)
		}
	}

	if right == "" {
		return left
	}

	leftLen := components.VisibleLen(left)
	rightLen := components.VisibleLen(right)
	gap := rowW - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// getTotalMemoryMB returns total system RAM in MB via sysctl (cached).
var totalMemoryMB float64

func getTotalMemoryMB() float64 {
	if totalMemoryMB > 0 {
		return totalMemoryMB
	}
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 16 * 1024 // fallback 16GB
	}
	var bytes int64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &bytes)
	if bytes > 0 {
		totalMemoryMB = float64(bytes) / (1024 * 1024)
	} else {
		totalMemoryMB = 16 * 1024
	}
	return totalMemoryMB
}

// renderProcessBars renders process items with colored horizontal bars
func (m Model) renderProcessBars(items []status.Item, width int) []string {
	var rows []string
	for _, item := range items {
		if item.Kind != status.KindProcess {
			continue
		}
		if !item.Loaded {
			rows = append(rows, " "+m.loadingIndicator())
			continue
		}

		maxItems := 5
		barWidth := 15
		valueWidth := 7
		nameWidth := width - barWidth - valueWidth - 3 // spaces between
		if nameWidth < 10 {
			nameWidth = 10
		}

		// Find max value for relative scaling (memory uses relative, CPU uses absolute)
		var maxVal float64
		isPercent := len(item.Processes) > 0 && strings.HasSuffix(item.Processes[0].Value, "%")
		if isPercent {
			maxVal = 100.0
		} else {
			for _, p := range item.Processes {
				v := parseProcessValue(p.Value)
				if v > maxVal {
					maxVal = v
				}
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}

		// For memory: color based on % of total system RAM, not relative to top process
		totalMem := getTotalMemoryMB()

		for i, p := range item.Processes {
			if i >= maxItems {
				break
			}
			name := p.Name
			if nameWidth > 3 && len(name) > nameWidth {
				name = name[:nameWidth-3] + "..."
			}

			v := parseProcessValue(p.Value)
			sizeRatio := v / maxVal
			if sizeRatio > 1.0 {
				sizeRatio = 1.0
			}

			// Color ratio: for CPU use the value itself, for memory use % of total RAM
			colorRatio := sizeRatio
			if !isPercent {
				colorRatio = v / totalMem
				if colorRatio > 1.0 {
					colorRatio = 1.0
				}
			}

			bar := renderBarWithColor(sizeRatio, colorRatio, barWidth)
			rows = append(rows, " "+fmt.Sprintf("%-*s", nameWidth, name)+" "+bar+" "+theme.Muted.Render(fmt.Sprintf("%*s", valueWidth, p.Value)))
		}
	}
	return rows
}

// renderBar renders a colored horizontal bar (color based on size ratio)
func renderBar(ratio float64, width int) string {
	return renderBarWithColor(ratio, ratio, width)
}

// renderBarWithColor renders a horizontal bar where size and color are independent.
// sizeRatio controls bar fill, colorRatio controls green→yellow→red color.
func renderBarWithColor(sizeRatio float64, colorRatio float64, width int) string {
	if sizeRatio < 0 {
		sizeRatio = 0
	}
	if sizeRatio > 1 {
		sizeRatio = 1
	}
	filled := int(sizeRatio * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	// Color based on colorRatio: green → yellow → red
	var filledStyle lipgloss.Style
	switch {
	case colorRatio < 0.5:
		filledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSuccess))
	case colorRatio < 0.8:
		filledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorWarning))
	default:
		filledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDanger))
	}

	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorBorder))
	return filledStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", empty))
}

// renderColoredSparkline renders a sparkline with per-block coloring
func renderColoredSparkline(values []float64, width int) string {
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	data := values
	if len(data) > width {
		data = data[len(data)-width:]
	}

	maxVal := 100.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSuccess))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorWarning))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDanger))

	var result strings.Builder

	// Pad left with spaces
	pad := width - len(data)
	result.WriteString(strings.Repeat(" ", pad))

	for _, v := range data {
		ratio := v / maxVal
		idx := int(ratio * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}

		ch := string(blocks[idx])
		switch {
		case ratio < 0.4:
			result.WriteString(green.Render(ch))
		case ratio < 0.7:
			result.WriteString(yellow.Render(ch))
		default:
			result.WriteString(red.Render(ch))
		}
	}

	return result.String()
}

// renderAutoSparkline renders a sparkline that auto-scales to its own max (for bytes/sec)
func renderAutoSparkline(values []float64, width int) string {
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	data := values
	if len(data) > width {
		data = data[len(data)-width:]
	}

	maxVal := 1.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}

	green := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorSpecial))

	var result strings.Builder
	pad := width - len(data)
	result.WriteString(strings.Repeat(" ", pad))

	for _, v := range data {
		ratio := v / maxVal
		idx := int(ratio * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		result.WriteString(green.Render(string(blocks[idx])))
	}

	return result.String()
}

// formatBytesPerSec formats bytes/sec into a fixed 6-char right-aligned string
func formatBytesPerSec(b float64) string {
	switch {
	case b >= 1e9:
		return fmt.Sprintf("%5.1fG", b/1e9)
	case b >= 1e6:
		return fmt.Sprintf("%5.1fM", b/1e6)
	case b >= 1e3:
		return fmt.Sprintf("%5.1fK", b/1e3)
	default:
		return fmt.Sprintf("%5.0fB", b)
	}
}


// ─────────────────────────────────────────────────────────────────────────────
// Data helpers
// ─────────────────────────────────────────────────────────────────────────────

type sectionMap = map[string]map[string][]status.Item

func (m Model) groupBySection() sectionMap {
	sections := make(sectionMap)

	for _, baseItem := range m.itemOrder {
		item := m.items[baseItem.ID]
		if item.Kind == status.KindHeader || item.Kind == status.KindSystemInfo {
			continue
		}
		if sections[item.Section] == nil {
			sections[item.Section] = make(map[string][]status.Item)
		}
		sections[item.Section][item.SubSection] = append(sections[item.Section][item.SubSection], item)
	}

	// Sort tool items A-Z
	if toolSubs, ok := sections["Tools"]; ok {
		for sub, items := range toolSubs {
			sort.Slice(items, func(i, j int) bool {
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			})
			toolSubs[sub] = items
		}
	}

	return sections
}

func (m Model) getSubsectionItems(sections sectionMap, section, subsection string) []status.Item {
	if subs, ok := sections[section]; ok {
		if items, ok := subs[subsection]; ok {
			return items
		}
	}
	return nil
}

func parseDisplaySize(s string) float64 {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	// Check longest suffixes first to avoid "B" matching "GB"
	suffixes := []struct {
		suffix string
		mult   float64
	}{
		{"TB", 1e12},
		{"GB", 1e9},
		{"MB", 1e6},
		{"KB", 1e3},
		{"B", 1},
	}
	for _, entry := range suffixes {
		if strings.HasSuffix(upper, entry.suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(entry.suffix)])
			var val float64
			fmt.Sscanf(numStr, "%f", &val)
			return val * entry.mult
		}
	}
	return 0
}

// ─────────────────────────────────────────────────────────────────────────────
// Responsive multi-column layout
// ─────────────────────────────────────────────────────────────────────────────

type namedColumn struct {
	title string
	rows  []string
}

const minColWidthResponsive = 30

// renderColumnsImpl renders named columns responsively:
// wide terminal → all columns side by side with dividers
// narrow terminal → stacked vertically with headers
// renderColumnsImpl is the shared column renderer
func renderColumnsImpl(cols []namedColumn, totalWidth int, outerPadding bool) string {
	if len(cols) == 0 {
		return ""
	}

	numCols := totalWidth / minColWidthResponsive
	if numCols < 1 {
		numCols = 1
	}
	if numCols > 3 {
		numCols = 3
	}
	if numCols > len(cols) {
		numCols = len(cols)
	}

	var b strings.Builder
	divider := theme.SectionBorder.Render("│")

	for start := 0; start < len(cols); start += numCols {
		end := start + numCols
		if end > len(cols) {
			end = len(cols)
		}
		group := cols[start:end]
		colW := totalWidth / len(group)

		if outerPadding {
			b.WriteString("\n")
		}

		// Headers with space before title
		for i, col := range group {
			header := " " + theme.SubSection.Render(strings.ToUpper(col.title))
			if i < len(group)-1 {
				b.WriteString(padTo(header, colW) + divider)
			} else {
				b.WriteString(header)
			}
		}
		b.WriteString("\n")

		// Empty separator line
		for i := range group {
			if i < len(group)-1 {
				b.WriteString(padTo("", colW) + divider)
			}
		}
		b.WriteString("\n")

		// Content rows
		maxRows := 0
		for _, col := range group {
			if len(col.rows) > maxRows {
				maxRows = len(col.rows)
			}
		}

		for r := 0; r < maxRows; r++ {
			for i, col := range group {
				row := getOr(col.rows, r, "")
				if i < len(group)-1 {
					b.WriteString(padTo(row, colW) + divider)
				} else {
					b.WriteString(row)
				}
			}
			b.WriteString("\n")
		}
	}

	if outerPadding {
		b.WriteString("\n")
	}
	return b.String()
}

// renderColumnsRaw renders columns for embedding in boxes (no outer newlines)
func renderColumnsRaw(cols []namedColumn, totalWidth int) string {
	return renderColumnsImpl(cols, totalWidth, false)
}


// parseProcessValue parses "48.4%", "884M", "1.2G" into a comparable float
func parseProcessValue(s string) float64 {
	var v float64
	if strings.HasSuffix(s, "%") {
		fmt.Sscanf(strings.TrimSuffix(s, "%"), "%f", &v)
	} else if strings.HasSuffix(s, "G") {
		fmt.Sscanf(strings.TrimSuffix(s, "G"), "%f", &v)
		v *= 1024
	} else if strings.HasSuffix(s, "M") {
		fmt.Sscanf(strings.TrimSuffix(s, "M"), "%f", &v)
	} else if strings.HasSuffix(s, "K") {
		fmt.Sscanf(strings.TrimSuffix(s, "K"), "%f", &v)
		v /= 1024
	}
	return v
}

func padTo(s string, width int) string {
	pad := width - components.VisibleLen(s)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

func padName(name string, width int) string {
	if len(name) > width {
		name = name[:width-1] + "…"
	}
	return fmt.Sprintf("%-*s", width, name)
}

func getOr(slice []string, i int, fallback string) string {
	if i < len(slice) {
		return slice[i]
	}
	return fallback
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
