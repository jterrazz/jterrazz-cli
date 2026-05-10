package commands

import (
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

const sshAccessGroup = "com.apple.access_ssh"

func installSshd() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.SectionDivider("SSHD ENABLE")
	print.Category("Before")
	dumpSshdState()
	print.Empty()

	print.Category("Applying")

	if out, err := runQuiet("/usr/sbin/systemsetup", "-setremotelogin", "on"); err != nil {
		print.Warning("systemsetup -setremotelogin on: " + oneLineOrDash(out))
	} else {
		print.Success("Remote Login enabled")
	}

	// Idempotently ensure the access_ssh group exists and contains jterrazz.agent.
	if !sshGroupHasMember(sshAccessGroup, autologinTargetUser) {
		_, _ = runQuiet("/usr/sbin/dseditgroup", "-o", "create", "-q", sshAccessGroup)
		if _, err := runQuiet("/usr/sbin/dseditgroup", "-o", "edit", "-a", autologinTargetUser, "-t", "user", sshAccessGroup); err == nil {
			print.Success(autologinTargetUser + " added to " + sshAccessGroup)
		} else {
			print.Warning("dseditgroup edit failed: " + err.Error())
		}
	} else {
		print.Success(autologinTargetUser + " already in " + sshAccessGroup)
	}

	print.Empty()
	print.Category("After")
	dumpSshdState()
	print.Empty()

	print.Warning("Manual step still required:")
	print.Dim("  System Settings → Privacy & Security → FileVault → enable remote-unlock toggle")
	print.Dim("  (label varies by macOS version; only available on supported hardware)")
	return nil
}

func uninstallSshd() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.SectionDivider("SSHD DISABLE")
	print.Category("Before")
	dumpSshdState()
	print.Empty()

	print.Category("Applying")
	if out, err := runQuiet("/usr/sbin/systemsetup", "-setremotelogin", "off"); err != nil {
		print.Warning("systemsetup -setremotelogin off: " + oneLineOrDash(out))
	} else {
		print.Success("Remote Login disabled")
	}
	if sshGroupHasMember(sshAccessGroup, autologinTargetUser) {
		if _, err := runQuiet("/usr/sbin/dseditgroup", "-o", "edit", "-d", autologinTargetUser, "-t", "user", sshAccessGroup); err == nil {
			print.Success(autologinTargetUser + " removed from " + sshAccessGroup)
		} else {
			print.Warning("dseditgroup edit -d failed: " + err.Error())
		}
	}

	print.Empty()
	print.Category("After")
	dumpSshdState()
	return nil
}

// checkSshdInstalled reports whether Remote Login is on AND the agent user is in
// the access_ssh group. Used as a CheckFn for the j config TUI.
func checkSshdInstalled() config.CheckResult {
	out, err := runQuiet("/usr/sbin/systemsetup", "-getremotelogin")
	if err != nil || !strings.Contains(strings.ToLower(out), "on") {
		return config.CheckResult{}
	}
	if !sshGroupHasMember(sshAccessGroup, autologinTargetUser) {
		return config.CheckResult{}
	}
	return config.InstalledWithDetail("Remote Login on, " + autologinTargetUser + " in access_ssh")
}

func dumpSshdState() {
	if out, err := runQuiet("/usr/sbin/systemsetup", "-getremotelogin"); err == nil {
		print.Linef("  systemsetup: %s", oneLineOrDash(out))
	} else {
		print.Linef("  systemsetup: error %s", err.Error())
	}
	if sshGroupHasMember(sshAccessGroup, autologinTargetUser) {
		print.Linef("  %s: contains %s", sshAccessGroup, autologinTargetUser)
	} else {
		print.Linef("  %s: missing %s", sshAccessGroup, autologinTargetUser)
	}
	if out, err := runQuiet("/usr/bin/fdesetup", "status"); err == nil {
		print.Linef("  fdesetup: %s", oneLineOrDash(out))
	}
}

func sshGroupHasMember(group, user string) bool {
	out, err := runQuiet("/usr/bin/dscl", ".", "-read", "/Groups/"+group, "GroupMembership")
	if err != nil {
		return false
	}
	for _, field := range strings.Fields(out) {
		if field == user {
			return true
		}
	}
	return false
}
