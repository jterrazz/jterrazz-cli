package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
)

// checkApp returns a CheckFn for a macOS .app bundle, using the plist for version info.
func checkApp(appName string) func() CheckResult {
	return func() CheckResult {
		if _, err := os.Stat("/Applications/" + appName + ".app"); err != nil {
			return CheckResult{}
		}
		version := tool.VersionFromAppPlist(appName)()
		return CheckResult{Installed: true, Version: version}
	}
}

// checkAppWithCask is like checkApp but uses brew cask for version info.
func checkAppWithCask(appName, caskName string) func() CheckResult {
	return func() CheckResult {
		if _, err := os.Stat("/Applications/" + appName + ".app"); err != nil {
			return CheckResult{}
		}
		version := tool.VersionFromBrewCask(caskName)()
		return CheckResult{Installed: true, Version: version}
	}
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
			out, err := exec.Command("brew", "--version").Output()
			if err != nil {
				return Installed()
			}
			version := tool.ParseBrewVersion(string(out))
			formulaeOut, _ := exec.Command("brew", "list", "--formula", "-1").Output()  // non-critical
			caskOut, _ := exec.Command("brew", "list", "--cask", "-1").Output()          // non-critical
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
	{
		Name:        "plannotator",
		Description: "Review AI agent plans and code before committing",
		Command:     "plannotator",
		Method:      InstallManual,
		Category:    CategoryAITooling,
		InstallFn: func() error {
			cmd := exec.Command("bash", "-c", "curl -fsSL https://plannotator.ai/install.sh | bash")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		},
		VersionFn: tool.VersionFromCmd("plannotator", []string{"--version"}, tool.TrimVersion),
	},

	// ==========================================================================
	// GUI Apps + Desktop Tooling
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
		CheckFn:      checkAppWithCask("Conductor", "conductor"),
	},
	{
		Name:         "ghostty",
		Formula:      "ghostty",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"ghostty"},
		CheckFn:      checkAppWithCask("Ghostty", "ghostty"),
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
		CheckFn:      checkAppWithCask("Lens", "lens"),
	},
	{
		Name:         "zed",
		Description:  "Zed code editor",
		Formula:      "zed",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		Scripts:      []string{"zed"},
		CheckFn:      checkApp("Zed"),
	},
	{
		Name:         "android-studio",
		Description:  "Android development IDE",
		Formula:      "android-studio",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Android Studio"),
	},
	{
		Name:         "bitwarden",
		Description:  "Password manager",
		Formula:      "bitwarden",
		Method:       InstallBrewCask,
		Category:     CategorySecurity,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Bitwarden"),
	},
	{
		Name:         "brave",
		Description:  "Privacy-focused web browser",
		Formula:      "brave-browser",
		Method:       InstallBrewCask,
		Category:     CategoryBrowse,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Brave Browser"),
	},
	{
		Name:         "chatgpt",
		Description:  "OpenAI ChatGPT desktop app",
		Formula:      "chatgpt",
		Method:       InstallBrewCask,
		Category:     CategoryAIApps,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("ChatGPT"),
	},
	{
		Name:         "claude-desktop",
		Description:  "Anthropic Claude desktop app",
		Formula:      "claude",
		Method:       InstallBrewCask,
		Category:     CategoryAIApps,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Claude"),
	},
	{
		Name:         "cursor",
		Description:  "AI-powered code editor",
		Formula:      "cursor",
		Method:       InstallBrewCask,
		Category:     CategoryDevelopment,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Cursor"),
	},
	{
		Name:         "discord",
		Description:  "Voice and text chat",
		Formula:      "discord",
		Method:       InstallBrewCask,
		Category:     CategoryCommunication,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Discord"),
	},
	{
		Name:         "linear",
		Description:  "Project management tool",
		Formula:      "linear-linear",
		Method:       InstallBrewCask,
		Category:     CategoryProductivity,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Linear"),
	},
	{
		Name:         "notion",
		Description:  "Workspace for notes and docs",
		Formula:      "notion",
		Method:       InstallBrewCask,
		Category:     CategoryProductivity,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Notion"),
	},
	{
		Name:         "obsidian",
		Description:  "Knowledge base and note-taking",
		Formula:      "obsidian",
		Method:       InstallBrewCask,
		Category:     CategoryProductivity,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Obsidian"),
	},
	{
		Name:         "slack",
		Description:  "Team communication",
		Formula:      "slack",
		Method:       InstallBrewCask,
		Category:     CategoryCommunication,
		Dependencies: []string{"homebrew"},
		CheckFn:      checkApp("Slack"),
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
		CheckFn:      checkApp("WhatsApp"),
	},

	// ==========================================================================
	// Terminal
	// ==========================================================================
	{
		Name:         "yazi",
		Description:  "Terminal file manager",
		Command:      "yazi",
		Formula:      "yazi",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("yazi", []string{"--version"}, tool.ParseBrewVersion),
	},
	{
		Name:         "lazygit",
		Description:  "Terminal UI for git",
		Command:      "lazygit",
		Formula:      "lazygit",
		Method:       InstallBrewFormula,
		Category:     CategoryTerminal,
		Dependencies: []string{"homebrew"},
		VersionFn:    tool.VersionFromCmd("lazygit", []string{"--version"}, tool.TrimVersion),
	},
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
	{Name: "adguard", Description: "Ad blocker for Safari", Method: InstallMAS, Category: CategorySecurity, CheckFn: checkApp("AdGuard for Safari")},
	{Name: "broadcasts", Method: InstallMAS, Category: CategoryEntertainment, CheckFn: checkApp("Broadcasts")},
	{Name: "compressor", Description: "Apple video compression tool", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Compressor")},
	{Name: "dia", Description: "AI assistant by Apple", Method: InstallMAS, Category: CategoryBrowse, CheckFn: checkApp("Dia")},
	{Name: "final-cut-pro", Description: "Professional video editor", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Final Cut Pro")},
	{Name: "lightroom", Description: "Adobe photo editor", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Adobe Lightroom")},
	{Name: "logic-pro", Description: "Professional music production", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Logic Pro")},
	{Name: "messenger", Description: "Facebook Messenger", Method: InstallMAS, Category: CategoryCommunication, CheckFn: checkApp("Messenger")},
	{Name: "keynote", Description: "Apple presentations", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Keynote")},
	{Name: "numbers", Description: "Apple spreadsheets", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Numbers")},
	{Name: "pages", Description: "Apple word processor", Method: InstallMAS, Category: CategoryCreative, CheckFn: checkApp("Pages")},
	{Name: "passepartout", Description: "VPN client", Method: InstallMAS, Category: CategorySecurity, CheckFn: checkApp("Passepartout")},
	{Name: "pipifier", Description: "Picture-in-Picture for Safari", Method: InstallMAS, Category: CategoryUtilities, CheckFn: checkApp("PiPifier")},
	{Name: "raindrop", Description: "Bookmark manager", Method: InstallMAS, Category: CategoryEntertainment, CheckFn: checkApp("Save to Raindrop.io")},
	{Name: "snippety", Description: "Code snippet manager", Method: InstallMAS, Category: CategoryUtilities, CheckFn: checkApp("Snippety")},
	{Name: "speedtest", Description: "Internet speed test", Method: InstallMAS, Category: CategoryUtilities, CheckFn: checkApp("Speedtest")},
	{Name: "xcode", Description: "Apple development IDE", Method: InstallMAS, Category: CategoryDevelopment, CheckFn: checkApp("Xcode")},
}
