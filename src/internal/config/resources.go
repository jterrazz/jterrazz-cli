package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
)

// ResourceCheck represents a system resource check (network, disk, cache)
type ResourceCheck struct {
	Name    string
	CheckFn func() ResourceResult
}

// ProcessInfo represents a single process entry
type ProcessInfo struct {
	Name  string
	Value string // CPU % or Memory
	PID   string
}

// ProcessResult holds multiple processes
type ProcessResult struct {
	Processes []ProcessInfo
	Available bool
}

// RepoInfo describes a single git repository.
type RepoInfo struct {
	Name        string // short name (prefix stripped)
	FullName    string // full directory name
	Branch      string
	ChangeCount int
	Clean       bool
}

// ProjectGroup is a set of repos sharing a project prefix.
type ProjectGroup struct {
	Prefix string
	Repos  []RepoInfo
}

// ResourceResult holds the result of a resource check
type ResourceResult struct {
	Value     string // The value to display (e.g., IP address, size)
	Style     string // "success", "warning", "muted", "special"
	Available bool   // Whether this resource is available/relevant
}

// NetworkChecks is the list of network resource checks
var NetworkChecks = []ResourceCheck{
	{
		Name: "local ip",
		CheckFn: func() ResourceResult {
			out, _ := exec.Command("ipconfig", "getifaddr", "en0").Output()
			ip := strings.TrimSpace(string(out))
			if ip != "" {
				return ResourceResult{Value: ip, Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
	{
		Name: "public ip",
		CheckFn: func() ResourceResult {
			cmd := exec.Command("curl", "-s", "--max-time", "2", "-4", "ifconfig.me")
			out, err := cmd.Output()
			if err == nil {
				ip := strings.TrimSpace(string(out))
				if ip != "" {
					return ResourceResult{Value: ip, Style: "muted", Available: true}
				}
			}
			return ResourceResult{Available: false}
		},
	},
	{
		Name: "vpn",
		CheckFn: func() ResourceResult {
			var vpnNames []string

			// Check system VPN connections (Passepartout, IKEv2, IPSec, etc.)
			out, _ := exec.Command("scutil", "--nc", "list").Output()
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "(Connected)") {
					if idx := strings.LastIndex(line, `"`); idx > 0 {
						start := strings.LastIndex(line[:idx], `"`)
						if start >= 0 && start < idx {
							vpnNames = append(vpnNames, line[start+1:idx])
							continue
						}
					}
					vpnNames = append(vpnNames, "connected")
				}
			}

			// Check if Tailscale has an active exit node
			if st, err := GetTailscaleFullStatus(); err == nil && st.ExitNode != "" {
				vpnNames = append(vpnNames, "Tailscale ("+st.ExitNode+")")
			}

			if len(vpnNames) > 0 {
				return ResourceResult{Value: strings.Join(vpnNames, ", "), Style: "success", Available: true}
			}
			return ResourceResult{Value: "none", Style: "muted", Available: true}
		},
	},
	{
		Name: "dns",
		CheckFn: func() ResourceResult {
			if IsDNSProfileInstalled() {
				return ResourceResult{Value: "Quad9 (encrypted)", Style: "success", Available: true}
			}
			out, _ := exec.Command("scutil", "--dns").Output()
			var servers []string
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "nameserver[") {
					idx := strings.Index(line, "] : ")
					if idx == -1 {
						continue
					}
					server := strings.TrimSpace(line[idx+4:])
					if server == "" || server == "127.0.0.1" || server == "::1" {
						continue
					}
					// Skip IPv6
					if strings.Contains(server, ":") {
						continue
					}
					found := false
					for _, s := range servers {
						if s == server {
							found = true
							break
						}
					}
					if !found && len(servers) < 2 {
						servers = append(servers, server)
					}
				}
			}
			if len(servers) > 0 {
				return ResourceResult{Value: strings.Join(servers, ", "), Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
}

// DiskCheck represents a disk usage check
type DiskCheck struct {
	Name    string
	Path    string                // Path to check (supports ~ expansion)
	Style   string                // Default style for this check
	CheckFn func() ResourceResult // Custom check (overrides Path)
}

// CacheChecks shows disk usage grouped by domain
var CacheChecks = []DiskCheck{
	{
		Name: "docker",
		CheckFn: func() ResourceResult {
			if !CommandExists("docker") {
				return ResourceResult{Available: false}
			}
			out, _ := exec.Command("docker", "system", "df", "--format", "{{.Size}}").Output()
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			var total int64
			for _, line := range lines {
				total += parseDockerSize(strings.TrimSpace(line))
			}
			if total > 0 {
				return ResourceResult{Value: tool.FormatBytes(total), Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
	{
		Name: "xcode",
		CheckFn: func() ResourceResult {
			paths := []string{
				"~/Library/Developer/Xcode/DerivedData",
				"~/Library/Developer/Xcode/Archives",
				"~/Library/Developer/Xcode/iOS DeviceSupport",
			}
			var total int64
			for _, p := range paths {
				total += GetDirSize(expandHome(p))
			}
			if total > 0 {
				return ResourceResult{Value: tool.FormatBytes(total), Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
	{Name: "homebrew", Path: "~/Library/Caches/Homebrew", Style: "muted"},
	{
		Name: "packages",
		CheckFn: func() ResourceResult {
			paths := []string{
				"~/.npm",
				"~/Library/pnpm",
				"~/.bun/install/cache",
				"~/Library/Caches/Yarn",
				"~/Library/Caches/CocoaPods",
				"~/go/pkg/mod",
				"~/.gradle/caches",
			}
			var total int64
			for _, p := range paths {
				total += GetDirSize(expandHome(p))
			}
			if total > 0 {
				return ResourceResult{Value: tool.FormatBytes(total), Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
	{
		Name: "multipass",
		CheckFn: func() ResourceResult {
			if !CommandExists("multipass") {
				return ResourceResult{Available: false}
			}
			path := expandHome("~/Library/Application Support/multipassd")
			if size := GetDirSize(path); size > 0 {
				return ResourceResult{Value: tool.FormatBytes(size), Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
	{
		Name: "logs",
		CheckFn: func() ResourceResult {
			total := GetDirSize("/var/log") + GetDirSize(expandHome("~/Library/Logs"))
			if total > 0 {
				return ResourceResult{Value: tool.FormatBytes(total), Style: "muted", Available: true}
			}
			return ResourceResult{Available: false}
		},
	},
	{Name: "trash", Path: "~/.Trash", Style: "muted"},
}

// parseDockerSize parses Docker size strings like "1.973GB", "469.4kB" into bytes
func parseDockerSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" {
		return 0
	}
	multipliers := map[string]float64{
		"B": 1, "kB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12,
	}
	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			var val float64
			fmt.Sscanf(numStr, "%f", &val)
			return int64(val * mult)
		}
	}
	return 0
}

// CheckDisk checks a disk path and returns the result
func (d DiskCheck) Check() ResourceResult {
	if d.CheckFn != nil {
		return d.CheckFn()
	}

	path := expandHome(d.Path)
	if size := GetDirSize(path); size > 0 {
		return ResourceResult{Value: tool.FormatBytes(size), Style: d.Style, Available: true}
	}
	return ResourceResult{Available: false}
}

// expandHome expands ~ to the user's home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(os.Getenv("HOME"), path[2:])
	}
	return path
}

// ProcessCheck represents a process resource check
type ProcessCheck struct {
	Name    string
	CheckFn func() []ProcessInfo
}

// ProcessChecks defines the process monitoring checks
var ProcessChecks = []ProcessCheck{
	{
		Name: "CPU",
		CheckFn: func() []ProcessInfo {
			out, err := exec.Command("ps", "-arcwwwxo", "pid,%cpu,comm").Output()
			if err != nil {
				return nil
			}
			return parseCPUOutput(out)
		},
	},
	{
		Name: "Memory",
		CheckFn: func() []ProcessInfo {
			out, err := exec.Command("ps", "-amcwwwxo", "pid,rss,comm").Output()
			if err != nil {
				return nil
			}
			return parseMemoryOutput(out)
		},
	},
	{
		Name: "Ports",
		CheckFn: func() []ProcessInfo {
			out, _ := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n", "-Fcn").Output()
			return parseListeningPortsFcn(out)
		},
	},
	{
		Name: "Containers",
		CheckFn: func() []ProcessInfo {
			if !CommandExists("docker") {
				return nil
			}
			out, err := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Status}}\t{{.Ports}}").Output()
			if err != nil {
				return nil
			}
			return parseDockerContainers(out)
		},
	},
	{
		Name: "Uptime",
		CheckFn: func() []ProcessInfo {
			return getUptimeInfo()
		},
	},
}

// System processes to hide from port listing
var systemPortProcesses = map[string]bool{
	"rapportd":      true, // macOS Rapport daemon
	"ControlCenter": true, // macOS Control Center (AirPlay)
	"IPNExtension":  true, // Tailscale (already shown in Network)
	"mDNSResponder": true, // macOS DNS
	"launchd":       true, // macOS init
	"systemd":       true, // Linux init
}

// parseListeningPortsFcn parses lsof -Fcn output into port → process entries
func parseListeningPortsFcn(out []byte) []ProcessInfo {
	lines := strings.Split(string(out), "\n")

	type portEntry struct {
		port    string
		portNum int
		cmd     string
	}

	var currentCmd string
	seen := make(map[string]bool)
	var entries []portEntry

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'c': // command name
			currentCmd = line[1:]
		case 'n': // network address (e.g. "*:8080" or "127.0.0.1:3000")
			addr := line[1:]
			if currentCmd == "" {
				continue
			}
			// Skip system processes
			if systemPortProcesses[currentCmd] {
				continue
			}
			// Extract port
			idx := strings.LastIndex(addr, ":")
			if idx < 0 {
				continue
			}
			port := addr[idx+1:]
			key := port + "/" + currentCmd
			if seen[key] {
				continue
			}
			seen[key] = true

			num, _ := strconv.Atoi(port)
			entries = append(entries, portEntry{port: port, portNum: num, cmd: currentCmd})
		}
	}

	// Sort by port number
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].portNum < entries[j].portNum
	})

	var result []ProcessInfo
	for _, e := range entries {
		result = append(result, ProcessInfo{
			Name:  e.cmd,
			Value: ":" + e.port,
		})
	}
	return result
}

// parseCPUOutput parses ps CPU output into ProcessInfo slice
func parseCPUOutput(out []byte) []ProcessInfo {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var processes []ProcessInfo

	// Skip header, take top 5
	for i := 1; i < len(lines) && len(processes) < 5; i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 3 {
			continue
		}
		pid := fields[0]
		cpuPercent := fields[1]
		name := strings.Join(fields[2:], " ")

		processes = append(processes, ProcessInfo{
			Name:  name,
			Value: cpuPercent + "%",
			PID:   pid,
		})
	}

	return processes
}

// parseMemoryOutput parses ps memory output (RSS in KB) into ProcessInfo slice
func parseMemoryOutput(out []byte) []ProcessInfo {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var processes []ProcessInfo

	// Skip header, take top 5
	for i := 1; i < len(lines) && len(processes) < 5; i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 3 {
			continue
		}
		pid := fields[0]
		rssKB := fields[1]
		name := strings.Join(fields[2:], " ")

		// Convert RSS from KB to human readable format
		var formatted string
		if kb, err := strconv.ParseInt(rssKB, 10, 64); err == nil {
			mb := kb / 1024
			if mb >= 1024 {
				formatted = fmt.Sprintf("%.1fG", float64(mb)/1024)
			} else {
				formatted = fmt.Sprintf("%dM", mb)
			}
		} else {
			formatted = rssKB + "K"
		}

		processes = append(processes, ProcessInfo{
			Name:  name,
			Value: formatted,
			PID:   pid,
		})
	}

	return processes
}

// parseDockerContainers parses docker ps output into ProcessInfo entries
func parseDockerContainers(out []byte) []ProcessInfo {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []ProcessInfo
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		statusStr := parts[1]

		short := statusStr
		if strings.HasPrefix(statusStr, "Up ") {
			short = shortenUptime(strings.TrimPrefix(statusStr, "Up "))
		}

		portInfo := ""
		if len(parts) == 3 && parts[2] != "" {
			portInfo = extractHostPorts(parts[2])
		}

		value := short
		if portInfo != "" {
			value = portInfo + " " + short
		}

		result = append(result, ProcessInfo{Name: name, Value: value})
	}
	return result
}

// extractHostPorts extracts unique host ports from docker port mappings
func extractHostPorts(ports string) string {
	var hostPorts []string
	seen := make(map[string]bool)
	for _, mapping := range strings.Split(ports, ", ") {
		if idx := strings.Index(mapping, "->"); idx > 0 {
			hostPart := mapping[:idx]
			if colonIdx := strings.LastIndex(hostPart, ":"); colonIdx >= 0 {
				port := hostPart[colonIdx+1:]
				if !seen[port] {
					seen[port] = true
					hostPorts = append(hostPorts, ":"+port)
				}
			}
		}
	}
	if len(hostPorts) == 0 {
		return ""
	}
	return strings.Join(hostPorts, ",")
}

// shortenUptime shortens "3 hours" → "3h", "2 days" → "2d", etc.
func shortenUptime(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, " ("); idx > 0 {
		s = s[:idx]
	}
	replacer := strings.NewReplacer(
		" seconds", "s", " second", "s",
		" minutes", "m", " minute", "m",
		" hours", "h", " hour", "h",
		" days", "d", " day", "d",
		" weeks", "w", " week", "w",
		" months", "mo", " month", "mo",
		"About ", "~",
	)
	return replacer.Replace(s)
}

// ScanAllRepos scans all git repos in ~/Developer and groups them by project prefix.
func ScanAllRepos() []ProjectGroup {
	devDir := os.Getenv("HOME") + "/Developer"
	entries, err := os.ReadDir(devDir)
	if err != nil {
		return nil
	}

	// Collect all repos
	var repos []RepoInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoPath := devDir + "/" + entry.Name()
		if _, err := os.Stat(repoPath + "/.git"); err != nil {
			continue
		}

		branchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
		branchOut, _ := branchCmd.Output()
		branch := strings.TrimSpace(string(branchOut))
		if branch == "" {
			branch = "?"
		}

		cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
		out, _ := cmd.Output()
		status := strings.TrimSpace(string(out))
		changeCount := 0
		if status != "" {
			changeCount = len(strings.Split(status, "\n"))
		}

		repos = append(repos, RepoInfo{
			FullName:    entry.Name(),
			Branch:      branch,
			ChangeCount: changeCount,
			Clean:       changeCount == 0,
		})
	}

	// Group by prefix (everything before the first "-")
	groupMap := make(map[string][]RepoInfo)
	var groupOrder []string
	for _, repo := range repos {
		prefix := repo.FullName
		if idx := strings.Index(repo.FullName, "-"); idx > 0 {
			prefix = repo.FullName[:idx]
			repo.Name = repo.FullName[idx+1:]
		} else {
			repo.Name = repo.FullName
		}
		if _, exists := groupMap[prefix]; !exists {
			groupOrder = append(groupOrder, prefix)
		}
		groupMap[prefix] = append(groupMap[prefix], repo)
	}

	sort.Strings(groupOrder)

	var result []ProjectGroup
	for _, prefix := range groupOrder {
		group := ProjectGroup{Prefix: prefix, Repos: groupMap[prefix]}
		sort.Slice(group.Repos, func(i, j int) bool {
			return group.Repos[i].Name < group.Repos[j].Name
		})
		result = append(result, group)
	}
	return result
}

// getUptimeInfo returns system uptime, load average, and battery info
func getUptimeInfo() []ProcessInfo {
	var result []ProcessInfo

	uptimeOut, _ := exec.Command("uptime").Output()
	uptimeStr := strings.TrimSpace(string(uptimeOut))
	if uptimeStr != "" {
		if idx := strings.Index(uptimeStr, "up "); idx >= 0 {
			rest := uptimeStr[idx+3:]
			if uIdx := strings.Index(rest, " user"); uIdx > 0 {
				upPart := rest[:uIdx]
				if cIdx := strings.LastIndex(upPart, ","); cIdx > 0 {
					upPart = strings.TrimSpace(upPart[:cIdx])
				}
				result = append(result, ProcessInfo{Name: "uptime", Value: strings.TrimRight(upPart, ",")})
			}
		}
		if idx := strings.Index(uptimeStr, "load averages: "); idx >= 0 {
			loads := strings.TrimSpace(uptimeStr[idx+len("load averages: "):])
			result = append(result, ProcessInfo{Name: "load", Value: loads})
		} else if idx := strings.Index(uptimeStr, "load average: "); idx >= 0 {
			loads := strings.TrimSpace(uptimeStr[idx+len("load average: "):])
			result = append(result, ProcessInfo{Name: "load", Value: loads})
		}
	}

	battOut, err := exec.Command("pmset", "-g", "batt").Output()
	if err == nil {
		for _, line := range strings.Split(string(battOut), "\n") {
			if strings.Contains(line, "%") {
				line = strings.TrimSpace(line)
				if tabIdx := strings.Index(line, "\t"); tabIdx >= 0 {
					info := strings.TrimSpace(line[tabIdx:])
					parts := strings.SplitN(info, ";", 3)
					if len(parts) >= 2 {
						pct := strings.TrimSpace(parts[0])
						state := strings.TrimSpace(parts[1])
						result = append(result, ProcessInfo{Name: "battery", Value: pct + " " + state})
					}
				}
			}
		}
	}

	return result
}
