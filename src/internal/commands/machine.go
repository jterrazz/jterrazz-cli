package commands

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

type machineProfile string

const (
	machineProfileHomelab     machineProfile = "homelab"
	machineProfileWorkstation machineProfile = "workstation"
	machineProfileVPS         machineProfile = "vps"
)

type machineCheckState string

const (
	machineStateOK      machineCheckState = "ok"
	machineStateWarn    machineCheckState = "warn"
	machineStateFail    machineCheckState = "fail"
	machineStateInfo    machineCheckState = "info"
	machineStateUnknown machineCheckState = "unknown"
)

type machineCheck struct {
	State  machineCheckState
	Scope  string
	Name   string
	Value  string
	Detail string
}

var machineStatusProfile string
var machineUnlockHost string
var machineUnlockUser string

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Inspect machine, homelab, and VPS posture",
	Run: func(cmd *cobra.Command, args []string) {
		runMachineStatus(machineProfile(machineStatusProfile))
	},
}

var machineStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show machine self-healing and service status",
	Run: func(cmd *cobra.Command, args []string) {
		runMachineStatus(machineProfile(machineStatusProfile))
	},
}

var machineUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock a FileVault-protected homelab Mac over preboot SSH",
	Run:   func(cmd *cobra.Command, args []string) { runMachineUnlock() },
}

func init() {
	machineCmd.PersistentFlags().StringVarP(&machineStatusProfile, "profile", "p", string(machineProfileHomelab), "machine profile: homelab, workstation, vps")
	machineUnlockCmd.Flags().StringVar(&machineUnlockHost, "host", "192.168.1.106", "target Mac host or IP")
	machineUnlockCmd.Flags().StringVar(&machineUnlockUser, "user", "jterrazz.agent", "FileVault-enabled macOS user")
	machineCmd.AddCommand(machineStatusCmd, machineUnlockCmd)
	rootCmd.AddCommand(machineCmd)
}

