package commands

import (
	"fmt"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

// powerHardenSettings is the canonical homelab power policy: never sleep, restart
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

var machinePowerCmd = &cobra.Command{
	Use:   "power",
	Short: "Manage homelab power policy (sleep, autorestart, wake)",
}

var machinePowerEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Apply always-on homelab power policy via pmset -a",
	Long: strings.TrimSpace(`Apply the always-on homelab power policy.

Sets: autorestart=1, sleep=0, displaysleep=5, disksleep=0, powernap=0,
hibernatemode=0, womp=1. Idempotent — re-running re-applies and prints the
current state.`),
	Run: func(cmd *cobra.Command, args []string) { failOn(enablePowerHarden()) },
}

var machinePowerDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Reset pmset to macOS defaults",
	Run:   func(cmd *cobra.Command, args []string) { failOn(disablePowerHarden()) },
}

var machinePowerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pmset -g custom output",
	Run:   func(cmd *cobra.Command, args []string) { failOn(statusPower()) },
}

func init() {
	machinePowerCmd.AddCommand(machinePowerEnableCmd, machinePowerDisableCmd, machinePowerStatusCmd)
	machineConfigCmd.AddCommand(machinePowerCmd)
}

func enablePowerHarden() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.SectionDivider("POWER ENABLE")
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

func disablePowerHarden() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	if err := requireRoot(); err != nil {
		return err
	}

	print.SectionDivider("POWER DISABLE")
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

func statusPower() error {
	if err := requireDarwin(); err != nil {
		return err
	}
	print.SectionDivider("POWER STATUS")
	dumpPmset()
	return nil
}

// checkPowerHardened reports whether all powerHardenSettings are currently
// applied. Used as a CheckFn for the j config TUI.
func checkPowerHardened() config.CheckResult {
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
