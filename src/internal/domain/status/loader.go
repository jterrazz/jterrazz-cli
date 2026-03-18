package status

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
)

// ItemKind represents the type of status item
type ItemKind int

const (
	KindHeader ItemKind = iota
	KindSetup
	KindSecurity
	KindIdentity
	KindTool
	KindProcess
	KindNetwork
	KindCache
	KindSystemInfo
)

// Item represents a single item in the status display
type Item struct {
	ID          string
	Kind        ItemKind
	Section     string
	SubSection  string
	Name        string
	Description string
	Loaded      bool

	// Result data (populated after loading)
	Installed bool
	Version   string
	Status    string
	Detail    string
	Value     string
	Style     string // Semantic style: "success", "warning", "muted", etc.
	GoodWhen  bool   // For checks: true means Installed=true is good
	Method    string // Install method for tools
	Available bool   // For resources: whether the resource exists

	// Process data (for KindProcess items)
	Processes []config.ProcessInfo
}

// UpdateMsg is sent when a status item finishes loading
type UpdateMsg struct {
	ID   string
	Item Item
}

// AllLoadedMsg is sent when all items have finished loading
type AllLoadedMsg struct{}

// Loader manages parallel loading of status items
type Loader struct {
	items   []Item
	updates chan UpdateMsg
	started bool
	mu      sync.Mutex
}

// NewLoader creates a new loader with all items in pending state
func NewLoader() *Loader {
	loader := &Loader{
		updates: make(chan UpdateMsg, 100),
	}
	loader.buildItems()
	return loader
}

// GetItems returns a copy of all items
func (l *Loader) GetItems() []Item {
	l.mu.Lock()
	defer l.mu.Unlock()
	items := make([]Item, len(l.items))
	copy(items, l.items)
	return items
}

// GetPendingCount returns the number of items that need loading
func (l *Loader) GetPendingCount() int {
	count := 0
	for _, item := range l.items {
		if !item.Loaded && item.Kind != KindHeader {
			count++
		}
	}
	return count
}

// buildItems creates all status items in display order
func (l *Loader) buildItems() {
	// System info (used in header subtitle)
	l.addItem(Item{
		ID:      "sysinfo",
		Kind:    KindSystemInfo,
		Section: "Activity",
		Name:    "System Info",
	})

	// ── Activity ──────────────────────────────────────────────────────
	// CPU and Memory process checks
	for _, check := range config.ProcessChecks {
		section := "Activity"
		subsection := check.Name
		// Services = Containers + Ports
		if check.Name == "Containers" || check.Name == "Ports" {
			section = "Environment"
			subsection = "Services"
		}
		// Uptime goes to Environment/System
		if check.Name == "Uptime" {
			section = "Environment"
			subsection = "System"
		}
		// Git goes to Workspace
		if check.Name == "Git" {
			section = "Workspace"
			subsection = "Git"
		}

		l.addItem(Item{
			ID:         "process-" + check.Name,
			Kind:       KindProcess,
			Section:    section,
			SubSection: subsection,
			Name:       check.Name,
		})
	}

	// ── Environment ───────────────────────────────────────────────────
	// Network checks
	for _, check := range config.NetworkChecks {
		l.addItem(Item{
			ID:         "network-" + check.Name,
			Kind:       KindNetwork,
			Section:    "Environment",
			SubSection: "Network",
			Name:       check.Name,
		})
	}

	// Security checks → Environment/System
	for _, check := range config.SecurityChecks {
		l.addItem(Item{
			ID:          "security-" + check.Name,
			Kind:        KindSecurity,
			Section:     "Environment",
			SubSection:  "System",
			Name:        check.Name,
			Description: check.Description,
			GoodWhen:    check.GoodWhen,
		})
	}

	// ── Workspace ─────────────────────────────────────────────────────
	// Disk/cache checks
	for _, check := range config.CacheChecks {
		l.addItem(Item{
			ID:         "cache-" + check.Name,
			Kind:       KindCache,
			Section:    "Workspace",
			SubSection: "Disk",
			Name:       check.Name,
		})
	}

	// ── Setup ─────────────────────────────────────────────────────────
	// Setup scripts
	for _, script := range config.Scripts {
		if script.CheckFn == nil {
			continue
		}
		l.addItem(Item{
			ID:          "setup-" + script.Name,
			Kind:        KindSetup,
			Section:     "Setup",
			SubSection:  "Setup",
			Name:        script.Name,
			Description: script.Description,
		})
	}
	l.addItem(Item{
		ID:          "setup-remote",
		Kind:        KindSetup,
		Section:     "Setup",
		SubSection:  "Setup",
		Name:        "remote",
		Description: "Configure remote SSH access",
	})

	// Identity checks
	for _, check := range config.IdentityChecks {
		l.addItem(Item{
			ID:          "identity-" + check.Name,
			Kind:        KindIdentity,
			Section:     "Setup",
			SubSection:  "Identity",
			Name:        check.Name,
			Description: check.Description,
			GoodWhen:    check.GoodWhen,
		})
	}

	// ── Tools ─────────────────────────────────────────────────────────
	for _, category := range config.ToolCategories {
		tools := config.GetToolsByCategory(category)
		if len(tools) == 0 {
			continue
		}
		for _, t := range tools {
			l.addItem(Item{
				ID:         "tool-" + t.Name,
				Kind:       KindTool,
				Section:    "Tools",
				SubSection: string(category),
				Name:       t.Name,
				Method:     t.Method.String(),
			})
		}
	}
}