func runMachineUnlock() {
	target := machineUnlockUser + "@" + machineUnlockHost
	// Preboot SSH advertises a different host key than the running OS, and
	// password auth is the only acceptable method (no authorized_keys before FV).
	cmd := exec.Command("ssh",
		"-o", "PreferredAuthentications=password",
		"-o", "PubkeyAuthentication=no",
		"-o", "StrictHostKeyChecking=accept-new",
		target,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	_ = cmd.Run() // a disconnect after the password is the success path
}

func runMachineStatus(profile machineProfile) {
	if !validMachineProfile(profile) {
		print.Warning(fmt.Sprintf("unknown profile %q, using homelab", profile))
		profile = machineProfileHomelab
	}

	print.SectionDivider("MACHINE")
	print.Linef("Profile: %s", profile)
	print.Linef("Host: %s", hostname())
	print.Linef("OS: %s", osSummary())
	print.Empty()

	sections := []struct {
		Title  string
		Checks []machineCheck
	}{
		{"Core", coreMachineChecks(profile)},
		{"OpenClaw", openClawMachineChecks(profile)},
		{"Remote / GUI", remoteGUIMachineChecks(profile)},
		{"Developer / Homelab", devMachineChecks(profile)},
	}

	for _, section := range sections {
		print.Category(section.Title)
		for _, check := range section.Checks {
			printMachineCheck(check)
		}
		print.Empty()
	}

	print.Usage(
		"Profiles: homelab | workstation | vps",
		"Tip: use `j machine status --profile homelab` on always-on personal machines.",
	)
}

func validMachineProfile(profile machineProfile) bool {
	switch profile {
	case machineProfileHomelab, machineProfileWorkstation, machineProfileVPS:
		return true
	default:
		return false
	}
}

func printMachineCheck(c machineCheck) {
	icon := map[machineCheckState]string{
		machineStateOK:      "✅",
		machineStateWarn:    "⚠️ ",
		machineStateFail:    "❌",
		machineStateInfo:    "ℹ️ ",
		machineStateUnknown: "？",
	}[c.State]
	if icon == "" {
		icon = "？"
	}

	value := c.Value
	if value == "" {
		value = "-"
	}
	if c.Detail != "" {
		fmt.Printf("  %s %-12s %-22s %-18s %s\n", icon, c.Scope, c.Name, value, c.Detail)
		return
	}
	fmt.Printf("  %s %-12s %-22s %s\n", icon, c.Scope, c.Name, value)
}

func coreMachineChecks(profile machineProfile) []machineCheck {
	checks := []machineCheck{
		checkFileVault(profile),
		checkSSH(profile),
		checkFirewall(profile),
		checkPower(profile),
		checkAutoBoot(profile),
		checkSleep(profile),
		checkWakeSettings(profile),
	}
	return checks
}

func openClawMachineChecks(profile machineProfile) []machineCheck {
	var checks []machineCheck
	checks = append(checks, checkOpenClawBinary(profile))
	checks = append(checks, checkOpenClawProcess(profile))
	checks = append(checks, checkOpenClawLaunchAgent(profile))
	checks = append(checks, checkOpenClawConfig(profile))
	checks = append(checks, checkOpenClawChannels(profile)...)
	return checks
}

func remoteGUIMachineChecks(profile machineProfile) []machineCheck {
	return []machineCheck{
		checkConsoleOwner(profile),
		checkJumpDesktopConnectApp(profile),
		checkJumpDesktopProcess(profile),
		checkJumpDesktopAudioDrivers(profile),
		checkLockAfterLogin(profile),
		checkAgentInbox(profile),
	}
}

func devMachineChecks(profile machineProfile) []machineCheck {
	return []machineCheck{
		checkDeveloperFolder(profile),
		checkOrbStackInstalled(profile),
		checkOrbStackLoginItem(profile),
		checkOrbStackStatus(profile),
		checkOrbStackProcess(profile),
		checkDocker(profile),
	}
}

func checkFileVault(profile machineProfile) machineCheck {
	out, err := runOutput("fdesetup", "status")
	if err != nil {
		return machineCheck{machineStateUnknown, "all", "FileVault", "unknown", trimOneLine(out)}
	}
	line := trimOneLine(out)
	if strings.Contains(line, "FileVault is On") {
		return machineCheck{machineStateOK, "all", "FileVault", "on", "remote SSH unlock requires macOS 26+ Apple silicon"}
	}
	if profile == machineProfileVPS {
		return machineCheck{machineStateInfo, "vps", "FileVault", line, "not usually applicable on Linux VPS"}
	}
	return machineCheck{machineStateFail, "all", "FileVault", line, "enable disk encryption"}
}

func checkSSH(profile machineProfile) machineCheck {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:22", 750*time.Millisecond)
	if err != nil {
		return machineCheck{machineStateFail, "all", "SSH", "closed", "Remote Login/sshd is not reachable on localhost:22"}
	}
	_ = conn.Close()
	return machineCheck{machineStateOK, "all", "SSH", "listening :22", "pre-FileVault unlock uses password auth, not authorized_keys"}
}

func checkFirewall(profile machineProfile) machineCheck {
	out, err := runOutput("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate")
	if err != nil {
		return machineCheck{machineStateUnknown, "all", "Firewall", "unknown", trimOneLine(out)}
	}
	line := trimOneLine(out)
	if strings.Contains(strings.ToLower(line), "enabled") || strings.Contains(line, "State = 1") {
		return machineCheck{machineStateOK, "all", "Firewall", "enabled", "application firewall is on"}
	}
	state := machineStateWarn
	if profile == machineProfileVPS {
		state = machineStateFail
	}
	return machineCheck{state, "all", "Firewall", "disabled", "consider enabling or relying on a trusted LAN/tailnet boundary"}
}

func checkPower(profile machineProfile) machineCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return machineCheck{machineStateUnknown, "macos", "Auto reboot", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	if settings["autorestart"] == "1" {
		return machineCheck{machineStateOK, "homelab", "Auto reboot", "autorestart=1", "restarts after power loss"}
	}
	return machineCheck{machineStateWarn, "homelab", "Auto reboot", "autorestart=" + settings["autorestart"], "run `sudo pmset -a autorestart 1` if desired"}
}

func checkAutoBoot(profile machineProfile) machineCheck {
	out, err := runOutput("nvram", "-p")
	if err != nil {
		return machineCheck{machineStateUnknown, "macos", "Auto boot", "unknown", trimOneLine(out)}
	}
	settings := parseNVRAM(out)
	value := settings["auto-boot"]
	if value == "" {
		return machineCheck{machineStateUnknown, "macos", "Auto boot", "missing", "NVRAM auto-boot not reported"}
	}
	if value == "true" || value == "%01" || value == "1" {
		return machineCheck{machineStateOK, "homelab", "Auto boot", "auto-boot=" + value, "boots when power is applied/restored"}
	}
	return machineCheck{machineStateWarn, "homelab", "Auto boot", "auto-boot=" + value, "enable for smart-plug recovery after shutdown"}
}

func checkSleep(profile machineProfile) machineCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return machineCheck{machineStateUnknown, "macos", "Sleep", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	parts := []string{"sleep=" + settingValue(settings, "sleep"), "displaysleep=" + settingValue(settings, "displaysleep"), "disksleep=" + settingValue(settings, "disksleep")}
	// Homelab requirement: system + disk never sleep. Display sleep is desirable on
	// a headless mini to avoid burn-in, so any displaysleep value is fine.
	if settings["sleep"] == "0" && settings["disksleep"] == "0" {
		return machineCheck{machineStateOK, "homelab", "Sleep", strings.Join(parts, " "), "system + disk never sleep; display sleep is fine"}
	}
	return machineCheck{machineStateWarn, "homelab", "Sleep", strings.Join(parts, " "), "set sleep=0 and disksleep=0 via `j machine power harden`"}
}

func checkWakeSettings(profile machineProfile) machineCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return machineCheck{machineStateUnknown, "macos", "Wake", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	parts := []string{"womp=" + settingValue(settings, "womp"), "tcpkeepalive=" + settingValue(settings, "tcpkeepalive"), "powernap=" + settingValue(settings, "powernap")}
	if settings["womp"] == "1" && settings["tcpkeepalive"] == "1" {
		return machineCheck{machineStateOK, "homelab", "Wake network", strings.Join(parts, " "), "network wake/keepalive enabled"}
	}
	return machineCheck{machineStateWarn, "homelab", "Wake network", strings.Join(parts, " "), "enable womp/tcpkeepalive for LAN recovery if supported"}
}

func checkOpenClawBinary(profile machineProfile) machineCheck {
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return machineCheck{machineStateFail, "all", "OpenClaw binary", "missing", "install OpenClaw"}
	}
	return machineCheck{machineStateOK, "all", "OpenClaw binary", "installed", path}
}

