package commands

import (
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

const prebootSSHGroup = "com.apple.access_ssh"

var hostPrebootSSHCmd = &cobra.Command{
	Use:   "preboot-ssh",
	Short: "Manage Remote Login + FileVault pre-boot SSH unlock",
}

var hostPrebootSSHEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Remote Login (sshd) and add jterrazz.agent to access_ssh",
	Long: strings.TrimSpace(`Enable Remote Login (sshd) and restrict it to admins + jterrazz.agent.

The FileVault "remote unlock at startup" feature itself must be toggled in the GUI:
  System Settings → Privacy & Security → FileVault → (the remote-unlock toggle)
This command prints that reminder; macOS does not expose the toggle via CLI on
recent versions without MDM.`),
	Run: func(cmd *cobra.Command, args []string) { runHostPrebootSSHEnable() },
}

var hostPrebootSSHStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Remote Login state and access_ssh group membership",
	Run:   func(cmd *cobra.Command, args []string) { runHostPrebootSSHStatus() },
}

func init() {
	hostPrebootSSHCmd.AddCommand(hostPrebootSSHEnableCmd, hostPrebootSSHStatusCmd)
	hostCmd.AddCommand(hostPrebootSSHCmd)
}

func runHostPrebootSSHEnable() {
	failOn(requireDarwin())
	failOn(requireRoot())

	print.SectionDivider("PREBOOT-SSH ENABLE")
	print.Category("Before")
	dumpPrebootSSHState()
	print.Empty()

	print.Category("Applying")

	if out, err := runQuiet("/usr/sbin/systemsetup", "-setremotelogin", "on"); err != nil {
		print.Warning("systemsetup -setremotelogin on: " + oneLineOrDash(out))
	} else {
		print.Success("Remote Login enabled")
	}

	// Idempotently ensure the access_ssh group exists and contains jterrazz.agent.
	if !sshGroupHasMember(prebootSSHGroup, autologinTargetUser) {
		_, _ = runQuiet("/usr/sbin/dseditgroup", "-o", "create", "-q", prebootSSHGroup)
		if _, err := runQuiet("/usr/sbin/dseditgroup", "-o", "edit", "-a", autologinTargetUser, "-t", "user", prebootSSHGroup); err == nil {
			print.Success(autologinTargetUser + " added to " + prebootSSHGroup)
		} else {
			print.Warning("dseditgroup edit failed: " + err.Error())
		}
	} else {
		print.Success(autologinTargetUser + " already in " + prebootSSHGroup)
	}

	print.Empty()
	print.Category("After")
	dumpPrebootSSHState()
	print.Empty()

	print.Warning("Manual step still required:")
	print.Dim("  System Settings → Privacy & Security → FileVault → enable remote-unlock toggle")
	print.Dim("  (label varies by macOS version; only available on supported hardware)")
}

func runHostPrebootSSHStatus() {
	failOn(requireDarwin())
	print.SectionDivider("PREBOOT-SSH STATUS")
	dumpPrebootSSHState()
}

func dumpPrebootSSHState() {
	if out, err := runQuiet("/usr/sbin/systemsetup", "-getremotelogin"); err == nil {
		print.Linef("  systemsetup: %s", oneLineOrDash(out))
	} else {
		print.Linef("  systemsetup: error %s", err.Error())
	}
	if sshGroupHasMember(prebootSSHGroup, autologinTargetUser) {
		print.Linef("  %s: contains %s", prebootSSHGroup, autologinTargetUser)
	} else {
		print.Linef("  %s: missing %s", prebootSSHGroup, autologinTargetUser)
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
