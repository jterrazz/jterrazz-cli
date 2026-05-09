package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

const (
	autologinTargetUser    = "jterrazz.agent"
	autologinDisableScript = "/Users/jterrazz.agent/.openclaw/scripts/disable-agent-autologin.sh"
)

var autologinPasswordEnv string

var hostAutologinCmd = &cobra.Command{
	Use:   "autologin",
	Short: "Manage GUI auto-login for the agent user",
	Long: strings.TrimSpace(`Manage GUI auto-login for jterrazz.agent on a FileVault-encrypted Mac.

Auto-login bypasses the loginwindow on cold boot or after fdesetup authrestart, so an
agent runtime can drive the Aqua session without anyone at the keyboard. Per-user
"lock after login" must be installed separately to keep the screen physically protected.`),
}

var hostAutologinEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable auto-login for jterrazz.agent (FileVault-aware)",
	Long: strings.TrimSpace(`Enable auto-login for jterrazz.agent.

If the env var named by --password-env (default AGENT_PASSWORD) is set, its value is
passed to sysadminctl directly. Otherwise sysadminctl will prompt interactively for
both the admin and the agent password. The password is never echoed and never written
to disk except via macOS's own /etc/kcpassword (which sysadminctl manages).`),
	Run: func(cmd *cobra.Command, args []string) { runHostAutologinEnable() },
}

var hostAutologinDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable auto-login and clear /etc/kcpassword",
	Run:   func(cmd *cobra.Command, args []string) { runHostAutologinDisable() },
}

var hostAutologinStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show auto-login + FileVault auto-login override state",
	Run:   func(cmd *cobra.Command, args []string) { runHostAutologinStatus() },
}

func init() {
	hostAutologinEnableCmd.Flags().StringVar(&autologinPasswordEnv, "password-env", "AGENT_PASSWORD", "env var that holds the agent password (read at run time, never logged)")
	hostAutologinCmd.AddCommand(hostAutologinEnableCmd, hostAutologinDisableCmd, hostAutologinStatusCmd)
	hostCmd.AddCommand(hostAutologinCmd)
}

func runHostAutologinStatus() {
	failOn(requireDarwin())
	print.SectionDivider("AUTOLOGIN STATUS")
	dumpAutologinState()
}

func runHostAutologinEnable() {
	failOn(requireDarwin())
	failOn(requireRoot())

	if _, err := exec.LookPath("sysadminctl"); err != nil {
		failOn(fmt.Errorf("sysadminctl not found: %w", err))
	}
	if _, err := os.Stat("/Users/" + autologinTargetUser); err != nil {
		failOn(fmt.Errorf("user %s does not exist on this Mac", autologinTargetUser))
	}

	password := os.Getenv(autologinPasswordEnv)

	print.SectionDivider("AUTOLOGIN ENABLE")
	print.Category("Before")
	dumpAutologinState()
	print.Empty()

	print.Category("Applying")
	failOn(run("/usr/bin/defaults", "write", "/Library/Preferences/com.apple.loginwindow", "DisableFDEAutoLogin", "-bool", "NO"))
	print.Success("DisableFDEAutoLogin = NO")

	args := []string{"-autologin", "set", "-userName", autologinTargetUser}
	if password != "" {
		args = append(args, "-password", password)
	}
	if err := run("/usr/sbin/sysadminctl", args...); err != nil {
		failOn(fmt.Errorf("sysadminctl -autologin set failed: %w", err))
	}
	print.Success("sysadminctl -autoLogin set")

	// Belt & braces: clear the screen-locked-after-resume flag so the GUI session
	// stays interactive after auto-login. lock-after-login handles the lock itself.
	_ = exec.Command("/usr/bin/defaults", "delete", "/Library/Preferences/com.apple.loginwindow", "autoLoginUserScreenLocked").Run()

	print.Empty()
	print.Category("After")
	dumpAutologinState()
	print.Empty()
	print.Dim("Verify end-to-end: `sudo fdesetup authrestart -delayminutes 0` (or `j host restart` from the MacBook).")
}

func runHostAutologinDisable() {
	failOn(requireDarwin())
	failOn(requireRoot())

	if _, err := os.Stat(autologinDisableScript); err == nil {
		// Reuse the existing rollback so behavior stays identical to the original setup script.
		print.SectionDivider("AUTOLOGIN DISABLE")
		print.Dim("Delegating to " + autologinDisableScript)
		failOn(run("/bin/zsh", autologinDisableScript))
		return
	}

	// Fallback: same effects, if the rollback script has been moved.
	print.SectionDivider("AUTOLOGIN DISABLE")
	_ = run("/usr/sbin/sysadminctl", "-autologin", "off")
	_ = os.Remove("/etc/kcpassword")
	for _, key := range []string{"autoLoginUser", "oneTimeAutoLogin", "autoLoginUserScreenLocked"} {
		_ = exec.Command("/usr/bin/defaults", "delete", "/Library/Preferences/com.apple.loginwindow", key).Run()
	}
	print.Empty()
	print.Category("After")
	dumpAutologinState()
}

func dumpAutologinState() {
	if out, err := runQuiet("/usr/sbin/sysadminctl", "-autologin", "status"); err == nil || out != "" {
		print.Linef("  sysadminctl: %s", oneLineOrDash(out))
	} else {
		print.Linef("  sysadminctl: %s", err.Error())
	}

	for _, key := range []string{"autoLoginUser", "DisableFDEAutoLogin"} {
		out, err := runQuiet("/usr/bin/defaults", "read", "/Library/Preferences/com.apple.loginwindow", key)
		if err != nil {
			print.Linef("  %s: (unset)", key)
			continue
		}
		print.Linef("  %s: %s", key, oneLineOrDash(out))
	}

	if info, err := os.Stat("/etc/kcpassword"); err == nil {
		print.Linef("  /etc/kcpassword: present (%d bytes, mode %v)", info.Size(), info.Mode().Perm())
	} else {
		print.Linef("  /etc/kcpassword: absent")
	}

	if owner, err := runQuiet("/usr/bin/stat", "-f", "%Su", "/dev/console"); err == nil {
		print.Linef("  /dev/console owner: %s", oneLineOrDash(owner))
	}
}

func oneLineOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return strings.Join(strings.Fields(strings.Split(s, "\n")[0]), " ")
}
