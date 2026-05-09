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

type hostProfile string

const (
	hostProfileHomelab     hostProfile = "homelab"
	hostProfileWorkstation hostProfile = "workstation"
	hostProfileVPS         hostProfile = "vps"
)

type hostCheckState string

const (
	hostStateOK      hostCheckState = "ok"
	hostStateWarn    hostCheckState = "warn"
	hostStateFail    hostCheckState = "fail"
	hostStateInfo    hostCheckState = "info"
	hostStateUnknown hostCheckState = "unknown"
)

type hostCheck struct {
	State  hostCheckState
	Scope  string
	Name   string
	Value  string
	Detail string
}

var hostStatusProfile string
var hostUnlockHost string
var hostUnlockUser string
var hostUnlockDryRun bool

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Inspect machine, homelab, and VPS posture",
	Run: func(cmd *cobra.Command, args []string) {
		runHostStatus(hostProfile(hostStatusProfile))
	},
}

var hostStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show host self-healing and service status",
	Run: func(cmd *cobra.Command, args []string) {
		runHostStatus(hostProfile(hostStatusProfile))
	},
}

var hostUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock a FileVault-protected homelab Mac over SSH",
	Long: strings.TrimSpace(`Unlock a FileVault-protected homelab Mac from the preboot SSH server.

This intentionally disables public-key authentication so macOS asks for the FileVault account password. On success, preboot SSH disconnects and macOS continues booting.`),
	Run: func(cmd *cobra.Command, args []string) {
		runHostUnlock()
	},
}

func init() {
	hostCmd.PersistentFlags().StringVarP(&hostStatusProfile, "profile", "p", string(hostProfileHomelab), "host profile: homelab, workstation, vps")
	hostUnlockCmd.Flags().StringVar(&hostUnlockHost, "host", "192.168.1.106", "target Mac host or IP")
	hostUnlockCmd.Flags().StringVar(&hostUnlockUser, "user", "jterrazz.agent", "FileVault-enabled macOS user")
	hostUnlockCmd.Flags().BoolVar(&hostUnlockDryRun, "dry-run", false, "print the SSH command without running it")
	hostCmd.AddCommand(hostStatusCmd, hostUnlockCmd)
	rootCmd.AddCommand(hostCmd)
}

func runHostUnlock() {
	target := hostUnlockUser + "@" + hostUnlockHost
	args := []string{
		"-o", "PreferredAuthentications=password",
		"-o", "PubkeyAuthentication=no",
		target,
	}

	print.SectionDivider("FILEVAULT UNLOCK")
	print.Linef("Target: %s", target)
	print.Dim("Enter the macOS/FileVault password when SSH prompts. A disconnect after success is expected: the Mac continues booting.")
	print.Empty()
	print.Dim("ssh " + strings.Join(args, " "))
	print.Empty()

	if hostUnlockDryRun {
		return
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		print.Warning("SSH exited: " + err.Error())
		print.Dim("If the Mac was at FileVault preboot, a disconnect can still mean the unlock succeeded. Wait 30-90s, then run `j host status --profile homelab` or try normal SSH.")
	}
}

func runHostStatus(profile hostProfile) {
	if !validHostProfile(profile) {
		print.Warning(fmt.Sprintf("unknown profile %q, using homelab", profile))
		profile = hostProfileHomelab
	}

	print.SectionDivider("HOST")
	print.Linef("Profile: %s", profile)
	print.Linef("Host: %s", hostname())
	print.Linef("OS: %s", osSummary())
	print.Empty()

	sections := []struct {
		Title  string
		Checks []hostCheck
	}{
		{"Core", coreHostChecks(profile)},
		{"OpenClaw", openClawHostChecks(profile)},
		{"Remote / GUI", remoteGUIHostChecks(profile)},
		{"Developer / Homelab", devHostChecks(profile)},
	}

	for _, section := range sections {
		print.Category(section.Title)
		for _, check := range section.Checks {
			printHostCheck(check)
		}
		print.Empty()
	}

	print.Usage(
		"Profiles: homelab | workstation | vps",
		"Tip: use `j host status --profile homelab` on always-on personal machines.",
	)
}

