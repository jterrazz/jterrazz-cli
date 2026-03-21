package status

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/status"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/theme"
)

// ProcessRefreshMsg triggers a refresh of process data
type ProcessRefreshMsg struct{}

// ProcessDataMsg carries refreshed process data
type ProcessDataMsg struct {
	Data     map[string][]config.ProcessInfo
	TotalCPU float64
	GPUPct   float64
	NetRx    int64 // total rx bytes (for diffing)
	NetTx    int64 // total tx bytes (for diffing)
}

const sparkHistorySize = 60

// Model is the Bubble Tea model for the status view
type Model struct {
	loader        *status.Loader
	items         map[string]status.Item
	itemOrder     []status.Item
	spinner       spinner.Model
	viewport      viewport.Model
	ready         bool
	width         int
	height        int
	loaded        int
	total         int
	quitting      bool
	allLoaded     bool
	diskSortOrder []string  // cached disk item IDs once sorted
	diskMaxSize   float64  // cached max disk size for stable bar ratios
	cpuHistory    []float64 // last N seconds of total CPU usage
	gpuHistory    []float64 // last N seconds of GPU utilization
	netRxHistory  []float64 // last N seconds of network bytes received/sec
	netTxHistory  []float64 // last N seconds of network bytes sent/sec
	lastNetRx     int64     // previous sample total rx bytes
	lastNetTx     int64     // previous sample total tx bytes
}

// New creates a new status view model
func New() Model {
	loader := status.NewLoader()
	items := make(map[string]status.Item)
	itemOrder := loader.GetItems()

	total := 0
	for _, item := range itemOrder {
		items[item.ID] = item
		if !item.Loaded && item.Kind != status.KindHeader {
			total++
		}
	}

	return Model{
		loader:    loader,
		items:     items,
		itemOrder: itemOrder,
		spinner:   components.NewSpinnerModel(),
		total:     total,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	m.loader.Start()
	return tea.Batch(
		m.spinner.Tick,
		m.loader.WaitForUpdate(),
		scheduleProcessRefresh(),
	)
}

// scheduleProcessRefresh returns a command that triggers a process refresh after 1 second
func scheduleProcessRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return ProcessRefreshMsg{}
	})
}

// liveRefreshChecks are process checks that refresh every second
var liveRefreshChecks = map[string]bool{
	"CPU": true, "Memory": true, "Ports": true,
}

// refreshProcesses runs live process checks in background and returns the data
func refreshProcesses() tea.Cmd {
	return func() tea.Msg {
		data := make(map[string][]config.ProcessInfo)
		for _, check := range config.ProcessChecks {
			if liveRefreshChecks[check.Name] {
				data["process-"+check.Name] = check.CheckFn()
			}
		}
		// Compute total CPU from top processes
		var totalCPU float64
		if cpuData, ok := data["process-CPU"]; ok {
			for _, p := range cpuData {
				var v float64
				fmt.Sscanf(strings.TrimSuffix(p.Value, "%"), "%f", &v)
				totalCPU += v
			}
		}
		gpuPct := getGPUUtilization()
		netRx, netTx := getNetworkBytes()
		return ProcessDataMsg{Data: data, TotalCPU: totalCPU, GPUPct: gpuPct, NetRx: netRx, NetTx: netTx}
	}
}

// getNetworkBytes reads total rx/tx bytes from en0 via netstat
func getNetworkBytes() (int64, int64) {
	out, err := exec.Command("netstat", "-ib").Output()
	if err != nil {
		return 0, 0
	}
	// Find first en0 line with <Link#> (the raw interface line)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || !strings.HasPrefix(fields[0], "en0") {
			continue
		}
		if !strings.HasPrefix(fields[2], "<Link") {
			continue
		}
		var rx, tx int64
		fmt.Sscanf(fields[6], "%d", &rx)
		fmt.Sscanf(fields[9], "%d", &tx)
		return rx, tx
	}
	return 0, 0
}

