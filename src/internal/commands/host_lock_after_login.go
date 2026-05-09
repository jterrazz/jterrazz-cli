package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

const (
	lockAfterLoginUser    = "jterrazz.agent"
	lockAfterLoginLabel   = "ai.jterrazz.lock-after-login"
	lockAfterLoginScript  = "/Users/jterrazz.agent/.openclaw/scripts/lock-after-login.sh"
	lockAfterLoginPlist   = "/Users/jterrazz.agent/Library/LaunchAgents/ai.jterrazz.lock-after-login.plist"
	lockAfterLoginLogDir  = "/Users/jterrazz.agent/.openclaw/logs"
)

var hostLockAfterLoginCmd = &cobra.Command{
	Use:     "lock-after-login",
	Aliases: []string{"lockafterlogin"},
	Short:   "Manage the LaunchAgent that locks the screen ~20s after auto-login",
}

var hostLockAfterLoginInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the lock-after-login LaunchAgent for jterrazz.agent",
	Long: strings.TrimSpace(`Install the per-user LaunchAgent that runs lock-after-login.sh on auto-login.

This wraps the existing ~/.openclaw/scripts/lock-after-login.sh — the script is reused, not duplicated.
Idempotent: rewrites the plist with current paths and re-bootstraps the agent if a GUI session is active.`),
	Run: func(cmd *cobra.Command, args []string) { runHostLockAfterLoginInstall() },
}

var hostLockAfterLoginUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Bootout and remove the lock-after-login LaunchAgent",
	Run:   func(cmd *cobra.Command, args []string) { runHostLockAfterLoginUninstall() },
}

var hostLockAfterLoginStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the lock-after-login LaunchAgent is installed and loaded",
	Run:   func(cmd *cobra.Command, args []string) { runHostLockAfterLoginStatus() },
}

func init() {
	hostLockAfterLoginCmd.AddCommand(hostLockAfterLoginInstallCmd, hostLockAfterLoginUninstallCmd, hostLockAfterLoginStatusCmd)
	hostCmd.AddCommand(hostLockAfterLoginCmd)
}

func runHostLockAfterLoginInstall() {
	failOn(requireDarwin())
	failOn(requireRoot())

	if _, err := os.Stat(lockAfterLoginScript); err != nil {
		failOn(fmt.Errorf("missing %s — expected the openclaw lock-after-login script", lockAfterLoginScript))
	}

	print.SectionDivider("LOCK-AFTER-LOGIN INSTALL")
	print.Category("Before")
	dumpLockAfterLoginState()
	print.Empty()

	if err := os.Chmod(lockAfterLoginScript, 0o755); err != nil {
		failOn(err)
	}
	if err := os.MkdirAll(lockAfterLoginLogDir, 0o755); err != nil {
		failOn(err)
	}
	if err := os.MkdirAll(filepath.Dir(lockAfterLoginPlist), 0o755); err != nil {
		failOn(err)
	}

	plist := buildLockAfterLoginPlist()
	if err := os.WriteFile(lockAfterLoginPlist, []byte(plist), 0o644); err != nil {
		failOn(err)
	}

	uid, gid := lookupTargetUserIDs(lockAfterLoginUser)
	if uid >= 0 && gid >= 0 {
		_ = os.Chown(lockAfterLoginPlist, uid, gid)
		_ = os.Chown(lockAfterLoginLogDir, uid, gid)
	}

	print.Success("Wrote " + lockAfterLoginPlist)

	// Migrate from the legacy ai.alfred.* label if it's still installed: bootout the
	// old launchd entry and remove its plist so we don't end up locking twice on login.
	migrateLegacyLockAfterLogin(uid)

	// Try to bootstrap the agent in the live GUI session if one exists.
	consoleOwner, _ := runQuiet("/usr/bin/stat", "-f", "%Su", "/dev/console")
	if strings.TrimSpace(consoleOwner) == lockAfterLoginUser && uid >= 0 {
		domain := fmt.Sprintf("gui/%d", uid)
		// Bootout first to make this idempotent if already loaded.
		_ = exec.Command("/bin/launchctl", "bootout", domain+"/"+lockAfterLoginLabel).Run()
		if out, err := runQuiet("/bin/launchctl", "bootstrap", domain, lockAfterLoginPlist); err != nil {
			print.Warning("launchctl bootstrap failed (will run on next auto-login regardless): " + oneLineOrDash(out))
		} else {
			print.Success("launchctl bootstrap " + domain)
		}
	} else {
		print.Dim("No active GUI session for " + lockAfterLoginUser + " — agent will load at next auto-login.")
	}

	print.Empty()
	print.Category("After")
	dumpLockAfterLoginState()
}

