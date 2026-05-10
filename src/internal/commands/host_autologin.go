package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

const autologinTargetUser = "jterrazz.agent"

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
	if password == "" {
		// Apple's sysadminctl on recent macOS silently no-ops when invoked under sudo
		// without -password, so we MUST get a password before calling it. Prompt on
		// /dev/tty with stty no-echo so the password never appears in scrollback or env.
		pw, err := promptPasswordTTY(fmt.Sprintf("Agent password for %s (sets /etc/kcpassword): ", autologinTargetUser))
		if err != nil {
			failOn(fmt.Errorf("read password: %w (or pre-set %s in the env via `sudo --preserve-env=%s ...`)", err, autologinPasswordEnv, autologinPasswordEnv))
		}
		password = pw
	}
	if password == "" {
		failOn(fmt.Errorf("empty password — refusing to call sysadminctl with no -password (it would silently no-op)"))
	}

	print.SectionDivider("AUTOLOGIN ENABLE")
	print.Category("Before")
	dumpAutologinState()
	print.Empty()

	print.Category("Applying")
	failOn(run("/usr/bin/defaults", "write", "/Library/Preferences/com.apple.loginwindow", "DisableFDEAutoLogin", "-bool", "NO"))
	print.Success("DisableFDEAutoLogin = NO")

	// macOS 26 sysadminctl -autoLogin set refuses on FileVault-enabled disks
	// regardless of DisableFDEAutoLogin or admin auth (the "FileVault is enabled"
	// message is non-recoverable on Tahoe). Bypass it: write /etc/kcpassword with
	// the well-known XOR cipher that loginwindow has consumed since 10.4, and set
	// autoLoginUser via defaults. loginwindow at boot still respects
	// DisableFDEAutoLogin and processes /etc/kcpassword.
	if err := os.WriteFile("/etc/kcpassword", encodeKCPassword(password), 0o600); err != nil {
		failOn(fmt.Errorf("write /etc/kcpassword: %w", err))
	}
	if err := os.Chmod("/etc/kcpassword", 0o600); err != nil {
		failOn(err)
	}
	// /etc/kcpassword must be owned by root:wheel; we already are root via sudo.
	_ = os.Chown("/etc/kcpassword", 0, 0)
	print.Success("/etc/kcpassword written (root:wheel 0600)")

	if err := run("/usr/bin/defaults", "write", "/Library/Preferences/com.apple.loginwindow", "autoLoginUser", "-string", autologinTargetUser); err != nil {
		failOn(fmt.Errorf("defaults write autoLoginUser: %w", err))
	}
	print.Success("autoLoginUser = " + autologinTargetUser)

	// Cross-check both side effects.
	if _, err := os.Stat("/etc/kcpassword"); err != nil {
		failOn(fmt.Errorf("/etc/kcpassword missing after write"))
	}
	out, err := runQuiet("/usr/bin/defaults", "read", "/Library/Preferences/com.apple.loginwindow", "autoLoginUser")
	if err != nil || strings.TrimSpace(out) != autologinTargetUser {
		failOn(fmt.Errorf("autoLoginUser cross-check failed (got %q)", strings.TrimSpace(out)))
	}

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

// promptPasswordTTY reads a password from /dev/tty without echo. We deliberately
// don't use os.Stdin: in this CLI, sudo+ssh pipelines often have stdin replaced by
// a pipe, but /dev/tty is the controlling terminal and stays interactive.
//
// Implementation note: shelling out to stty avoids adding a third-party dep just
// to set raw termios for one prompt. On panic/early exit, the deferred stty echo
// restores the terminal state.
func promptPasswordTTY(prompt string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("open /dev/tty: %w", err)
	}
	defer tty.Close()

	stty := func(args ...string) error {
		c := exec.Command("/bin/stty", args...)
		c.Stdin = tty
		return c.Run()
	}
	if err := stty("-echo"); err != nil {
		return "", fmt.Errorf("stty -echo: %w", err)
	}
	defer stty("echo") //nolint:errcheck // best-effort restore

	if _, err := fmt.Fprint(tty, prompt); err != nil {
		return "", err
	}
	reader := bufio.NewReader(tty)
	line, err := reader.ReadString('\n')
	fmt.Fprintln(tty)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func oneLineOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return strings.Join(strings.Fields(strings.Split(s, "\n")[0]), " ")
}