func checkOpenClawProcess(profile machineProfile) machineCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,ppid,command")
	lines := filterLines(out, "openclaw/dist/index.js", "gateway")
	if len(lines) == 0 {
		return machineCheck{machineStateFail, "all", "OpenClaw runtime", "not running", "gateway process not found"}
	}
	line := strings.Join(lines, " | ")
	state := machineStateOK
	value := "running"
	detail := line
	if !strings.Contains(line, "jterrazz.agent") {
		state = machineStateWarn
		detail = "unexpected owner: " + line
	}
	return machineCheck{state, "all", "OpenClaw runtime", value, detail}
}

// checkOpenClawLaunchAgent expects the user LaunchAgent at
// ~/Library/LaunchAgents/ai.openclaw.gateway.plist (loaded in the Aqua session).
func checkOpenClawLaunchAgent(profile machineProfile) machineCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/LaunchAgents/ai.openclaw.gateway.plist")
	if _, err := os.Stat(path); err != nil {
		return machineCheck{machineStateFail, "homelab", "OpenClaw agent", "missing", "expected " + path}
	}
	out, err := runOutput("launchctl", "print", fmt.Sprintf("gui/%d/ai.openclaw.gateway", os.Getuid()))
	if err != nil {
		return machineCheck{machineStateWarn, "homelab", "OpenClaw agent", "plist only", "agent plist present but not loaded in this gui/* domain"}
	}
	if strings.Contains(out, "state = running") {
		return machineCheck{machineStateOK, "homelab", "OpenClaw agent", "running", path}
	}
	return machineCheck{machineStateWarn, "homelab", "OpenClaw agent", "loaded", "loaded but not running — check gateway logs"}
}

func checkOpenClawConfig(profile machineProfile) machineCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".openclaw/openclaw.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return machineCheck{machineStateFail, "all", "OpenClaw config", "missing", path}
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return machineCheck{machineStateFail, "all", "OpenClaw config", "invalid", err.Error()}
	}
	model := jsonPathString(cfg, "models", "default")
	if model == "" {
		model = jsonPathString(cfg, "agents", "default", "model")
	}
	if model == "" {
		model = "configured"
	}
	return machineCheck{machineStateOK, "all", "OpenClaw config", model, path}
}

func checkOpenClawChannels(profile machineProfile) []machineCheck {
	out, err := runOutput("openclaw", "channels", "status")
	if err != nil {
		return []machineCheck{{machineStateWarn, "all", "Channels", "unknown", trimOneLine(out)}}
	}
	checks := []machineCheck{
		channelCheck(out, "Slack"),
		channelCheck(out, "Telegram"),
		channelCheck(out, "BlueBubbles"),
	}
	return checks
}

