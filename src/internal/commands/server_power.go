package commands

import (
	"fmt"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

// powerHardenSettings is the canonical server power policy: never sleep, restart
// on power return, no hibernate-to-disk (faster recovery), display can sleep,
// wake on network access. Order is fixed so pmset's diagnostics are deterministic.
//
// Note: SleepOnPowerButton is not settable via pmset on Apple silicon — it shows
// in `pmset -g` but is firmware-controlled. We don't try to set it.
var powerHardenSettings = [][2]string{
	{"autorestart", "1"},
	{"sleep", "0"},
	{"displaysleep", "5"},
	{"disksleep", "0"},
	{"powernap", "0"},
	{"hibernatemode", "0"},
	{"womp", "1"},
}

func installPower() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.Header("install power", "")
	print.Category("Before")
	dumpPmset()
	print.Empty()

	print.Category("Applying")
	args := []string{"-a"}
	for _, kv := range powerHardenSettings {
		args = append(args, kv[0], kv[1])
	}
	if err := run("/usr/bin/pmset", args...); err != nil {
		return err
	}
	for _, kv := range powerHardenSettings {
		print.Success(kv[0] + "=" + kv[1])
	}

	print.Empty()
	print.Category("After")
	dumpPmset()
	return nil
}

func uninstallPower() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.Header("uninstall power", "")
	print.Category("Before")
	dumpPmset()
	print.Empty()

	print.Category("Applying")
	if err := run("/usr/bin/pmset", "-a", "-resetdefaults"); err != nil {
		return fmt.Errorf("pmset -resetdefaults: %w", err)
	}
	print.Success("pmset reset to macOS defaults")

	print.Empty()
	print.Category("After")
	dumpPmset()
	return nil
}

// checkPowerInstalled reports whether all powerHardenSettings are currently
// applied. Used as a CheckFn for the j config TUI.
func checkPowerInstalled() config.CheckResult {
	out, err := runQuiet("/usr/bin/pmset", "-g", "custom")
	if err != nil {
		return config.CheckResult{}
	}
	settings := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 {
			settings[fields[0]] = fields[1]
		}
	}
	for _, kv := range powerHardenSettings {
		if settings[kv[0]] != kv[1] {
			return config.CheckResult{}
		}
	}
	return config.InstalledWithDetail("harden applied")
}

func dumpPmset() {
	out, err := runQuiet("/usr/bin/pmset", "-g", "custom")
	if err != nil {
		print.Warning("pmset -g custom failed: " + err.Error())
		return
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		print.Linef("  %s", line)
	}
}
