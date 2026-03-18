package commands

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var upgradeFlags = make(map[string]*bool)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [package...]",
	Short: "Upgrade system packages",
	Long: `Upgrade system packages.

Examples:
  j upgrade --all             Upgrade all package managers
  j upgrade --brew            Upgrade Homebrew packages only
  j upgrade --npm             Upgrade npm global packages only
  j upgrade --bun             Upgrade bun global packages only
  j upgrade node              Upgrade specific brew package
  j upgrade claude opencode   Upgrade specific packages
  j upgrade                   List available options`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var all []string
		for _, pkg := range config.Tools {
			if pkg.Method == config.InstallBrewFormula || pkg.Method == config.InstallBrewCask {
				all = append(all, pkg.Name)
			}
		}
		return tool.FilterStrings(all, args), cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Check for --all flag
		allFlag, _ := cmd.Flags().GetBool("all")
		if allFlag {
			print.Action("🔄", "Upgrading all packages...")
			config.UpgradeAll()
			print.Done("All upgrades completed")
			return
		}

		// Check for specific manager flags
		anyFlagSet := false
		for _, pm := range config.PackageManagers {
			if flagVal, ok := upgradeFlags[pm.Flag]; ok && *flagVal {
				anyFlagSet = true
				config.UpgradePackageManager(pm)
			}
		}
		if anyFlagSet {
			print.Done("Upgrades completed")
			return
		}

		// If specific package names provided
		if len(args) > 0 {
			print.Action("🔄", "Upgrading selected packages...")
			for _, name := range args {
				if err := config.UpgradePackageByName(name); err != nil {
					print.Error(err.Error())
				}
			}
			print.Done("Upgrades completed")
			return
		}

		// No args, list options
		listUpgradeOptions()
	},
}

func init() {
	upgradeCmd.Flags().BoolP("all", "a", false, "Upgrade all package managers")

	// Dynamically add flags for each package manager
	for _, pm := range config.PackageManagers {
		flagPtr := new(bool)
		upgradeFlags[pm.Flag] = flagPtr
		upgradeCmd.Flags().BoolVar(flagPtr, pm.Flag, false, "Upgrade "+pm.Name+" packages")
	}

	rootCmd.AddCommand(upgradeCmd)
}

func listUpgradeOptions() {
	print.SectionDivider("Available Upgrade Targets")

	for _, pm := range config.PackageManagers {
		available := config.CommandExists(pm.RequiresCmd)
		print.Row(available, pm.Name, "--"+pm.Flag)
	}

	print.Empty()
	print.Usage(
		"Usage: j upgrade <package> [package...]",
		"       j upgrade --brew --bun",
		"       j upgrade --all",
	)
}