func channelCheck(statusOutput, name string) machineCheck {
	for _, line := range strings.Split(statusOutput, "\n") {
		if strings.Contains(line, name+" ") || strings.Contains(line, name+":") {
			state := machineStateOK
			value := "connected"
			if !strings.Contains(line, "connected") {
				state = machineStateWarn
				value = "check"
			}
			if strings.Contains(line, "health:") && !strings.Contains(line, "health:healthy") {
				state = machineStateWarn
				value = "check"
			}
			return machineCheck{state, "all", name, value, strings.TrimSpace(line)}
		}
	}
	return machineCheck{machineStateWarn, "all", name, "not found", "channel not reported by OpenClaw"}
}

func checkConsoleOwner(profile machineProfile) machineCheck {
	out, err := runOutput("stat", "-f", "%Su", "/dev/console")
	if err != nil {
		return machineCheck{machineStateUnknown, "macos", "Console", "unknown", trimOneLine(out)}
	}
	owner := trimOneLine(out)
	if owner == "root" {
		return machineCheck{machineStateOK, "homelab", "Console", "loginwindow", "headless mode; no GUI user logged in"}
	}
	return machineCheck{machineStateInfo, "homelab", "Console", owner, "GUI session is active"}
}

func checkJumpDesktopConnectApp(profile machineProfile) machineCheck {
	path := "/Applications/Jump Desktop Connect.app"
	if _, err := os.Stat(path); err == nil {
		return machineCheck{machineStateOK, "homelab", "Jump Connect app", "installed", path}
	}
	return machineCheck{machineStateWarn, "homelab", "Jump Connect app", "missing", "install Jump Desktop Connect for remote GUI recovery"}
}

func checkJumpDesktopProcess(profile machineProfile) machineCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,command")
	lines := filterLinesCaseInsensitive(out, "JumpConnect")
	if len(lines) == 0 {
		return machineCheck{machineStateWarn, "homelab", "Jump service", "not running", "Jump may need GUI/session or service setup"}
	}
	state := machineStateOK
	value := "running"
	detail := fmt.Sprintf("%d process(es)", len(lines))
	if !strings.Contains(out, "--service") {
		state = machineStateWarn
		detail = "Jump processes found, but service process not obvious"
	}
	return machineCheck{state, "homelab", "Jump service", value, detail}
}

func checkJumpDesktopAudioDrivers(profile machineProfile) machineCheck {
	out, _ := runOutput("ps", "-axo", "command")
	hasOut := strings.Contains(out, "JumpAudio.driver")
	hasIn := strings.Contains(out, "JumpAudioMic.driver")
	if hasOut && hasIn {
		return machineCheck{machineStateOK, "homelab", "Jump audio", "running", "speaker and microphone drivers loaded"}
	}
	if hasOut || hasIn {
		return machineCheck{machineStateWarn, "homelab", "Jump audio", "partial", "one Jump audio driver is missing"}
	}
	return machineCheck{machineStateInfo, "homelab", "Jump audio", "not running", "optional; remote GUI may still work without audio"}
}

func checkLockAfterLogin(profile machineProfile) machineCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/LaunchAgents/ai.jterrazz.lock-after-login.plist")
	if _, err := os.Stat(path); err == nil {
		return machineCheck{machineStateOK, "homelab", "Lock after login", "installed", path}
	}
	return machineCheck{machineStateInfo, "homelab", "Lock after login", "missing", "run `j machine lock-after-login install`"}
}

func checkAgentInbox(profile machineProfile) machineCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/Mobile Documents/com~apple~CloudDocs/Agent Inbox")
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return machineCheck{machineStateOK, "all", "Agent Inbox", "available", path}
	}
	return machineCheck{machineStateWarn, "all", "Agent Inbox", "missing", "shared iCloud folder not found"}
}

func checkDeveloperFolder(profile machineProfile) machineCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Developer")
	entries, err := os.ReadDir(path)
	if err != nil {
		return machineCheck{machineStateWarn, "all", "Developer", "missing", path}
	}
	repos := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(path, entry.Name(), ".git")); err == nil {
			repos++
		}
	}
	return machineCheck{machineStateOK, "all", "Developer", fmt.Sprintf("%d repos", repos), path}
}