func validHostProfile(profile hostProfile) bool {
	switch profile {
	case hostProfileHomelab, hostProfileWorkstation, hostProfileVPS:
		return true
	default:
		return false
	}
}

func printHostCheck(c hostCheck) {
	icon := map[hostCheckState]string{
		hostStateOK:      "✅",
		hostStateWarn:    "⚠️ ",
		hostStateFail:    "❌",
		hostStateInfo:    "ℹ️ ",
		hostStateUnknown: "？",
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

func coreHostChecks(profile hostProfile) []hostCheck {
	checks := []hostCheck{
		checkFileVault(profile),
		checkSSH(profile),
		checkFirewall(profile),
		checkPower(profile),
		checkAutoBoot(profile),
		checkSleep(profile),
		checkWakeSettings(profile),
		checkPowerButtonSleep(profile),
	}
	return checks
}

func openClawHostChecks(profile hostProfile) []hostCheck {
	var checks []hostCheck
	checks = append(checks, checkOpenClawBinary(profile))
	checks = append(checks, checkOpenClawProcess(profile))
	checks = append(checks, checkOpenClawLaunchDaemon(profile))
	checks = append(checks, checkOpenClawLaunchAgent(profile))
	checks = append(checks, checkOpenClawConfig(profile))
	checks = append(checks, checkOpenClawChannels(profile)...)
	return checks
}

func remoteGUIHostChecks(profile hostProfile) []hostCheck {
	return []hostCheck{
		checkConsoleOwner(profile),
		checkScreenSharing(profile),
		checkJumpDesktopConnectApp(profile),
		checkJumpDesktopClientApp(profile),
		checkJumpDesktopProcess(profile),
		checkJumpDesktopAudioDrivers(profile),
		checkLockAfterLogin(profile),
		checkAgentInbox(profile),
	}
}

func devHostChecks(profile hostProfile) []hostCheck {
	return []hostCheck{
		checkDeveloperFolder(profile),
		checkOrbStackInstalled(profile),
		checkOrbStackBackgroundAgent(profile),
		checkOrbStackObsoleteDaemon(profile),
		checkOrbStackStatus(profile),
		checkOrbStackProcess(profile),
		checkDocker(profile),
	}
}

func checkFileVault(profile hostProfile) hostCheck {
	out, err := runOutput("fdesetup", "status")
	if err != nil {
		return hostCheck{hostStateUnknown, "all", "FileVault", "unknown", trimOneLine(out)}
	}
	line := trimOneLine(out)
	if strings.Contains(line, "FileVault is On") {
		return hostCheck{hostStateOK, "all", "FileVault", "on", "remote SSH unlock requires macOS 26+ Apple silicon"}
	}
	if profile == hostProfileVPS {
		return hostCheck{hostStateInfo, "vps", "FileVault", line, "not usually applicable on Linux VPS"}
	}
	return hostCheck{hostStateFail, "all", "FileVault", line, "enable disk encryption"}
}

func checkSSH(profile hostProfile) hostCheck {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:22", 750*time.Millisecond)
	if err != nil {
		return hostCheck{hostStateFail, "all", "SSH", "closed", "Remote Login/sshd is not reachable on localhost:22"}
	}
	_ = conn.Close()
	return hostCheck{hostStateOK, "all", "SSH", "listening :22", "pre-FileVault unlock uses password auth, not authorized_keys"}
}

func checkFirewall(profile hostProfile) hostCheck {
	out, err := runOutput("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate")
	if err != nil {
		return hostCheck{hostStateUnknown, "all", "Firewall", "unknown", trimOneLine(out)}
	}
	line := trimOneLine(out)
	if strings.Contains(strings.ToLower(line), "enabled") || strings.Contains(line, "State = 1") {
		return hostCheck{hostStateOK, "all", "Firewall", "enabled", "application firewall is on"}
	}
	state := hostStateWarn
	if profile == hostProfileVPS {
		state = hostStateFail
	}
	return hostCheck{state, "all", "Firewall", "disabled", "consider enabling or relying on a trusted LAN/tailnet boundary"}
}

func checkPower(profile hostProfile) hostCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return hostCheck{hostStateUnknown, "macos", "Auto reboot", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	if settings["autorestart"] == "1" {
		return hostCheck{hostStateOK, "homelab", "Auto reboot", "autorestart=1", "restarts after power loss"}
	}
	return hostCheck{hostStateWarn, "homelab", "Auto reboot", "autorestart=" + settings["autorestart"], "run `sudo pmset -a autorestart 1` if desired"}
}

func checkAutoBoot(profile hostProfile) hostCheck {
	out, err := runOutput("nvram", "-p")
	if err != nil {
		return hostCheck{hostStateUnknown, "macos", "Auto boot", "unknown", trimOneLine(out)}
	}
	settings := parseNVRAM(out)
	value := settings["auto-boot"]
	if value == "" {
		return hostCheck{hostStateUnknown, "macos", "Auto boot", "missing", "NVRAM auto-boot not reported"}
	}
	if value == "true" || value == "%01" || value == "1" {
		return hostCheck{hostStateOK, "homelab", "Auto boot", "auto-boot=" + value, "boots when power is applied/restored"}
	}
	return hostCheck{hostStateWarn, "homelab", "Auto boot", "auto-boot=" + value, "enable for smart-plug recovery after shutdown"}
}

func checkSleep(profile hostProfile) hostCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return hostCheck{hostStateUnknown, "macos", "Sleep", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	parts := []string{"sleep=" + settingValue(settings, "sleep"), "displaysleep=" + settingValue(settings, "displaysleep"), "disksleep=" + settingValue(settings, "disksleep")}
	if settings["sleep"] == "0" {
		state := hostStateOK
		detail := "system sleep disabled"
		if settings["displaysleep"] != "0" || settings["disksleep"] != "0" {
			state = hostStateWarn
			detail = "system stays awake; display/disk sleep still configured"
		}
		return hostCheck{state, "homelab", "Sleep", strings.Join(parts, " "), detail}
	}
	return hostCheck{hostStateWarn, "homelab", "Sleep", strings.Join(parts, " "), "disable system sleep for always-on hosts"}
}

func checkWakeSettings(profile hostProfile) hostCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return hostCheck{hostStateUnknown, "macos", "Wake", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	parts := []string{"womp=" + settingValue(settings, "womp"), "tcpkeepalive=" + settingValue(settings, "tcpkeepalive"), "powernap=" + settingValue(settings, "powernap")}
	if settings["womp"] == "1" && settings["tcpkeepalive"] == "1" {
		return hostCheck{hostStateOK, "homelab", "Wake network", strings.Join(parts, " "), "network wake/keepalive enabled"}
	}
	return hostCheck{hostStateWarn, "homelab", "Wake network", strings.Join(parts, " "), "enable womp/tcpkeepalive for LAN recovery if supported"}
}

func checkPowerButtonSleep(profile hostProfile) hostCheck {
	out, err := runOutput("pmset", "-g", "custom")
	if err != nil {
		return hostCheck{hostStateUnknown, "macos", "Power button", "unknown", trimOneLine(out)}
	}
	settings := parsePMSet(out)
	value := settingValue(settings, "SleepOnPowerButton")
	if value == "0" {
		return hostCheck{hostStateOK, "homelab", "Power button", "sleep=0", "power button sleep shortcut disabled"}
	}
	if value == "1" {
		return hostCheck{hostStateWarn, "homelab", "Power button", "sleep=1", "physical power button can sleep the Mac"}
	}
	return hostCheck{hostStateUnknown, "macos", "Power button", "unknown", "Sleep On Power Button not reported"}
}

func checkOpenClawBinary(profile hostProfile) hostCheck {
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return hostCheck{hostStateFail, "all", "OpenClaw binary", "missing", "install OpenClaw"}
	}
	state := hostStateOK
	detail := path
	if strings.Contains(path, "/.nvm/") {
		state = hostStateWarn
		detail = "uses nvm Node path; system Node is more robust for daemons"
	}
	return hostCheck{state, "all", "OpenClaw binary", "installed", detail}
}

func checkOpenClawProcess(profile hostProfile) hostCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,ppid,command")
	lines := filterLines(out, "openclaw/dist/index.js", "gateway")
	if len(lines) == 0 {
		return hostCheck{hostStateFail, "all", "OpenClaw runtime", "not running", "gateway process not found"}
	}
	line := strings.Join(lines, " | ")
	state := hostStateOK
	value := "running"
	detail := line
	if !strings.Contains(line, "jterrazz.agent") {
		state = hostStateWarn
		detail = "unexpected owner: " + line
	}
	return hostCheck{state, "all", "OpenClaw runtime", value, detail}
}

