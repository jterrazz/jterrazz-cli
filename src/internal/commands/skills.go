package commands

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/skill"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/views/skills"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage AI agent skills (interactive TUI)",
	Run: func(cmd *cobra.Command, args []string) {
		if !skill.IsInstalled() {
			print.Error("skills CLI not installed. Run: npm install -g skills")
			return
		}
		skillsview.InitSkillsState()
		components.RunOrExit(skillsview.SkillsConfig())
	},
}

func init() {
	rootCmd.AddCommand(skillsCmd)
}