func checkOrbStackInstalled(profile machineProfile) machineCheck {
	path := "/Applications/OrbStack.app"
	if _, err := os.Stat(path); err == nil {
		return machineCheck{machineStateOK, "homelab", "OrbStack", "installed", "can run headless via Background LaunchAgent"}
	}
	return machineCheck{machineStateInfo, "homelab", "OrbStack", "missing", "optional for local containers/Kubernetes"}
}

// checkOrbStackLoginItem confirms OrbStack is registered as a Login Item, which is
// how the auto-logged-in Aqua session brings up Docker.
func checkOrbStackLoginItem(profile machineProfile) machineCheck {
	// Querying Login Items needs an Automation entitlement for "System Events". If
	// that's missing, osascript errors instead of prompting in a non-GUI shell, so
	// fall back to an info-level note instead of a noisy warning.
	out, err := runOutput("osascript", "-e", `tell application "System Events" to get the name of every login item`)
	if err != nil {
		return machineCheck{machineStateInfo, "homelab", "OrbStack autostart", "unverified", "grant System Events automation to verify, or check System Settings → General → Login Items"}
	}
	if strings.Contains(strings.ToLower(out), "orbstack") {
		return machineCheck{machineStateOK, "homelab", "OrbStack autostart", "Login Item", "starts via the auto-logged-in Aqua session"}
	}
	return machineCheck{machineStateWarn, "homelab", "OrbStack autostart", "missing", "OrbStack is not in Login Items — open OrbStack → Settings → General → Open at login"}
}

func checkOrbStackStatus(profile machineProfile) machineCheck {
	out, err := runOutput("/usr/local/bin/orbctl", "status")
	if err != nil {
		return machineCheck{machineStateWarn, "homelab", "OrbStack status", "unknown", trimOneLine(out)}
	}
	status := trimOneLine(out)
	if strings.EqualFold(status, "Running") {
		return machineCheck{machineStateOK, "homelab", "OrbStack status", "Running", "Docker/Kubernetes backend is up"}
	}
	return machineCheck{machineStateWarn, "homelab", "OrbStack status", status, "expected to start after FileVault unlock via background agent"}
}

func checkOrbStackProcess(profile machineProfile) machineCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,command")
	lines := filterLinesCaseInsensitive(out, "OrbStack")
	if len(lines) == 0 {
		return machineCheck{machineStateInfo, "homelab", "OrbStack runtime", "not running", "background agent may not have run yet"}
	}
	return machineCheck{machineStateOK, "homelab", "OrbStack runtime", "running", fmt.Sprintf("%d process(es); can run while console owner is root", len(lines))}
}

func checkDocker(profile machineProfile) machineCheck {
	if _, err := exec.LookPath("docker"); err != nil {
		return machineCheck{machineStateInfo, "homelab", "Docker", "missing", "optional unless using containers"}
	}
	out, err := runOutput("docker", "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		return machineCheck{machineStateWarn, "homelab", "Docker", "client only", trimOneLine(out)}
	}
	return machineCheck{machineStateOK, "homelab", "Docker", trimOneLine(out), "daemon reachable"}
}

func runOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func parsePMSet(out string) map[string]string {
	settings := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Sleep On Power Button") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				settings["SleepOnPowerButton"] = fields[len(fields)-1]
			}
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 {
			settings[fields[0]] = fields[1]
		}
	}
	return settings
}

func parseNVRAM(out string) map[string]string {
	settings := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			settings[fields[0]] = fields[1]
		}
	}
	return settings
}

func settingValue(settings map[string]string, key string) string {
	if value := settings[key]; value != "" {
		return value
	}
	return "?"
}

func filterLines(out string, needles ...string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		ok := true
		for _, needle := range needles {
			if !strings.Contains(line, needle) {
				ok = false
				break
			}
		}
		if ok && strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	sort.Strings(lines)
	return lines
}

func filterLinesCaseInsensitive(out string, needle string) []string {
	needle = strings.ToLower(needle)
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(strings.ToLower(line), needle) && strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	sort.Strings(lines)
	return lines
}

func trimOneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return strings.Join(strings.Fields(strings.Split(s, "\n")[0]), " ")
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}

func osSummary() string {
	if runtime.GOOS == "darwin" {
		out, err := runOutput("sw_vers", "-productVersion")
		if err == nil {
			return "macOS " + trimOneLine(out) + " (" + runtime.GOARCH + ")"
		}
	}
	return runtime.GOOS + " " + runtime.GOARCH
}

func jsonPathString(root map[string]any, path ...string) string {
	var current any = root
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}
