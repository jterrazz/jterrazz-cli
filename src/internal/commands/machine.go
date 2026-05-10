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

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
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
	Name   string
	Value  string
	Detail string
}

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Manage and inspect this machine and its services",
}

var machineStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show machine + service status",
	Run: func(cmd *cobra.Command, args []string) {
		runMachineStatus()
	},
}

var machineUnlockCmd = &cobra.Command{
	Use:   "unlock <alias>",
	Short: "Unlock a FileVault-protected homelab Mac over preboot SSH",
	Long: `Open an interactive SSH session in pre-boot mode to enter the FileVault
unlock password. The endpoint comes from the registry (see j machine list).

Pre-boot SSH advertises a different host key than the running OS and only
accepts password auth — no authorized_keys exist before FileVault unlock.`,
	Args: cobra.ExactArgs(1),
	Run:  func(cmd *cobra.Command, args []string) { runMachineUnlock(args[0]) },
}

func init() {
	machineCmd.AddCommand(machineStatusCmd, machineUnlockCmd)
	rootCmd.AddCommand(machineCmd)
}

func runMachineUnlock(alias string) {
	target := resolveRemoteSSH(alias)
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

// resolveRemoteSSH looks up the alias in the registry and returns its ssh
// endpoint (user@host). Exits with an error if the alias is unknown, has no
// SSH endpoint (it's a local-only machine), or is the alias marked as self.
func resolveRemoteSSH(alias string) string {
	selfAlias, _, _ := config.SelfMachine()
	if alias == selfAlias {
		failOn(fmt.Errorf("%q is this machine — refusing to act on self", alias))
	}
	m, ok := config.GetMachine(alias)
	if !ok {
		failOn(fmt.Errorf("machine %q not found in registry — see `j machine list`", alias))
	}
	if m.SSH == "" {
		failOn(fmt.Errorf("machine %q has no ssh endpoint configured", alias))
	}
	return m.SSH
}

// machineSelfRole returns the role of the current machine according to the
// registry, defaulting to RoleDev when no machine has been declared as self.
func machineSelfRole() config.Role {
	if _, m, ok := config.SelfMachine(); ok {
		return m.Role
	}
	return config.RoleDev
}

func runMachineStatus() {
	role := machineSelfRole()

	print.SectionDivider("MACHINE")
	print.Linef("Host: %s", hostname())
	print.Linef("Role: %s", role)
	print.Linef("OS: %s", osSummary())
	print.Empty()

	// State of the machine itself: encryption + reachability.
	print.Category("Machine")
	for _, check := range machineStateChecks() {
		printMachineCheck(check)
	}
	print.Empty()

	// Application state. Only meaningful on homelab — a dev box doesn't
	// host these services so the rows would always read "not running".
	if role == config.RoleHomelab {
		print.Category("Services")
		for _, check := range serviceStateChecks() {
			printMachineCheck(check)
		}
		print.Empty()
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
		fmt.Printf("  %s %-22s %-18s %s\n", icon, c.Name, value, c.Detail)
		return
	}
	fmt.Printf("  %s %-22s %s\n", icon, c.Name, value)
}

func machineStateChecks() []machineCheck {
	return []machineCheck{
		checkFileVault(),
		checkSSH(),
	}
}

func serviceStateChecks() []machineCheck {
	checks := []machineCheck{
		checkOpenClawProcess(),
		checkOpenClawConfig(),
	}
	checks = append(checks, checkOpenClawChannels()...)
	checks = append(checks, checkOrbStackStatus())
	return checks
}

func checkFileVault() machineCheck {
	out, err := runOutput("fdesetup", "status")
	if err != nil {
		return machineCheck{machineStateUnknown, "FileVault", "unknown", trimOneLine(out)}
	}
	line := trimOneLine(out)
	if strings.Contains(line, "FileVault is On") {
		return machineCheck{machineStateOK, "FileVault", "on", "remote SSH unlock requires macOS 26+ Apple silicon"}
	}
	return machineCheck{machineStateFail, "FileVault", line, "enable disk encryption"}
}

func checkSSH() machineCheck {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:22", 750*time.Millisecond)
	if err != nil {
		return machineCheck{machineStateFail, "SSH", "closed", "Remote Login/sshd is not reachable on localhost:22"}
	}
	_ = conn.Close()
	return machineCheck{machineStateOK, "SSH", "listening :22", "pre-FileVault unlock uses password auth, not authorized_keys"}
}

func checkOpenClawProcess() machineCheck {
	out, _ := runOutput("ps", "-axo", "user,pid,ppid,command")
	lines := filterLines(out, "openclaw/dist/index.js", "gateway")
	if len(lines) == 0 {
		return machineCheck{machineStateFail, "OpenClaw runtime", "not running", "gateway process not found"}
	}
	line := strings.Join(lines, " | ")
	state := machineStateOK
	value := "running"
	detail := line
	if !strings.Contains(line, "jterrazz.agent") {
		state = machineStateWarn
		detail = "unexpected owner: " + line
	}
	return machineCheck{state, "OpenClaw runtime", value, detail}
}

func checkOpenClawConfig() machineCheck {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".openclaw/openclaw.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return machineCheck{machineStateFail, "OpenClaw config", "missing", path}
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return machineCheck{machineStateFail, "OpenClaw config", "invalid", err.Error()}
	}
	model := jsonPathString(cfg, "models", "default")
	if model == "" {
		model = jsonPathString(cfg, "agents", "default", "model")
	}
	if model == "" {
		model = "configured"
	}
	return machineCheck{machineStateOK, "OpenClaw config", model, path}
}

func checkOpenClawChannels() []machineCheck {
	out, err := runOutput("openclaw", "channels", "status")
	if err != nil {
		return []machineCheck{{machineStateWarn, "Channels", "unknown", trimOneLine(out)}}
	}
	return []machineCheck{
		channelCheck(out, "Slack"),
		channelCheck(out, "Telegram"),
		channelCheck(out, "BlueBubbles"),
	}
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
			return machineCheck{state, name, value, strings.TrimSpace(line)}
		}
	}
	return machineCheck{machineStateWarn, name, "not found", "channel not reported by OpenClaw"}
}

func checkOrbStackStatus() machineCheck {
	out, err := runOutput("/usr/local/bin/orbctl", "status")
	if err != nil {
		return machineCheck{machineStateWarn, "OrbStack", "unknown", trimOneLine(out)}
	}
	status := trimOneLine(out)
	if strings.EqualFold(status, "Running") {
		return machineCheck{machineStateOK, "OrbStack", "Running", "Docker/Kubernetes backend is up"}
	}
	return machineCheck{machineStateWarn, "OrbStack", status, "expected to start after FileVault unlock"}
}

func runOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
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
