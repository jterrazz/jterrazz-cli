package commands

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/skill"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	setupview "github.com/jterrazz/jterrazz-cli/src/internal/presentation/views/setup"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup system configurations (interactive)",
	Run: func(cmd *cobra.Command, args []string) {
		setupview.RunOrExit(runScript)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

// runScript runs a script by name
func runScript(name string) {
	script := config.GetScriptByName(name)
	if script == nil {
		print.Error("Unknown script: " + name)
		return
	}

	if script.RunFn == nil {
		print.Error("No runner for script: " + name)
		return
	}

	if err := script.RunFn(); err != nil {
		print.Error("Failed to run " + name + ": " + err.Error())
	}
}

// runSkillsUI runs the skills management UI
func runSkillsUI() {
	if !skill.IsInstalled() {
		print.Error("skills CLI not installed. Run: npm install -g skills")
		return
	}

	setupview.InitSkillsState()
	components.RunOrExit(setupview.SkillsConfig())
}
