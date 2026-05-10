package commands

import (
	"strings"

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

var machinePowerHardenCmd = &cobra.Command{
	Use:   "harden",
	Short: "Apply always-on homelab power policy via pmset -a",
	Long: strings.TrimSpace(`Apply the always-on homelab power policy.

Sets: autorestart=1, sleep=0, displaysleep=5, disksleep=0, powernap=0,
hibernatemode=0, womp=1. Idempotent — re-running re-applies and prints the
current state.`),
	Run: func(cmd *cobra.Command, args []string) { runMachinePowerHarden() },
}

var machinePowerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pmset -g custom output",
	Run:   func(cmd *cobra.Command, args []string) { runMachinePowerStatus() },
}

func init() {
	machinePowerCmd.AddCommand(machinePowerHardenCmd, machinePowerStatusCmd)
	machineCmd.AddCommand(machinePowerCmd)
}

func runMachinePowerHarden() {
	failOn(requireDarwin())
	failOn(requireRoot())

	print.SectionDivider("POWER HARDEN")
	print.Category("Before")
	dumpPmset()
	print.Empty()

	print.Category("Applying")
	args := []string{"-a"}
	for _, kv := range powerHardenSettings {
		args = append(args, kv[0], kv[1])
	}
	failOn(run("/usr/bin/pmset", args...))
	for _, kv := range powerHardenSettings {
		print.Success(kv[0] + "=" + kv[1])
	}

	print.Empty()
	print.Category("After")
	dumpPmset()
}

func runMachinePowerStatus() {
	failOn(requireDarwin())
	print.SectionDivider("POWER STATUS")
	dumpPmset()
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