func (l *Loader) addItem(item Item) {
	l.items = append(l.items, item)
}

// Start launches all checks in parallel (call only once)
func (l *Loader) Start() {
	l.mu.Lock()
	if l.started {
		l.mu.Unlock()
		return
	}
	l.started = true
	l.mu.Unlock()

	var wg sync.WaitGroup

	// System info
	wg.Add(1)
	go func() {
		defer wg.Done()
		item := l.loadSystemInfo()
		l.updates <- UpdateMsg{ID: item.ID, Item: item}
	}()

	// Setup checks
	for _, script := range config.Scripts {
		if script.CheckFn == nil {
			continue
		}
		wg.Add(1)
		go func(s config.Script) {
			defer wg.Done()
			result := config.CheckScript(s)
			item := Item{
				ID:        "setup-" + s.Name,
				Kind:      KindSetup,
				Name:      s.Name,
				Loaded:    true,
				Installed: result.Installed,
				Detail:    result.Detail,
			}
			l.updates <- UpdateMsg{ID: item.ID, Item: item}
		}(script)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		item := Item{
			ID:     "setup-remote",
			Kind:   KindSetup,
			Name:   "remote",
			Loaded: true,
		}
		settings, err := config.LoadRemoteSettings()
		if err == nil && config.ValidateRemoteSettings(settings) == nil {
			item.Installed = true
			detail := fmt.Sprintf("%s/%s", settings.Mode, settings.AuthMethod)
			if settings.Hostname != "" {
				detail += " " + settings.Hostname
			}
			if st, statusErr := config.RemoteStatusInfo(settings); statusErr == nil {
				if st.Connected {
					state := "connected"
					if st.Mode != "" {
						state += " " + string(st.Mode)
					}
					if st.IP != "" {
						state += " " + st.IP
					}
					detail += " • " + state
				} else if st.BackendState != "" {
					detail += " • " + strings.ToLower(st.BackendState)
				}
			}
			item.Detail = detail
		}
		l.updates <- UpdateMsg{ID: item.ID, Item: item}
	}()

	// Security checks
	for _, check := range config.SecurityChecks {
		wg.Add(1)
		go func(c config.SecurityCheck) {
			defer wg.Done()
			result := c.CheckFn()
			l.updates <- UpdateMsg{ID: "security-" + c.Name, Item: Item{
				ID: "security-" + c.Name, Kind: KindSecurity, Name: c.Name,
				Description: c.Description, Loaded: true, Installed: result.Installed,
				Detail: result.Detail, GoodWhen: c.GoodWhen,
			}}
		}(check)
	}

	// Identity checks
	for _, check := range config.IdentityChecks {
		wg.Add(1)
		go func(c config.IdentityCheck) {
			defer wg.Done()
			result := c.CheckFn()
			l.updates <- UpdateMsg{ID: "identity-" + c.Name, Item: Item{
				ID: "identity-" + c.Name, Kind: KindIdentity, Name: c.Name,
				Description: c.Description, Loaded: true, Installed: result.Installed,
				Detail: result.Detail, GoodWhen: c.GoodWhen,
			}}
		}(check)
	}

	// Tool checks
	for _, t := range config.Tools {
		wg.Add(1)
		go func(t config.Tool) {
			defer wg.Done()
			result := t.Check()
			l.updates <- UpdateMsg{ID: "tool-" + t.Name, Item: Item{
				ID: "tool-" + t.Name, Kind: KindTool, Name: t.Name,
				Loaded: true, Installed: result.Installed, Version: result.Version,
				Status: result.Status, Method: t.Method.String(),
			}}
		}(t)
	}

	// Process checks
	for _, check := range config.ProcessChecks {
		wg.Add(1)
		go func(c config.ProcessCheck) {
			defer wg.Done()
			processes := c.CheckFn()
			l.updates <- UpdateMsg{ID: "process-" + c.Name, Item: Item{
				ID: "process-" + c.Name, Kind: KindProcess, Name: c.Name,
				Loaded: true, Available: len(processes) > 0, Processes: processes,
			}}
		}(check)
	}

	// Network checks
	for _, check := range config.NetworkChecks {
		wg.Add(1)
		go func(c config.ResourceCheck) {
			defer wg.Done()
			result := c.CheckFn()
			l.updates <- UpdateMsg{ID: "network-" + c.Name, Item: Item{
				ID: "network-" + c.Name, Kind: KindNetwork, Name: c.Name,
				Loaded: true, Available: result.Available, Value: result.Value, Style: result.Style,
			}}
		}(check)
	}

	// Cache checks
	for _, check := range config.CacheChecks {
		wg.Add(1)
		go func(c config.DiskCheck) {
			defer wg.Done()
			result := c.Check()
			l.updates <- UpdateMsg{ID: "cache-" + c.Name, Item: Item{
				ID: "cache-" + c.Name, Kind: KindCache, Name: c.Name,
				Loaded: true, Available: result.Available, Value: result.Value, Style: result.Style,
			}}
		}(check)
	}

	go func() {
		wg.Wait()
		close(l.updates)
	}()
}

// WaitForUpdate returns a command that waits for the next update
func (l *Loader) WaitForUpdate() tea.Cmd {
	return func() tea.Msg {
		update, ok := <-l.updates
		if !ok {
			return AllLoadedMsg{}
		}
		return update
	}
}

// loadSystemInfo loads system information
func (l *Loader) loadSystemInfo() Item {
	hostname, _ := os.Hostname()
	if idx := strings.Index(hostname, "."); idx > 0 {
		hostname = hostname[:idx]
	}
	if len(hostname) > 20 {
		hostname = hostname[:20]
	}

	osInfo := tool.GetCommandOutput("uname", "-sr")
	arch := tool.GetCommandOutput("uname", "-m")
	user := os.Getenv("USER")
	shell := filepath.Base(os.Getenv("SHELL"))

	return Item{
		ID:     "sysinfo",
		Kind:   KindSystemInfo,
		Loaded: true,
		Detail: osInfo + " " + arch + " • " + hostname + " • " + user + " • " + shell,
	}
}
