package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var syncAllFlag bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync project with copier templates",
	Long: `Sync the current project with copier templates.

Running without a subcommand updates the current project from its template.

Examples:
  j sync              Update current project from its template
  j sync init         Initialize a project from a template
  j sync status       Show template link status
  j sync diff         Preview changes before updating`,
	Run: func(cmd *cobra.Command, args []string) {
		if syncAllFlag {
			syncAllProjects()
			return
		}
		syncUpdate()
	},
}

var syncInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project from template",
	Run: func(cmd *cobra.Command, args []string) {
		syncInit()
	},
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show template link status",
	Run: func(cmd *cobra.Command, args []string) {
		syncStatus()
	},
}

var syncDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Preview changes before updating",
	Run: func(cmd *cobra.Command, args []string) {
		syncDiff()
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncAllFlag, "all", false, "Update all projects in ~/Developer that use copier")
	syncCmd.AddCommand(syncInitCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncDiffCmd)
	rootCmd.AddCommand(syncCmd)
}

// getTemplatePath returns the local path to the copier templates directory
func getTemplatePath() (string, error) {
	return config.GetRepoConfigPath("dotfiles/blueprints")
}

// hasCopierAnswers checks if the current directory has a .copier-answers.yml file
func hasCopierAnswers() bool {
	_, err := os.Stat(".copier-answers.yml")
	return err == nil
}

// requireCopier checks that copier is installed and exits with a message if not
func requireCopier() bool {
	if _, err := exec.LookPath("copier"); err != nil {
		print.Error("copier not installed. Run: j install copier")
		return false
	}
	return true
}

// detectLanguage tries to detect the project language from files in the current directory
func detectLanguage() string {
	if _, err := os.Stat("go.mod"); err == nil {
		return "go"
	}
	if _, err := os.Stat("package.json"); err == nil {
		return "typescript"
	}
	return ""
}

// syncUpdate runs copier update on the current project
func syncUpdate() {
	if !hasCopierAnswers() {
		print.Warning("No .copier-answers.yml found in current directory")
		print.Dim("Run 'j sync init' to initialize this project from a template")
		return
	}

	if !requireCopier() {
		return
	}

	print.Action("🔄", "Updating project from template...")

	cmd := exec.Command("copier", "update", "--trust")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		print.Error("Update failed: " + err.Error())
		return
	}

	print.Done("Project updated")
}

// syncInit initializes a project from the copier template
func syncInit() {
	if !requireCopier() {
		return
	}

	if hasCopierAnswers() {
		print.Warning("Project already linked to a template (.copier-answers.yml exists)")
		print.Dim("Run 'j sync' to update instead")
		return
	}

	templatePath, err := getTemplatePath()
	if err != nil {
		print.Error("Template not found: " + err.Error())
		print.Dim("Make sure jterrazz-studio is cloned at ~/Developer/jterrazz/jterrazz-studio (or set J_REPO_PATH)")
		return
	}

	// Auto-detect language and show it
	lang := detectLanguage()
	if lang != "" {
		print.Dim("Detected language: " + lang)
	}

	print.Action("📋", "Initializing project from template...")
	print.Dim("Source: " + templatePath)
	print.Empty()

	args := []string{"copy", "--trust", templatePath, "."}
	if lang != "" {
		args = []string{"copy", "--trust", "--data", fmt.Sprintf("language=%s", lang), templatePath, "."}
	}

	cmd := exec.Command("copier", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		print.Error("Init failed: " + err.Error())
		return
	}

	print.Empty()
	print.Done("Project initialized from template")
	print.Dim("Run 'j sync' anytime to pull template updates")
}

// syncStatus shows the template link status for the current project
func syncStatus() {
	if !hasCopierAnswers() {
		print.Row(false, "Not linked", "no .copier-answers.yml")
		print.Empty()
		print.Dim("Run 'j sync init' to link this project to a template")
		return
	}

	// Read and display the copier answers
	data, err := os.ReadFile(".copier-answers.yml")
	if err != nil {
		print.Error("Failed to read .copier-answers.yml: " + err.Error())
		return
	}

	print.Row(true, "Linked", ".copier-answers.yml")
	print.Empty()

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Show key-value pairs
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Skip internal copier keys starting with _
			if strings.HasPrefix(key, "_") {
				print.Dim(fmt.Sprintf("  %s: %s", key, val))
			} else {
				fmt.Printf("  %-16s %s\n", key+":", val)
			}
		}
	}
	print.Empty()
}

// syncDiff previews what would change on the next update
func syncDiff() {
	if !hasCopierAnswers() {
		print.Warning("No .copier-answers.yml found in current directory")
		print.Dim("Run 'j sync init' to initialize this project from a template")
		return
	}

	if !requireCopier() {
		return
	}

	print.Action("🔍", "Previewing template changes...")
	print.Empty()

	cmd := exec.Command("copier", "update", "--pretend", "--diff", "--trust")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		print.Error("Diff failed: " + err.Error())
		return
	}
}

// syncAllProjects finds all projects in ~/Developer with .copier-answers.yml and updates them
func syncAllProjects() {
	if !requireCopier() {
		return
	}

	devDir := os.Getenv("HOME") + "/Developer"
	entries, err := os.ReadDir(devDir)
	if err != nil {
		print.Error("Failed to read ~/Developer: " + err.Error())
		return
	}

	var projects []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		answersPath := filepath.Join(devDir, entry.Name(), ".copier-answers.yml")
		if _, err := os.Stat(answersPath); err == nil {
			projects = append(projects, entry.Name())
		}
	}

	if len(projects) == 0 {
		print.Dim("No projects with .copier-answers.yml found in ~/Developer")
		return
	}

	print.Action("🔄", fmt.Sprintf("Updating %d projects...", len(projects)))
	print.Empty()

	for _, name := range projects {
		projectDir := filepath.Join(devDir, name)
		print.Info(name)

		cmd := exec.Command("copier", "update", "--trust")
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			print.Error("  Failed: " + err.Error())
		} else {
			print.Success("  Updated")
		}
		print.Empty()
	}

	print.Done(fmt.Sprintf("Updated %d projects", len(projects)))
}
