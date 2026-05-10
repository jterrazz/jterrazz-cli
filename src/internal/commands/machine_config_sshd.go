package commands

import (
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

const sshAccessGroup = "com.apple.access_ssh"

var machineSshdCmd = &cobra.Command{
	Use:   "sshd",
	Short: "Manage Remote Login (sshd) + FileVault pre-boot SSH unlock",
}

var machineSshdEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Remote Login (sshd) and add jterrazz.agent to access_ssh",
	Long: strings.TrimSpace(`Enable Remote Login (sshd) and restrict it to admins + jterrazz.agent.

The FileVault "remote unlock at startup" feature itself must be toggled in the GUI:
  System Settings → Privacy & Security → FileVault → (the remote-unlock toggle)
This command prints that reminder; macOS does not expose the toggle via CLI on
recent versions without MDM.`),
	Run: func(cmd *cobra.Command, args []string) { runMachineSshdEnable() },
}

var machineSshdStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Remote Login state and access_ssh group membership",
	Run:   func(cmd *cobra.Command, args []string) { runMachineSshdStatus() },
}

func init() {
	machineSshdCmd.AddCommand(machineSshdEnableCmd, machineSshdStatusCmd)
	machineConfigCmd.AddCommand(machineSshdCmd)
}

func runMachineSshdEnable() {
	failOn(requireDarwin())
	failOn(requireRoot())

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
}

func runMachineSshdStatus() {
	failOn(requireDarwin())
	print.SectionDivider("SSHD STATUS")
	dumpSshdState()
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
