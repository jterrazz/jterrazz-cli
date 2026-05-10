package commands

import (
	configview "github.com/jterrazz/jterrazz-cli/src/internal/presentation/views/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure this machine (interactive TUI)",
	Long: "Open the configuration TUI. Lists every configurable item — terminal " +
		"setup, security, editor, system tweaks, and (on a homelab-registered " +
		"machine) homelab services. Items show their current state via CheckFn; " +
		"toggleable items run their disable action when already configured.",
	Run: func(cmd *cobra.Command, args []string) {
		configview.RunOrExit()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
