package commands

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	setupview "github.com/jterrazz/jterrazz-cli/src/internal/presentation/views/setup"
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
		setupview.RunOrExit(runScript)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// runScript runs a script by name. For toggleable items (UninstallFn != nil),
// runs UninstallFn when the item is currently installed, InstallFn otherwise.
func runScript(name string) {
	script := config.GetScriptByName(name)
	if script == nil {
		print.Error("Unknown script: " + name)
		return
	}

	fn := script.InstallFn
	verb := "install"
	if script.UninstallFn != nil && script.CheckFn != nil && script.CheckFn().Installed {
		fn = script.UninstallFn
		verb = "uninstall"
	}
	if fn == nil {
		print.Error("No runner for script: " + name)
		return
	}

	if err := fn(); err != nil {
		print.Error("Failed to " + verb + " " + name + ": " + err.Error())
	}
}