// checkOpenClawLaunchDaemon flags the legacy /Library/LaunchDaemons plist as a
// leftover. The new model runs OpenClaw as a user LaunchAgent in the auto-logged-in
// Aqua session — see checkOpenClawLaunchAgent.
func checkOpenClawLaunchDaemon(profile hostProfile) hostCheck {
	if _, err := os.Stat("/Library/LaunchDaemons/ai.openclaw.gateway.plist"); err == nil {
		return hostCheck{hostStateWarn, "homelab", "OpenClaw daemon", "leftover", "legacy /Library/LaunchDaemons plist; remove now that OpenClaw runs as a user LaunchAgent"}
	}
	out, err := runOutput("launchctl", "print", "system/ai.openclaw.gateway")
	if err == nil && strings.Contains(out, "state = running") {
		return hostCheck{hostStateWarn, "homelab", "OpenClaw daemon", "running", "system LaunchDaemon still loaded; bootout and remove the plist"}
	}
	return hostCheck{hostStateOK, "homelab", "OpenClaw daemon", "absent", "no leftover LaunchDaemon — gateway runs as a user LaunchAgent now"}
}

// checkOpenClawLaunchAgent expects the user LaunchAgent at
// ~/Library/LaunchAgents/ai.openclaw.gateway.plist (loaded in the Aqua session).
func checkOpenClawLaunchAgent(profile hostProfile) hostCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/LaunchAgents/ai.openclaw.gateway.plist")
	if _, err := os.Stat(path); err != nil {
		return hostCheck{hostStateFail, "homelab", "OpenClaw agent", "missing", "expected " + path}
	}
	out, err := runOutput("launchctl", "print", fmt.Sprintf("gui/%d/ai.openclaw.gateway", os.Getuid()))
	if err != nil {
		return hostCheck{hostStateWarn, "homelab", "OpenClaw agent", "plist only", "agent plist present but not loaded in this gui/* domain"}
	}
	if strings.Contains(out, "state = running") {
		return hostCheck{hostStateOK, "homelab", "OpenClaw agent", "running", path}
	}
	return hostCheck{hostStateWarn, "homelab", "OpenClaw agent", "loaded", "loaded but not running — check gateway logs"}
}

