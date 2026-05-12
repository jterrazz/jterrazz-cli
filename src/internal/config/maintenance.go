package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MaintenanceCheck is the same shape as IdentityCheck and SecurityCheck —
// repeated locally instead of generalised because each domain wants its own
// loader plumbing and I want clear blast radius when one probe goes wrong.
type MaintenanceCheck struct {
	Name        string
	Description string
	CheckFn     func() CheckResult
	GoodWhen    bool
}

// MaintenanceChecks groups passive system-state probes that don't fit
// Identity, Security, or per-Script categories. Each Detail is short
// enough to fit a typical Configuration box's right column.
var MaintenanceChecks = []MaintenanceCheck{
	{
		Name:        "macOS",
		Description: "Pending macOS software updates",
		CheckFn:     checkMacOSUpdates,
		GoodWhen:    false, // healthy when no updates pending
	},
	{
		Name:        "brew",
		Description: "Outdated Homebrew formulae and casks",
		CheckFn:     checkBrewOutdated,
		GoodWhen:    false, // healthy when no outdated packages
	},
}

// /Library/Updates/index.plist is touched by softwareupdate when an update
// has been downloaded and is awaiting install. Reading it directly avoids
// a `softwareupdate -l` network round-trip.
func checkMacOSUpdates() CheckResult {
	info, err := os.Stat("/Library/Updates/index.plist")
	if err != nil {
		return CheckResult{Installed: false, Detail: "up to date"}
	}
	age := humanizeAge(time.Since(info.ModTime()))
	return CheckResult{Installed: true, Detail: "pending · " + age + " old"}
}

// `brew outdated --quiet` lists outdated package names, one per line, on
// stdout. Stderr noise (permission warnings, JSON cache messages) is
// discarded — we only care about the count.
func checkBrewOutdated() CheckResult {
	if _, err := exec.LookPath("brew"); err != nil {
		return CheckResult{}
	}
	cmd := exec.Command("brew", "outdated", "--quiet")
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Detail: "unknown"}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return CheckResult{Installed: false, Detail: "all current"}
	}
	return CheckResult{Installed: true, Detail: fmt.Sprintf("%d outdated", len(lines))}
}

func humanizeAge(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
}

// DaemonCheck represents a single user-installed LaunchAgent we care about.
// Discovered dynamically — anything in ~/Library/LaunchAgents matching
// daemonNamePrefixes is included.
type DaemonCheck struct {
	Label string // e.g. "ai.jterrazz.lock-after-login"
	Path  string // full plist path
}

// daemonNamePrefixes filters LaunchAgents to just our own. macOS ships
// dozens of agents and third-party apps add more; we don't want to surface
// Google Updater or Adobe in the j status view.
var daemonNamePrefixes = []string{"ai.jterrazz.", "com.jterrazz.", "ai.openclaw.", "ai.hermes."}

// DiscoverDaemons enumerates ~/Library/LaunchAgents for plists matching
// daemonNamePrefixes. Sorted by label so the rendered list is stable.
func DiscoverDaemons() []DaemonCheck {
	dir := filepath.Join(os.Getenv("HOME"), "Library/LaunchAgents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []DaemonCheck
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".plist") {
			continue
		}
		label := strings.TrimSuffix(name, ".plist")
		matched := false
		for _, p := range daemonNamePrefixes {
			if strings.HasPrefix(label, p) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		out = append(out, DaemonCheck{Label: label, Path: filepath.Join(dir, name)})
	}
	return out
}

// CheckDaemonState returns Installed=true when the agent is loaded into
// launchd. Detail describes its state: "running PID …", "idle", or
// "exit N" when launchd recorded a non-zero exit from the last invocation.
//
// We parse `launchctl list` output rather than `launchctl print` because
// list is cheap (one fork) and gives all three datapoints we need: PID,
// last exit code, and label.
func CheckDaemonState(label string) CheckResult {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return CheckResult{Detail: "launchctl error"}
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[2] != label {
			continue
		}
		pid, exitCode := fields[0], fields[1]
		switch {
		case pid != "-":
			return CheckResult{Installed: true, Detail: "running · pid " + pid}
		case exitCode != "0" && exitCode != "-":
			return CheckResult{Installed: true, Detail: "idle · last exit " + exitCode}
		default:
			return CheckResult{Installed: true, Detail: "idle"}
		}
	}
	return CheckResult{Installed: false, Detail: "not loaded"}
}
