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

// runScript runs a script by name from the legacy setup view. The new j config
// view dispatches actions directly via configview, but this callback is still
// used by setup/skills/remote sub-views until they're migrated.
//
// For toggleable items (UninstallFn != nil), runs UninstallFn when the item is
// currently installed, InstallFn otherwise. Inputs are not collected here —
// scripts that need them must be run from the new j config TUI.
func runScript(name string) {
	script := config.GetScriptByName(name)
	if script == nil {
		print.Error("Unknown script: " + name)
		return
	}

	if script.UninstallFn != nil && script.CheckFn != nil && script.CheckFn().Installed {
		if err := script.UninstallFn(); err != nil {
			print.Error("Failed to uninstall " + name + ": " + err.Error())
		}
		return
	}
	if script.InstallFn == nil {
		print.Error("No runner for script: " + name)
		return
	}
	if err := script.InstallFn(config.InputValues{}); err != nil {
		print.Error("Failed to install " + name + ": " + err.Error())
	}
}

