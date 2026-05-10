package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var machineInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Register this machine in ~/.jterrazz/config.json (interactive)",
	Long: strings.TrimSpace(`Interactive bootstrap of the machine registry.

Prompts for an alias (default = hostname) and a role (dev or homelab),
adds the entry to ~/.jterrazz/config.json, and marks it as self. Run this
once per machine you own.`),
	Run: func(cmd *cobra.Command, args []string) { runMachineInit() },
}

var machineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered machines",
	Run:   func(cmd *cobra.Command, args []string) { runMachineList() },
}

var (
	machineAddRole     string
	machineAddSSH      string
	machineAddIdentity string
)

var machineAddCmd = &cobra.Command{
	Use:   "add <alias>",
	Short: "Add a machine to the registry",
	Long: strings.TrimSpace(`Add a remote (or local) machine to the registry.

When --ssh is set, also writes a managed Host block in ~/.ssh/config so the
alias is reachable via the same name from this CLI's other commands.`),
	Args: cobra.ExactArgs(1),
	Run:  func(cmd *cobra.Command, args []string) { runMachineAdd(args[0]) },
}

var machineRemoveCmd = &cobra.Command{
	Use:     "remove <alias>",
	Aliases: []string{"rm"},
	Short:   "Remove a machine from the registry",
	Args:    cobra.ExactArgs(1),
	Run:     func(cmd *cobra.Command, args []string) { runMachineRemove(args[0]) },
}

func init() {
	machineAddCmd.Flags().StringVar(&machineAddRole, "role", "", "machine role: dev or homelab (required)")
	machineAddCmd.Flags().StringVar(&machineAddSSH, "ssh", "", "ssh endpoint user@host (required for remote machines)")
	machineAddCmd.Flags().StringVar(&machineAddIdentity, "identity", "", "ssh identity file path (default ~/.ssh/id_ed25519)")
	machineCmd.AddCommand(machineInitCmd, machineListCmd, machineAddCmd, machineRemoveCmd)
}

func runMachineInit() {
	print.SectionDivider("MACHINE INIT")

	if alias, m, ok := config.SelfMachine(); ok {
		print.Warning(fmt.Sprintf("This machine is already registered as %q (role=%s)", alias, m.Role))
		print.Dim("Edit ~/.jterrazz/config.json directly, or remove + re-init to change.")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	defaultAlias := hostname()
	fmt.Printf("Alias for this machine [%s]: ", defaultAlias)
	line, _ := reader.ReadString('\n')
	alias := strings.TrimSpace(line)
	if alias == "" {
		alias = defaultAlias
	}

	fmt.Print("Role [dev/homelab]: ")
	line, _ = reader.ReadString('\n')
	role := config.Role(strings.TrimSpace(strings.ToLower(line)))

	m := config.Machine{Role: role}
	if err := config.AddMachine(alias, m); err != nil {
		failOn(err)
	}
	if err := config.SetSelf(alias); err != nil {
		_ = config.RemoveMachine(alias)
		failOn(err)
	}
	print.Success(fmt.Sprintf("Registered %q as self (role=%s)", alias, role))
}

func runMachineList() {
	machines := config.ListMachines()
	selfAlias, _, _ := config.SelfMachine()

	print.SectionDivider("MACHINES")
	if len(machines) == 0 {
		print.Dim("No machines registered. Run `j machine init` for this machine, or `j machine add <alias>` for a remote.")
		return
	}

	fmt.Printf("  %-15s %-10s %-32s %s\n", "ALIAS", "ROLE", "ENDPOINT", "SELF")
	for _, entry := range machines {
		endpoint := entry.Machine.SSH
		if endpoint == "" {
			endpoint = "(local)"
		}
		self := ""
		if entry.Alias == selfAlias {
			self = "*"
		}
		fmt.Printf("  %-15s %-10s %-32s %s\n", entry.Alias, entry.Machine.Role, endpoint, self)
	}
}

func runMachineAdd(alias string) {
	if machineAddRole == "" {
		failOn(fmt.Errorf("--role is required (dev or homelab)"))
	}
	m := config.Machine{
		Role:     config.Role(machineAddRole),
		SSH:      machineAddSSH,
		Identity: machineAddIdentity,
	}
	if err := config.AddMachine(alias, m); err != nil {
		failOn(err)
	}
	print.Success(fmt.Sprintf("Added machine %q (role=%s)", alias, m.Role))

	if m.SSH != "" {
		if err := config.WriteSSHAlias(alias, m); err != nil {
			// Rollback so the registry and ~/.ssh/config never disagree.
			_ = config.RemoveMachine(alias)
			failOn(fmt.Errorf("ssh config update failed (registry rolled back): %w", err))
		}
		print.Success("Wrote ~/.ssh/config block for " + alias)
	}
}

func runMachineRemove(alias string) {
	m, ok := config.GetMachine(alias)
	if !ok {
		failOn(fmt.Errorf("machine %q not found", alias))
	}
	if m.SSH != "" {
		if err := config.RemoveSSHAlias(alias); err != nil {
			print.Warning("ssh config cleanup failed: " + err.Error())
		}
	}
	if err := config.RemoveMachine(alias); err != nil {
		failOn(err)
	}
	print.Success(fmt.Sprintf("Removed machine %q", alias))
}