func checkOpenClawConfig(profile hostProfile) hostCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".openclaw/openclaw.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return hostCheck{hostStateFail, "all", "OpenClaw config", "missing", path}
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return hostCheck{hostStateFail, "all", "OpenClaw config", "invalid", err.Error()}
	}
	model := jsonPathString(cfg, "models", "default")
	if model == "" {
		model = jsonPathString(cfg, "agents", "default", "model")
	}
	if model == "" {
		model = "configured"
	}
	return hostCheck{hostStateOK, "all", "OpenClaw config", model, path}
}

func checkOpenClawChannels(profile hostProfile) []hostCheck {
	out, err := runOutput("openclaw", "channels", "status")
	if err != nil {
		return []hostCheck{{hostStateWarn, "all", "Channels", "unknown", trimOneLine(out)}}
	}
	checks := []hostCheck{
		channelCheck(out, "Slack"),
		channelCheck(out, "Telegram"),
	}
	return checks
}

func channelCheck(statusOutput, name string) hostCheck {
	for _, line := range strings.Split(statusOutput, "\n") {
		if strings.Contains(line, name+" ") || strings.Contains(line, name+":") {
			state := hostStateOK
			value := "connected"
			if !strings.Contains(line, "connected") {
				state = hostStateWarn
				value = "check"
			}
			if strings.Contains(line, "health:") && !strings.Contains(line, "health:healthy") {
				state = hostStateWarn
				value = "check"
			}
			return hostCheck{state, "all", name, value, strings.TrimSpace(line)}
		}
	}
	return hostCheck{hostStateWarn, "all", name, "not found", "channel not reported by OpenClaw"}
}

