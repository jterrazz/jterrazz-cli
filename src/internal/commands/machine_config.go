package commands

import (
	"fmt"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/spf13/cobra"
)

var machineConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure this machine (homelab-only)",
	Long: "Local configuration commands that mutate macOS state — auto-login, " +
		"power policy, screen lock-after-login, and Remote Login (sshd).\n\n" +
		"All sub-commands refuse to run unless the current machine is registered " +
		"as homelab in ~/.jterrazz/config.json. This avoids accidentally applying " +
		"server-side configuration to a dev box.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return requireHomelabRole()
	},
}

func init() {
	machineCmd.AddCommand(machineConfigCmd)
}

// requireHomelabRole returns nil if the current machine is registered as
// homelab in the registry, otherwise an error explaining what to fix.
func requireHomelabRole() error {
	_, m, ok := config.SelfMachine()
	if !ok {
		return fmt.Errorf("no machine declared as self in ~/.jterrazz/config.json — register this Mac with role=homelab to use `j machine config`")
	}
	if m.Role != config.RoleHomelab {
		return fmt.Errorf("`j machine config` is homelab-only; current machine role is %q", m.Role)
	}
	return nil
}