// getGPUUtilization reads GPU device utilization from ioreg (macOS, no sudo)
func getGPUUtilization() float64 {
	out, err := exec.Command("ioreg", "-r", "-d", "1", "-c", "IOAccelerator").Output()
	if err != nil {
		return 0
	}
	// Look for "Device Utilization %" in the output
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, `"Device Utilization %"`) {
			// Format: "Device Utilization %" = 28
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				var v float64
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &v)
				return v
			}
		}
	}
	return 0
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "home", "g":
			m.viewport.GotoTop()
		case "end", "G":
			m.viewport.GotoBottom()
		}

	case tea.WindowSizeMsg:
		headerHeight := components.PageHeaderHeight(true) // title + subtitle
		footerHeight := 1

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}
		m.width = msg.Width
		m.height = msg.Height

	case status.UpdateMsg:
		if existing, ok := m.items[msg.ID]; ok {
			existing.Loaded = msg.Item.Loaded
			existing.Installed = msg.Item.Installed
			existing.Version = msg.Item.Version
			existing.Status = msg.Item.Status
			existing.Detail = msg.Item.Detail
			existing.Value = msg.Item.Value
			existing.Style = msg.Item.Style
			existing.Available = msg.Item.Available
			existing.Processes = msg.Item.Processes
			existing.TailscaleStatus = msg.Item.TailscaleStatus
			existing.ProjectGroups = msg.Item.ProjectGroups
			existing.DockerStatus = msg.Item.DockerStatus
			existing.DepGroups = msg.Item.DepGroups
			m.items[msg.ID] = existing
		} else {
			m.items[msg.ID] = msg.Item
		}
		if msg.Item.Loaded {
			m.loaded++
		}
		cmds = append(cmds, m.loader.WaitForUpdate())

	case status.AllLoadedMsg:
		m.allLoaded = true
		// Cache disk sort order so it never shuffles again
		var diskItems []status.Item
		for _, base := range m.itemOrder {
			item := m.items[base.ID]
			if item.Kind == status.KindCache && item.Loaded {
				diskItems = append(diskItems, item)
			}
		}
		sort.Slice(diskItems, func(i, j int) bool {
			si := parseDisplaySize(diskItems[i].Value)
			sj := parseDisplaySize(diskItems[j].Value)
			if si != sj {
				return si > sj
			}
			return diskItems[i].Name < diskItems[j].Name
		})
		m.diskSortOrder = make([]string, len(diskItems))
		m.diskMaxSize = 0
		for idx, item := range diskItems {
			m.diskSortOrder[idx] = item.ID
			s := parseDisplaySize(item.Value)
			if s > m.diskMaxSize {
				m.diskMaxSize = s
			}
		}

	case ProcessRefreshMsg:
		// Trigger async process data refresh
		cmds = append(cmds, refreshProcesses(), scheduleProcessRefresh())
		return m, tea.Batch(cmds...) // Don't re-render yet

	case ProcessDataMsg:
		// Apply refreshed process data
		for id, processes := range msg.Data {
			if existing, ok := m.items[id]; ok {
				existing.Processes = processes
				existing.Available = len(processes) > 0
				m.items[id] = existing
			}
		}
		// Push to history
		m.cpuHistory = append(m.cpuHistory, msg.TotalCPU)
		if len(m.cpuHistory) > sparkHistorySize {
			m.cpuHistory = m.cpuHistory[len(m.cpuHistory)-sparkHistorySize:]
		}
		m.gpuHistory = append(m.gpuHistory, msg.GPUPct)
		if len(m.gpuHistory) > sparkHistorySize {
			m.gpuHistory = m.gpuHistory[len(m.gpuHistory)-sparkHistorySize:]
		}
		// Network: diff with previous sample to get bytes/sec
		if m.lastNetRx > 0 && msg.NetRx >= m.lastNetRx {
			rxPerSec := float64(msg.NetRx - m.lastNetRx)
			txPerSec := float64(msg.NetTx - m.lastNetTx)
			m.netRxHistory = append(m.netRxHistory, rxPerSec)
			if len(m.netRxHistory) > sparkHistorySize {
				m.netRxHistory = m.netRxHistory[len(m.netRxHistory)-sparkHistorySize:]
			}
			m.netTxHistory = append(m.netTxHistory, txPerSec)
			if len(m.netTxHistory) > sparkHistorySize {
				m.netTxHistory = m.netTxHistory[len(m.netTxHistory)-sparkHistorySize:]
			}
		}
		m.lastNetRx = msg.NetRx
		m.lastNetTx = msg.NetTx

	case spinner.TickMsg:
		if !m.allLoaded {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
		// Only re-render content on spinner tick if still loading
		if m.ready && !m.allLoaded {
			m.viewport.SetContent(m.renderContent())
		}
		return m, tea.Batch(cmds...)
	}

	// Re-render content for data updates (UpdateMsg, ProcessDataMsg, etc.)
	if m.ready {
		m.viewport.SetContent(m.renderContent())
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return m.spinner.View() + " Initializing..."
	}

	var b strings.Builder

	// System info subtitle
	subtitle := ""
	if sysinfo, ok := m.items["sysinfo"]; ok && sysinfo.Loaded {
		subtitle = sysinfo.Detail
	} else {
		subtitle = m.spinner.View() + " Loading..."
	}

	// Header
	b.WriteString(components.PageHeader("STATUS", subtitle))

	// Content
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Footer
	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	help := fmt.Sprintf("↑/↓ scroll • g/G top/bottom • %d%% • q quit", scrollPercent)

	if m.allLoaded {
		footer := theme.Help.Render(help) + components.ColumnSeparator + theme.Success.Render(theme.IconCheck+" All checks complete")
		b.WriteString(footer)
	} else {
		progressBar := m.renderProgressBar()
		footer := theme.Help.Render(help) + components.ColumnSeparator + progressBar
		b.WriteString(footer)
	}

	return b.String()
}

func (m Model) renderProgressBar() string {
	if m.allLoaded {
		return theme.Success.Render(theme.IconCheck + " All checks complete")
	}

	width := 30
	filled := int(float64(m.loaded) / float64(m.total) * float64(width))
	if filled > width {
		filled = width
	}

	bar := theme.ProgressFilled.Render(strings.Repeat(theme.IconProgressFull, filled)) +
		theme.ProgressEmpty.Render(strings.Repeat(theme.IconProgressEmpty, width-filled))

	return fmt.Sprintf("%s %s %d/%d",
		m.spinner.View(),
		bar,
		m.loaded,
		m.total,
	)
}

// Run starts the status TUI
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunOrExit runs the status TUI and exits on error
func RunOrExit() {
	if err := Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// Helper to get visible length (strip ANSI)
func visibleLen(s string) int {
	return components.VisibleLen(s)
}
