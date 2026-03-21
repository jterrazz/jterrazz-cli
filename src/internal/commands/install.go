package commands

import (
	"fmt"
	"sync"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [tool...]",
	Short: "Install development tools",
	Long: `Install development tools.

Examples:
  j install homebrew        Install Homebrew
  j install nvm             Install NVM
  j install go python node  Install specific tools
  j install                 List available tools`,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var all []string
		for _, t := range config.Tools {
			all = append(all, t.Name)
		}
		return tool.FilterStrings(all, args), cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			listAvailableTools()
			return
		}

		print.Action("📦", "Installing selected tools...")
		for _, name := range args {
			installToolByName(name)
		}
		print.Done("Done")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func listAvailableTools() {
	print.SectionDivider("Available Tools")

	// Check all tools in parallel
	results := make(map[string]config.CheckResult, len(config.Tools))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range config.Tools {
		wg.Add(1)
		go func(t *config.Tool) {
			defer wg.Done()
			result := t.Check()
			mu.Lock()
			results[t.Name] = result
			mu.Unlock()
		}(&config.Tools[i])
	}
	wg.Wait()

	knownCategories := make(map[config.ToolCategory]bool, len(config.ToolCategories))
	for _, category := range config.ToolCategories {
		knownCategories[category] = true
		tools := config.GetToolsByCategory(category)
		if len(tools) == 0 {
			continue
		}

		print.Category(string(category))
		for _, t := range tools {
			print.Row(results[t.Name].Installed, t.Name, t.Method.String())
		}
	}

	// Fallback: show any tools using categories not listed in ToolCategories.
	currentCategory := config.ToolCategory("")
	for _, t := range config.Tools {
		if knownCategories[t.Category] {
			continue
		}
		if t.Category != currentCategory {
			currentCategory = t.Category
			print.Category(string(currentCategory))
		}
		print.Row(results[t.Name].Installed, t.Name, t.Method.String())
	}

	print.Empty()
	print.Usage("Usage: j install <tool> [tool...]")
}

func installToolByName(name string) {
	// Handle "brew" as alias for "homebrew"
	if name == "brew" {
		name = "homebrew"
	}

	t := config.GetToolByName(name)
	if t == nil {
		print.Error("Unknown tool: " + name)
		return
	}

	result := t.Check()
	if result.Installed {
		print.Row(true, t.Name, "already installed")
		return
	}

	// Check dependencies
	for _, depName := range t.Dependencies {
		depTool := config.GetToolByName(depName)
		if depTool == nil {
			continue
		}
		depResult := depTool.Check()
		if !depResult.Installed {
			print.Error(depName + " required for " + t.Name + ". Run: j install " + depName)
			return
		}
	}

	print.Installing(t.Name)
	if err := t.Install(); err != nil {
		print.Error(fmt.Sprintf("Failed to install %s: %v", t.Name, err))
	} else {
		print.Row(true, t.Name, "installed")
		// Run post-install scripts
		for _, scriptName := range t.Scripts {
			runScript(scriptName)
		}
	}
}