func checkConsoleOwner(profile hostProfile) hostCheck {
	out, err := runOutput("stat", "-f", "%Su", "/dev/console")
	if err != nil {
		return hostCheck{hostStateUnknown, "macos", "Console", "unknown", trimOneLine(out)}
	}
	owner := trimOneLine(out)
	if owner == "root" {
		return hostCheck{hostStateOK, "homelab", "Console", "loginwindow", "headless mode; no GUI user logged in"}
	}
	return hostCheck{hostStateInfo, "homelab", "Console", owner, "GUI session is active"}
}

func checkScreenSharing(profile hostProfile) hostCheck {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:5900", 750*time.Millisecond)
	if err != nil {
		return hostCheck{hostStateInfo, "homelab", "Screen Sharing", "closed", "optional recovery path; enable if remote GUI login is needed"}
	}
	_ = conn.Close()
	return hostCheck{hostStateOK, "homelab", "Screen Sharing", "listening :5900", "usable after FileVault SSH unlock, not before"}
}

func checkJumpDesktopConnectApp(profile hostProfile) hostCheck {
	path := "/Applications/Jump Desktop Connect.app"
	if _, err := os.Stat(path); err == nil {
		return hostCheck{hostStateOK, "homelab", "Jump Connect app", "installed", path}
	}
	return hostCheck{hostStateWarn, "homelab", "Jump Connect app", "missing", "install Jump Desktop Connect for remote GUI recovery"}
}

func checkJumpDesktopClientApp(profile hostProfile) hostCheck {
	path := "/Applications/Jump Desktop.app"
	if _, err := os.Stat(path); err == nil {
		return hostCheck{hostStateOK, "workstation", "Jump client app", "installed", path}
	}
	return hostCheck{hostStateInfo, "workstation", "Jump client app", "missing", "optional viewer app; Connect service is enough for this host"}
}

func checkJumpDesktopProcess(profile hostProfile) hostCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,command")
	lines := filterLinesCaseInsensitive(out, "JumpConnect")
	if len(lines) == 0 {
		return hostCheck{hostStateWarn, "homelab", "Jump service", "not running", "Jump may need GUI/session or service setup"}
	}
	state := hostStateOK
	value := "running"
	detail := fmt.Sprintf("%d process(es)", len(lines))
	if !strings.Contains(out, "--service") {
		state = hostStateWarn
		detail = "Jump processes found, but service process not obvious"
	}
	return hostCheck{state, "homelab", "Jump service", value, detail}
}

func checkJumpDesktopAudioDrivers(profile hostProfile) hostCheck {
	out, _ := runOutput("ps", "-axo", "command")
	hasOut := strings.Contains(out, "JumpAudio.driver")
	hasIn := strings.Contains(out, "JumpAudioMic.driver")
	if hasOut && hasIn {
		return hostCheck{hostStateOK, "homelab", "Jump audio", "running", "speaker and microphone drivers loaded"}
	}
	if hasOut || hasIn {
		return hostCheck{hostStateWarn, "homelab", "Jump audio", "partial", "one Jump audio driver is missing"}
	}
	return hostCheck{hostStateInfo, "homelab", "Jump audio", "not running", "optional; remote GUI may still work without audio"}
}

func checkLockAfterLogin(profile hostProfile) hostCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/LaunchAgents/ai.jterrazz.lock-after-login.plist")
	if _, err := os.Stat(path); err == nil {
		return hostCheck{hostStateOK, "homelab", "Lock after login", "installed", path}
	}
	legacy := filepath.Join(home, "Library/LaunchAgents/ai.alfred.lock-after-login.plist")
	if _, err := os.Stat(legacy); err == nil {
		return hostCheck{hostStateWarn, "homelab", "Lock after login", "legacy label", "run `j host lock-after-login install` to migrate to ai.jterrazz.*"}
	}
	return hostCheck{hostStateInfo, "homelab", "Lock after login", "missing", "run `j host lock-after-login install`"}
}

