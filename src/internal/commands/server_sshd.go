package commands

import (
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

func installSshd() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.Header("install sshd", machineContext())
	print.Category("Before")
	dumpSshdState()
	print.Empty()

	print.Category("Applying")
	if out, err := runQuiet("/usr/sbin/systemsetup", "-setremotelogin", "on"); err != nil {
		print.Warning("systemsetup -setremotelogin on: " + oneLineOrDash(out))
	} else {
		print.Success("Remote Login enabled")
	}

	print.Empty()
	print.Category("After")
	dumpSshdState()
	return nil
}

func uninstallSshd() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.Header("uninstall sshd", machineContext())
	print.Category("Before")
	dumpSshdState()
	print.Empty()

	print.Category("Applying")
	if out, err := runQuiet("/usr/sbin/systemsetup", "-setremotelogin", "off"); err != nil {
		print.Warning("systemsetup -setremotelogin off: " + oneLineOrDash(out))
	} else {
		print.Success("Remote Login disabled")
	}

	print.Empty()
	print.Category("After")
	dumpSshdState()
	return nil
}

// checkSshdInstalled reports whether macOS Remote Login is on.
// Used as a CheckFn for the j config TUI.
func checkSshdInstalled() config.CheckResult {
	out, err := runQuiet("/usr/sbin/systemsetup", "-getremotelogin")
	if err != nil || !strings.Contains(strings.ToLower(out), "on") {
		return config.CheckResult{}
	}
	return config.InstalledWithDetail("Remote Login on")
}

func dumpSshdState() {
	if out, err := runQuiet("/usr/sbin/systemsetup", "-getremotelogin"); err == nil {
		print.Linef("  systemsetup: %s", oneLineOrDash(out))
	} else {
		print.Linef("  systemsetup: error %s", err.Error())
	}
}
