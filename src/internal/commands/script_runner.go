package commands

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

// runScript runs a script by name from a non-TUI context — currently used by
// `j install` for post-install scripts attached to a Tool. The j config TUI
// dispatches actions itself (with proper Inputs collection); this helper is
// the headless fallback and ignores any Inputs the script declares.
//
// For toggleable items (UninstallFn != nil), runs UninstallFn when the item is
// currently installed, InstallFn otherwise.
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