func checkAgentInbox(profile hostProfile) hostCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/Mobile Documents/com~apple~CloudDocs/Agent Inbox")
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return hostCheck{hostStateOK, "all", "Agent Inbox", "available", path}
	}
	return hostCheck{hostStateWarn, "all", "Agent Inbox", "missing", "shared iCloud folder not found"}
}

func checkDeveloperFolder(profile hostProfile) hostCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Developer")
	entries, err := os.ReadDir(path)
	if err != nil {
		return hostCheck{hostStateWarn, "all", "Developer", "missing", path}
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
	return hostCheck{hostStateOK, "all", "Developer", fmt.Sprintf("%d repos", repos), path}
}

func checkOrbStackInstalled(profile hostProfile) hostCheck {
	path := "/Applications/OrbStack.app"
	if _, err := os.Stat(path); err == nil {
		return hostCheck{hostStateOK, "homelab", "OrbStack", "installed", "can run headless via Background LaunchAgent"}
	}
	return hostCheck{hostStateInfo, "homelab", "OrbStack", "missing", "optional for local containers/Kubernetes"}
}

func checkOrbStackBackgroundAgent(profile hostProfile) hostCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Library/LaunchAgents/ai.orbstack.background-start.plist")
	if _, err := os.Stat(path); err != nil {
		return hostCheck{hostStateWarn, "homelab", "OrbStack bg agent", "missing", "install user/UID Background LaunchAgent to start without GUI login"}
	}
	out, err := runOutput("launchctl", "print", fmt.Sprintf("user/%d/ai.orbstack.background-start", os.Getuid()))
	if err != nil {
		return hostCheck{hostStateWarn, "homelab", "OrbStack bg agent", "plist only", "plist exists but launchd user service is not loaded"}
	}
	if strings.Contains(out, "last exit code = 0") || strings.Contains(out, "state = running") {
		return hostCheck{hostStateOK, "homelab", "OrbStack bg agent", "loaded", "user Background LaunchAgent; works before GUI login"}
	}
	return hostCheck{hostStateWarn, "homelab", "OrbStack bg agent", "loaded", "check launchctl/logs; last run not obviously successful"}
}

func checkOrbStackObsoleteDaemon(profile hostProfile) hostCheck {
	path := "/Library/LaunchDaemons/ai.orbstack.headless-start.plist"
	if _, err := os.Stat(path); err != nil {
		return hostCheck{hostStateOK, "homelab", "OrbStack old daemon", "absent", "obsolete system LaunchDaemon cleaned up"}
	}
	return hostCheck{hostStateWarn, "homelab", "OrbStack old daemon", "present", "remove: LaunchDaemon hits TCC; use Background LaunchAgent instead"}
}

func checkOrbStackStatus(profile hostProfile) hostCheck {
	out, err := runOutput("/usr/local/bin/orbctl", "status")
	if err != nil {
		return hostCheck{hostStateWarn, "homelab", "OrbStack status", "unknown", trimOneLine(out)}
	}
	status := trimOneLine(out)
	if strings.EqualFold(status, "Running") {
		return hostCheck{hostStateOK, "homelab", "OrbStack status", "Running", "Docker/Kubernetes backend is up"}
	}
	return hostCheck{hostStateWarn, "homelab", "OrbStack status", status, "expected to start after FileVault unlock via background agent"}
}

func checkOrbStackProcess(profile hostProfile) hostCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,command")
	lines := filterLinesCaseInsensitive(out, "OrbStack")
	if len(lines) == 0 {
		return hostCheck{hostStateInfo, "homelab", "OrbStack runtime", "not running", "background agent may not have run yet"}
	}
	return hostCheck{hostStateOK, "homelab", "OrbStack runtime", "running", fmt.Sprintf("%d process(es); can run while console owner is root", len(lines))}
}

func checkDocker(profile hostProfile) hostCheck {
	if _, err := exec.LookPath("docker"); err != nil {
		return hostCheck{hostStateInfo, "homelab", "Docker", "missing", "optional unless using containers"}
	}
	out, err := runOutput("docker", "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		return hostCheck{hostStateWarn, "homelab", "Docker", "client only", trimOneLine(out)}
	}
	return hostCheck{hostStateOK, "homelab", "Docker", trimOneLine(out), "daemon reachable"}
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