func runHostLockAfterLoginUninstall() {
	failOn(requireDarwin())
	failOn(requireRoot())

	uid, _ := lookupTargetUserIDs(lockAfterLoginUser)
	if uid >= 0 {
		_ = exec.Command("/bin/launchctl", "bootout", fmt.Sprintf("gui/%d/%s", uid, lockAfterLoginLabel)).Run()
	}
	if err := os.Remove(lockAfterLoginPlist); err != nil && !os.IsNotExist(err) {
		failOn(err)
	}
	print.SectionDivider("LOCK-AFTER-LOGIN UNINSTALL")
	print.Success("Removed " + lockAfterLoginPlist)
	print.Category("After")
	dumpLockAfterLoginState()
}

func runHostLockAfterLoginStatus() {
	failOn(requireDarwin())
	print.SectionDivider("LOCK-AFTER-LOGIN STATUS")
	dumpLockAfterLoginState()
}

func dumpLockAfterLoginState() {
	if _, err := os.Stat(lockAfterLoginPlist); err == nil {
		print.Linef("  plist: present (%s)", lockAfterLoginPlist)
	} else {
		print.Linef("  plist: absent")
	}
	if _, err := os.Stat(lockAfterLoginScript); err == nil {
		print.Linef("  script: present (%s)", lockAfterLoginScript)
	} else {
		print.Linef("  script: MISSING (%s)", lockAfterLoginScript)
	}
	if uid, _ := lookupTargetUserIDs(lockAfterLoginUser); uid >= 0 {
		out, err := runQuiet("/bin/launchctl", "print", fmt.Sprintf("gui/%d/%s", uid, lockAfterLoginLabel))
		if err != nil {
			print.Linef("  launchctl: not loaded in gui/%d", uid)
			return
		}
		state := "loaded"
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "state = ") {
				state = strings.TrimPrefix(line, "state = ")
				break
			}
		}
		print.Linef("  launchctl: %s in gui/%d", state, uid)
	}
}

func buildLockAfterLoginPlist() string {
	return strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		`<dict>`,
		`    <key>Label</key>`,
		`    <string>` + lockAfterLoginLabel + `</string>`,
		`    <key>ProgramArguments</key>`,
		`    <array>`,
		`        <string>/bin/zsh</string>`,
		`        <string>` + lockAfterLoginScript + `</string>`,
		`    </array>`,
		`    <key>RunAtLoad</key>`,
		`    <true/>`,
		`    <key>StandardOutPath</key>`,
		`    <string>` + lockAfterLoginLogDir + `/lock-after-login.log</string>`,
		`    <key>StandardErrorPath</key>`,
		`    <string>` + lockAfterLoginLogDir + `/lock-after-login.err.log</string>`,
		`    <key>ProcessType</key>`,
		`    <string>Interactive</string>`,
		`</dict>`,
		`</plist>`,
		"",
	}, "\n")
}

const legacyLockAfterLoginLabel = "ai.alfred.lock-after-login"

var legacyLockAfterLoginPlist = "/Users/jterrazz.agent/Library/LaunchAgents/" + legacyLockAfterLoginLabel + ".plist"

func migrateLegacyLockAfterLogin(uid int) {
	if _, err := os.Stat(legacyLockAfterLoginPlist); err != nil {
		return
	}
	if uid >= 0 {
		_ = exec.Command("/bin/launchctl", "bootout", fmt.Sprintf("gui/%d/%s", uid, legacyLockAfterLoginLabel)).Run()
	}
	if err := os.Remove(legacyLockAfterLoginPlist); err == nil {
		print.Success("Migrated: removed legacy " + legacyLockAfterLoginPlist)
	}
}

func lookupTargetUserIDs(username string) (int, int) {
	uidStr, err := runQuiet("/usr/bin/id", "-u", username)
	if err != nil {
		return -1, -1
	}
	gidStr, err := runQuiet("/usr/bin/id", "-g", username)
	if err != nil {
		return -1, -1
	}
	uid, err := strconv.Atoi(strings.TrimSpace(uidStr))
	if err != nil {
		return -1, -1
	}
	gid, err := strconv.Atoi(strings.TrimSpace(gidStr))
	if err != nil {
		return -1, -1
	}
	return uid, gid
}
