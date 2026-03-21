package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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

// Tools is the single source of truth for all installable software
var Tools = []Tool{
	// ==========================================================================
	// Package Managers
	// ==========================================================================
	{
		Name:         "bun",
		Command:      "bun",
		Formula:      "bun",
		Method:       InstallBrewFormula,
		Category:     CategoryPackageManager,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("bun", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:         "uv",
		Command:      "uv",
		Formula:      "uv",
		Method:       InstallBrewFormula,
		Category:     CategoryPackageManager,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("uv", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "cocoapods",
		Command:      "pod",
		Formula:      "cocoapods",
		Method:       InstallBrewFormula,
		Category:     CategoryPackageManager,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("pod", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:     "homebrew",
		Command:  "brew",
		Method:   InstallManual,
		Category: CategoryPackageManager,
		CheckFn: func() CheckResult {
			if _, err := exec.LookPath("brew"); err != nil {
				return CheckResult{}
			}
			out, _ := exec.Command("brew", "--version").Output()
			version := tool.ParseBrewVersion(string(out))
			formulaeOut, _ := exec.Command("brew", "list", "--formula", "-1").Output()
			caskOut, _ := exec.Command("brew", "list", "--cask", "-1").Output()
			formulaeCount := 0
			caskCount := 0
			if len(strings.TrimSpace(string(formulaeOut))) > 0 {
				formulaeCount = len(strings.Split(strings.TrimSpace(string(formulaeOut)), "\n"))
			}
			if len(strings.TrimSpace(string(caskOut))) > 0 {
				caskCount = len(strings.Split(strings.TrimSpace(string(caskOut)), "\n"))
			}
			return CheckResult{
				Installed: true,
				Version:   version,
				Status:    fmt.Sprintf("%d formulae, %d casks", formulaeCount, caskCount),
			}
		},
		InstallFn: func() error {
			cmd := exec.Command("/bin/bash", "-c", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		},
	},
	{
		Name:         "npm",
		Command:      "npm",
		Method:       InstallNvm,
		Category:     CategoryPackageManager,
		Dependencies: []string{"node"},
		CheckFn: func() CheckResult {
			if _, err := exec.LookPath("npm"); err != nil {
				return CheckResult{}
			}
			out, _ := exec.Command("npm", "--version").Output()
			version := tool.TrimVersion(string(out))
			npmOut, _ := exec.Command("npm", "list", "-g", "--depth=0", "--parseable").Output()
			npmLines := strings.Split(strings.TrimSpace(string(npmOut)), "\n")
			count := len(npmLines) - 1
			if count < 0 {
				count = 0
			}
			return CheckResult{
				Installed: true,
				Version:   version,
				Status:    fmt.Sprintf("%d global", count),
			}
		},
	},
	{
		Name:         "nvm",
		Command:      "",
		Formula:      "nvm",
		Method:       InstallBrewFormula,
		Category:     CategoryPackageManager,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			nvmDir := os.Getenv("HOME") + "/.nvm"
			if _, err := os.Stat(nvmDir); err != nil {
				return CheckResult{}
			}
			versionsDir := nvmDir + "/versions/node"
			entries, err := os.ReadDir(versionsDir)
			status := ""
			if err == nil {
				count := 0
				for _, e := range entries {
					if e.IsDir() && strings.HasPrefix(e.Name(), "v") {
						count++
					}
				}
				if count > 0 {
					status = fmt.Sprintf("%d versions", count)
				}
			}
			version := tool.VersionFromBrewFormula("nvm")()
			return CheckResult{Installed: true, Version: version, Status: status}
		},
	},
	{
		Name:         "pnpm",
		Command:      "pnpm",
		Formula:      "pnpm",
		Method:       InstallBrewFormula,
		Category:     CategoryPackageManager,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("pnpm", []string{"--version"}, tool.TrimVersion),
	},

	// ==========================================================================
	// Runtimes
	// ==========================================================================
	{
		Name:         "go",
		Command:      "go",
		Formula:      "go",
		Method:       InstallBrewFormula,
		Category:     CategoryRuntimes,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("go", []string{"version"}, tool.ParseGoVersion),
	},
	{
		Name:         "node",
		Command:      "node",
		Method:       InstallNvm,
		Category:     CategoryRuntimes,
		Dependencies: []string{"nvm"},
		VersionFn:    tool.VersionFromCmd("node", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:         "openjdk",
		Command:      "java",
		Formula:      "openjdk",
		Method:       InstallBrewFormula,
		Category:     CategoryRuntimes,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"java"},
		CheckFn: func() CheckResult {
			brewJava := "/opt/homebrew/opt/openjdk/bin/java"
			if _, err := os.Stat(brewJava); err == nil {
				out, _ := exec.Command(brewJava, "-version").CombinedOutput()
				return CheckResult{Installed: true, Version: tool.ParseJavaVersion(string(out))}
			}
			cmd := exec.Command("/usr/libexec/java_home")
			if err := cmd.Run(); err != nil {
				return CheckResult{}
			}
			out, _ := exec.Command("java", "-version").CombinedOutput()
			return CheckResult{Installed: true, Version: tool.ParseJavaVersion(string(out))}
		},
	},
	{
		Name:         "python",
		Command:      "python3",
		Method:       InstallManual,
		Category:     CategoryRuntimes,
		Dependencies: []string{"uv"},
		VersionFn:    tool.VersionFromCmd("python3", []string{"--version"}, tool.ParsePythonVersion),
		InstallFn: func() error {
			return ExecCommand("uv", "python", "install")
		},
	},
	{
		Name:         "rust",
		Command:      "rustc",
		Formula:      "rust",
		Method:       InstallBrewFormula,
		Category:     CategoryRuntimes,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("rustc", []string{"--version"}, tool.ParseRustVersion),
	},

	// ==========================================================================
	// Deploy
	// ==========================================================================
	{
		Name:         "ansible",
		Command:      "ansible",
		Formula:      "ansible",
		Method:       InstallBrewFormula,
		Category:     CategoryDeploy,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("ansible", []string{"--version"}, tool.ParseAnsibleVersion),
	},
	{
		Name:         "copier",
		Description:  "Project template engine with update support",
		Command:      "copier",
		Formula:      "copier",
		Method:       InstallBrewFormula,
		Category:     CategoryDeploy,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("copier", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:         "eas",
		Command:      "eas",
		Formula:      "eas-cli",
		Method:       InstallBun,
		Category:     CategoryDeploy,
		Dependencies: []string{"bun"},
		VersionFn:    tool.VersionFromCmd("eas", []string{"--version"}, tool.ParseEasVersion),
	},
	{
		Name:         "pulumi",
		Command:      "pulumi",
		Formula:      "pulumi/tap/pulumi",
		Method:       InstallBrewFormula,
		Category:     CategoryDeploy,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("pulumi", []string{"version"}, tool.ParsePulumiVersion),
	},
	{
		Name:         "terraform",
		Command:      "terraform",
		Formula:      "terraform",
		Method:       InstallBrewFormula,
		Category:     CategoryDeploy,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("terraform", []string{"--version"}, tool.ParseTerraformVersion),
	},

	// ==========================================================================
	// System
	// ==========================================================================
	{
		Name:         "mole",
		Command:      "mo",
		Formula:      "tw93/tap/mole",
		Method:       InstallBrewFormula,
		Category:     CategorySystem,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("mo", []string{"--version"}, tool.ParseMoleVersion),
	},
	{
		Name:         "multipass",
		Command:      "multipass",
		Formula:      "multipass",
		Method:       InstallBrewFormula,
		Category:     CategorySystem,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("multipass", []string{"--version"}, tool.ParseMultipassVersion),
	},

	// ==========================================================================
	// AI Agents
	// ==========================================================================
	{
		Name:      "claude",
		Command:   "claude",
		Method:    InstallManual,
		Category:  CategoryAIAgents,
		VersionFn: tool.VersionFromCmd("claude", []string{"--version"}, tool.ParseClaudeVersion),
		InstallFn: func() error {
			cmd := exec.Command("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		},
	},
	{
		Name:         "claude-agent-acp",
		Command:      "claude-agent-acp",
		Formula:      "@zed-industries/claude-agent-acp",
		Method:       InstallBun,
		Category:     CategoryAIAgents,
		Dependencies: []string{"bun"},
	},
	{
		Name:         "codex",
		Command:      "codex",
		Formula:      "codex",
		Method:       InstallBun,
		Category:     CategoryAIAgents,
		Dependencies: []string{"bun"},
		VersionFn:    tool.VersionFromCmd("codex", []string{"--version"}, tool.ParseCodexVersion),
	},
	{
		Name:         "gemini",
		Command:      "gemini",
		Formula:      "gemini-cli",
		Method:       InstallBrewFormula,
		Category:     CategoryAIAgents,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("gemini", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:         "opencode",
		Command:      "opencode",
		Formula:      "opencode",
		Method:       InstallBrewFormula,
		Category:     CategoryAIAgents,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("opencode", []string{"--version"}, tool.TrimVersion),
	},

	// ==========================================================================
	// AI Tooling
	// ==========================================================================
	{
		Name:         "ollama",
		Command:      "ollama",
		Formula:      "ollama-app",
		Method:       InstallBrewCask,
		Category:     CategoryAITooling,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			_, appErr := os.Stat("/Applications/Ollama.app")
			if appErr != nil {
				return CheckResult{}
			}
			version := tool.VersionFromBrewCask("ollama-app")()
			status := "stopped"
			if err := exec.Command("pgrep", "-x", "ollama").Run(); err == nil {
				status = "running"
			}
			return CheckResult{Installed: true, Version: version, Status: status}
		},
	},
	{
		Name:         "qmd",
		Command:      "qmd",
		Formula:      "https://github.com/tobi/qmd",
		Method:       InstallBun,
		Category:     CategoryAITooling,
		Dependencies: []string{"bun"},
		VersionFn:    tool.VersionFromCmd("qmd", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:         "rtk",
		Command:      "rtk",
		Formula:      "rtk",
		Method:       InstallBrewFormula,
		Category:     CategoryAITooling,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("rtk", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:         "skills",
		Command:      "skills",
		Formula:      "skills",
		Method:       InstallBun,
		Category:     CategoryAITooling,
		Dependencies: []string{"bun"},
		VersionFn:    tool.VersionFromCmd("skills", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:          "browser-use",
		Command:       "browser-use",
		Formula:       "browser-use",
		Method:        InstallUV,
		PythonVersion: "3.13",
		Category:      CategoryAITooling,
		Dependencies:  []string{"uv"},
		VersionFn:     tool.VersionFromCmd("browser-use", []string{"--version"}, tool.TrimVersion),
	},
	{
		Name:          "markitdown",
		Description:   "Convert files to Markdown for LLMs",
		Command:       "markitdown",
		Formula:       "markitdown[all]",
		Method:        InstallUV,
		PythonVersion: "3.13",
		Category:      CategoryAITooling,
		Dependencies:  []string{"uv"},
	},
	{
		Name:         "playwright-mcp",
		Description:  "Browser automation for AI agents via MCP",
		Command:      "npx",
		Formula:      "@playwright/mcp",
		Method:       InstallNpm,
		Category:     CategoryAITooling,
		Dependencies: []string{"node"},
		Scripts:      []string{"claude"},
	},
	{
		Name:         "agent-browser",
		Command:      "agent-browser",
		Formula:      "agent-browser",
		Method:       InstallBun,
		Category:     CategoryAITooling,
		Dependencies: []string{"bun"},
		VersionFn:    tool.VersionFromCmd("agent-browser", []string{"--version"}, tool.TrimVersion),
	},

	// ==========================================================================
	// GUI Apps + desktop tooling
	// ==========================================================================
	{
		Name:         "orbstack",
		Description:  "OrbStack container runtime (provides docker CLI)",
		Formula:      "orbstack",
		Method:       InstallBrewCask,
		Category:     CategorySystem,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/OrbStack.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("OrbStack")()
			status := "stopped"
			if err := exec.Command("docker", "info").Run(); err == nil {
				status = "running"
			}
			return CheckResult{Installed: true, Version: version, Status: status}
		},
	},
	{
		Name:         "conductor",
		Formula:      "conductor",
		Method:       InstallBrewCask,
		Category:     CategoryAIAgents,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Conductor.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromBrewCask("conductor")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "ghostty",
		Formula:      "ghostty",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"ghostty"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Ghostty.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromBrewCask("ghostty")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "gpg",
		Description:  "GNU Privacy Guard for encryption and signing",
		Command:      "gpg",
		Formula:      "gnupg",
		Method:       InstallBrewFormula,
		Category:     CategoryGit,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"gpg"},
		VersionFn:    tool.VersionFromBrewFormula("gnupg"),
	},
	{
		Name:        "ohmyzsh",
		Description: "Oh My Zsh shell framework",
		Command:     "",
		Method:      InstallManual,
		Category:    CategoryTerminal,
		CheckFn: func() CheckResult {
			omzPath := os.Getenv("HOME") + "/.oh-my-zsh"
			if _, err := os.Stat(omzPath); err != nil {
				return CheckResult{}
			}
			// Get git commit hash as version
			cmd := exec.Command("git", "-C", omzPath, "rev-parse", "--short", "HEAD")
			out, err := cmd.Output()
			version := ""
			if err == nil {
				version = strings.TrimSpace(string(out))
			}
			return CheckResult{Installed: true, Version: version}
		},
		InstallFn: func() error {
			cmd := exec.Command("sh", "-c", "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		},
	},
	{
		Name:         "lens",
		Formula:      "lens",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Lens.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromBrewCask("lens")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "zed",
		Description:  "Zed code editor",
		Formula:      "zed",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"zed"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Zed.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Zed")()
			return CheckResult{Installed: true, Version: version}
		},
	},

	{
		Name:         "android-studio",
		Description:  "Android development IDE",
		Formula:      "android-studio",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Android Studio.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Android Studio")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "bitwarden",
		Description:  "Password manager",
		Formula:      "bitwarden",
		Method:       InstallBrewCask,
		Category:     CategorySecurity,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Bitwarden.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Bitwarden")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "brave",
		Description:  "Privacy-focused web browser",
		Formula:      "brave-browser",
		Method:       InstallBrewCask,
		Category:     CategoryBrowse,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Brave Browser.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Brave Browser")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "chatgpt",
		Description:  "OpenAI ChatGPT desktop app",
		Formula:      "chatgpt",
		Method:       InstallBrewCask,
		Category:     CategoryAIApps,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/ChatGPT.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("ChatGPT")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "claude-desktop",
		Description:  "Anthropic Claude desktop app",
		Formula:      "claude",
		Method:       InstallBrewCask,
		Category:     CategoryAIApps,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Claude.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Claude")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "cursor",
		Description:  "AI-powered code editor",
		Formula:      "cursor",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Cursor.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Cursor")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "discord",
		Description:  "Voice and text chat",
		Formula:      "discord",
		Method:       InstallBrewCask,
		Category:     CategoryCommunication,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Discord.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Discord")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "linear",
		Description:  "Project management tool",
		Formula:      "linear-linear",
		Method:       InstallBrewCask,
		Category:     CategoryProductivity,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Linear.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Linear")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "notion",
		Description:  "Workspace for notes and docs",
		Formula:      "notion",
		Method:       InstallBrewCask,
		Category:     CategoryProductivity,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Notion.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Notion")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "obsidian",
		Description:  "Knowledge base and note-taking",
		Formula:      "obsidian",
		Method:       InstallBrewCask,
		Category:     CategoryProductivity,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Obsidian.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Obsidian")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "slack",
		Description:  "Team communication",
		Formula:      "slack",
		Method:       InstallBrewCask,
		Category:     CategoryCommunication,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Slack.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Slack")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:         "tailscale",
		Description:  "Mesh VPN built on WireGuard",
		Command:      "tailscale",
		Formula:      "tailscale",
		Method:       InstallBrewFormula,
		Category:     CategorySecurity,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := exec.LookPath("tailscale"); err == nil {
				version := tool.VersionFromCmd("tailscale", []string{"version"}, tool.ParseTailscaleVersion)()
				status := "installed"
				if out, err := exec.Command("tailscale", "status", "--json").Output(); err == nil {
					if strings.Contains(string(out), `"BackendState":"Running"`) {
						status = "running"
					}
				}
				return CheckResult{Installed: true, Version: version, Status: status}
			}
			if _, err := os.Stat("/Applications/Tailscale.app"); err == nil {
				version := tool.VersionFromAppPlist("Tailscale")()
				// App exists but CLI is not on PATH; keep installable to provide `tailscale`/`tailscaled`.
				return CheckResult{Installed: false, Version: version, Status: "app only"}
			}
			return CheckResult{}
		},
	},
	{
		Name:         "whatsapp",
		Description:  "Messaging app",
		Formula:      "whatsapp",
		Method:       InstallBrewCask,
		Category:     CategoryCommunication,
		Dependencies: []string{"homebrew"},
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/WhatsApp.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("WhatsApp")()
			return CheckResult{Installed: true, Version: version}
		},
	},

	// ==========================================================================
	// Terminal
	// ==========================================================================
	{
		Name:         "tmux",
		Command:      "tmux",
		Formula:      "tmux",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"tmux"},
		VersionFn:    tool.VersionFromCmd("tmux", []string{"-V"}, tool.ParseTmuxVersion),
	},
	{
		Name:         "bat",
		Description:  "Cat clone with syntax highlighting",
		Command:      "bat",
		Formula:      "bat",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("bat", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "dust",
		Description:  "Intuitive disk usage tool",
		Command:      "dust",
		Formula:      "dust",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("dust", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "eza",
		Description:  "Modern ls replacement",
		Command:      "eza",
		Formula:      "eza",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("eza", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "fd",
		Description:  "Fast find alternative",
		Command:      "fd",
		Formula:      "fd",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("fd", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "ripgrep",
		Description:  "Fast grep alternative",
		Command:      "rg",
		Formula:      "ripgrep",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("rg", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "sd",
		Description:  "Intuitive sed alternative",
		Command:      "sd",
		Formula:      "sd",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("sd", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "zoxide",
		Description:  "Smarter cd command",
		Command:      "zoxide",
		Formula:      "zoxide",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("zoxide", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "difftastic",
		Description:  "Structural diff tool",
		Command:      "difft",
		Formula:      "difftastic",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("difft", []string{"--version"}, tool.ParseBrewVersion),
	},

	// ==========================================================================
	// Git
	// ==========================================================================
	{
		Name:      "git",
		Command:   "git",
		Method:    InstallXcode,
		Category:  CategoryGit,
		VersionFn: tool.VersionFromCmd("git", []string{"--version"}, tool.ParseGitVersion),
	},
	{
		Name:         "gh",
		Description:  "GitHub CLI for repository management",
		Command:      "gh",
		Formula:      "gh",
		Method:       InstallBrewFormula,
		Category:     CategoryGit,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"gh"},
		VersionFn:    tool.VersionFromCmd("gh", []string{"--version"}, tool.ParseGhVersion),
	},

	// ==========================================================================
	// Mac App Store (check-only, not auto-installable)
	// ==========================================================================
	{
		Name:        "adguard",
		Description: "Ad blocker for Safari",
		Method:      InstallMAS,
		Category:    CategorySecurity,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/AdGuard for Safari.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("AdGuard for Safari")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:     "broadcasts",
		Method:   InstallMAS,
		Category: CategoryEntertainment,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Broadcasts.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Broadcasts")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "compressor",
		Description: "Apple video compression tool",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Compressor.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Compressor")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "dia",
		Description: "AI assistant by Apple",
		Method:      InstallMAS,
		Category:    CategoryBrowse,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Dia.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Dia")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "final-cut-pro",
		Description: "Professional video editor",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Final Cut Pro.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Final Cut Pro")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "lightroom",
		Description: "Adobe photo editor",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Adobe Lightroom.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Adobe Lightroom")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "logic-pro",
		Description: "Professional music production",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Logic Pro.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Logic Pro")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "messenger",
		Description: "Facebook Messenger",
		Method:      InstallMAS,
		Category:    CategoryCommunication,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Messenger.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Messenger")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "keynote",
		Description: "Apple presentations",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Keynote.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Keynote")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "numbers",
		Description: "Apple spreadsheets",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Numbers.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Numbers")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "pages",
		Description: "Apple word processor",
		Method:      InstallMAS,
		Category:    CategoryCreative,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Pages.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Pages")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "passepartout",
		Description: "VPN client",
		Method:      InstallMAS,
		Category:    CategorySecurity,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Passepartout.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Passepartout")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "pipifier",
		Description: "Picture-in-Picture for Safari",
		Method:      InstallMAS,
		Category:    CategoryUtilities,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/PiPifier.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("PiPifier")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "raindrop",
		Description: "Bookmark manager",
		Method:      InstallMAS,
		Category:    CategoryEntertainment,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Save to Raindrop.io.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Save to Raindrop.io")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "snippety",
		Description: "Code snippet manager",
		Method:      InstallMAS,
		Category:    CategoryUtilities,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Snippety.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Snippety")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "speedtest",
		Description: "Internet speed test",
		Method:      InstallMAS,
		Category:    CategoryUtilities,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Speedtest.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Speedtest")()
			return CheckResult{Installed: true, Version: version}
		},
	},
	{
		Name:        "xcode",
		Description: "Apple development IDE",
		Method:      InstallMAS,
		Category:    CategoryDevelopment,
		CheckFn: func() CheckResult {
			if _, err := os.Stat("/Applications/Xcode.app"); err != nil {
				return CheckResult{}
			}
			version := tool.VersionFromAppPlist("Xcode")()
			return CheckResult{Installed: true, Version: version}
		},
	},
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
	for _, tool := range Tools {
		if tool.Category == category {
			result = append(result, tool)
		}
	}
	return result
}

// GetInstallableTools returns tools that can be installed
func GetInstallableTools() []Tool {
	var result []Tool
	for _, tool := range Tools {
		if tool.Method == InstallBrewFormula || tool.Method == InstallBrewCask || tool.Method == InstallNpm || tool.Method == InstallBun || tool.InstallFn != nil {
			result = append(result, tool)
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

		tool := toolMap[name]
		if tool == nil {
			tool = GetToolByName(name)
		}
		if tool == nil {
			return
		}

		for _, dep := range tool.Dependencies {
			visit(dep)
		}

		visited[name] = true

		if toolMap[name] != nil {
			result = append(result, *tool)
		}
	}

	for _, tool := range installable {
		visit(tool.Name)
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
