package config

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
)

// CheckResult is the unified result type for all check operations
type CheckResult struct {
	Installed bool   // Whether the item is installed/configured
	Version   string // Version string (if applicable)
	Status    string // Additional status: "running", "stopped", "3 versions", etc.
	Detail    string // Extra detail: path, config location, etc.
}

// CheckResult constructors for common patterns

// Installed creates a CheckResult for an installed item
func Installed() CheckResult {
	return CheckResult{Installed: true}
}

// InstalledWithVersion creates a CheckResult for an installed item with version
func InstalledWithVersion(version string) CheckResult {
	return CheckResult{Installed: true, Version: version}
}

// InstalledWithDetail creates a CheckResult for an installed item with detail
func InstalledWithDetail(detail string) CheckResult {
	return CheckResult{Installed: true, Detail: detail}
}

// InstalledWithStatus creates a CheckResult for an installed item with status
func InstalledWithStatus(version, status string) CheckResult {
	return CheckResult{Installed: true, Version: version, Status: status}
}

// NotInstalled creates a CheckResult for a not installed item
func NotInstalled() CheckResult {
	return CheckResult{}
}

// ToolCategory groups tools by their purpose
type ToolCategory string

const (
	CategoryPackageManager ToolCategory = "Package Managers"
	CategoryRuntimes       ToolCategory = "Runtimes"
	CategoryTerminal       ToolCategory = "Terminal"
	CategoryGit            ToolCategory = "Git"
	CategorySystem         ToolCategory = "System"
	CategoryDeploy         ToolCategory = "Deploy"
	CategoryAIAgents       ToolCategory = "AI Agents"
	CategoryAITooling      ToolCategory = "AI Tooling"
	CategoryDevelopment    ToolCategory = "Development"
	CategoryCreative       ToolCategory = "Creative"
	CategoryCommunication  ToolCategory = "Communication"
	CategoryProductivity   ToolCategory = "Productivity"
	CategoryAIApps         ToolCategory = "AI Apps"
	CategorySecurity       ToolCategory = "Security"
	CategoryBrowse         ToolCategory = "Browse"
	CategoryEntertainment  ToolCategory = "Entertainment"
	CategoryUtilities      ToolCategory = "Utilities"
)

// InstallMethod defines how a tool is installed
type InstallMethod string

const (
	InstallBrewFormula InstallMethod = "brew"
	InstallBrewCask    InstallMethod = "cask"
	InstallNpm         InstallMethod = "npm"
	InstallBun         InstallMethod = "bun"
	InstallUV          InstallMethod = "uv"
	InstallNvm         InstallMethod = "nvm"
	InstallXcode       InstallMethod = "xcode"
	InstallManual      InstallMethod = "manual"
	InstallMAS         InstallMethod = "mas"
)

// String returns a display string for the install method
func (m InstallMethod) String() string {
	switch m {
	case InstallBrewFormula, InstallBrewCask:
		return "brew"
	case InstallNpm:
		return "npm"
	case InstallBun:
		return "bun"
	case InstallUV:
		return "uv"
	case InstallNvm:
		return "nvm"
	case InstallXcode:
		return "xcode"
	case InstallManual:
		return "sh"
	case InstallMAS:
		return "mas"
	default:
		return "-"
	}
}

// Tool represents an installable piece of software
type Tool struct {
	Name        string
	Description string
	Category    ToolCategory

	// Check - how to verify if installed
	Command string             // CLI command to check existence
	CheckFn func() CheckResult // Custom check (overrides Command)

	// Install - how to install
	Method        InstallMethod // brew, npm, manual, etc.
	Formula       string        // Brew formula or npm package name
	PythonVersion string        // Python version constraint for uv tools (e.g. "3.13")
	InstallFn     func() error  // Custom install (overrides Method)
	Dependencies  []string      // Tool names this depends on

	// Version - how to get version info
	VersionFn func() string // Returns version string

	// Scripts - post-install or related scripts
	Scripts []string // Script names to run after install
}

// =============================================================================
// Tool Functions
// =============================================================================

// GetAllTools returns all tools
func GetAllTools() []Tool {
	return Tools
}

// GetToolsByCategory returns tools filtered by category
func GetToolsByCategory(category ToolCategory) []Tool {
	var result []Tool
	for _, t := range Tools {
		if t.Category == category {
			result = append(result, t)
		}
	}
	return result
}

// GetInstallableTools returns tools that can be installed
func GetInstallableTools() []Tool {
	var result []Tool
	for _, t := range Tools {
		if t.Method == InstallBrewFormula || t.Method == InstallBrewCask || t.Method == InstallNpm || t.Method == InstallBun || t.InstallFn != nil {
			result = append(result, t)
		}
	}
	return result
}

// GetToolByName returns a tool by name
func GetToolByName(name string) *Tool {
	for i := range Tools {
		if Tools[i].Name == name {
			return &Tools[i]
		}
	}
	return nil
}

// GetToolsInDependencyOrder returns all installable tools sorted by dependencies
func GetToolsInDependencyOrder() []Tool {
	installable := GetInstallableTools()

	toolMap := make(map[string]*Tool)
	for i := range installable {
		toolMap[installable[i].Name] = &installable[i]
	}

	visited := make(map[string]bool)
	var result []Tool

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}

		t := toolMap[name]
		if t == nil {
			t = GetToolByName(name)
		}
		if t == nil {
			return
		}

		for _, dep := range t.Dependencies {
			visit(dep)
		}

		visited[name] = true

		if toolMap[name] != nil {
			result = append(result, *t)
		}
	}

	for _, t := range installable {
		visit(t.Name)
	}

	return result
}

// Check checks if a tool is installed and returns its status
func (t Tool) Check() CheckResult {
	if t.CheckFn != nil {
		return t.CheckFn()
	}

	if t.Command == "" {
		return CheckResult{}
	}

	if _, err := exec.LookPath(t.Command); err != nil {
		return CheckResult{}
	}

	result := CheckResult{Installed: true}

	if t.VersionFn != nil {
		result.Version = t.VersionFn()
	}

	// Fallback: for bun packages, get version from bun global list if VersionFn returned nothing
	if result.Version == "" && t.Method == InstallBun && t.Formula != "" {
		result.Version = tool.VersionFromBunGlobal(t.Formula)()
	}

	return result
}

// Install installs the tool
func (t Tool) Install() error {
	if t.InstallFn != nil {
		return t.InstallFn()
	}

	switch t.Method {
	case InstallBrewFormula:
		return RunBrewCommand("install", t.Formula)
	case InstallBrewCask:
		return RunBrewCommand("install", "--cask", t.Formula)
	case InstallNpm:
		return ExecCommand("npm", "install", "-g", t.Formula)
	case InstallBun:
		return ExecCommand("bun", "install", "-g", t.Formula)
	case InstallUV:
		if t.PythonVersion != "" {
			return ExecCommand("uv", "tool", "install", t.Formula, "--python", t.PythonVersion)
		}
		return ExecCommand("uv", "tool", "install", t.Formula)
	default:
		return fmt.Errorf("cannot auto-install %s (method: %s)", t.Name, t.Method)
	}
}

// RunBrewCommand runs a brew command with ARM architecture forced
func RunBrewCommand(args ...string) error {
	cmd := exec.Command("arch", append([]string{"-arm64", "brew"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
