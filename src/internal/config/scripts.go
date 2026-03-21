package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jterrazz/jterrazz-cli/src/internal/domain/tool"
	out "github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

const dnsProfileIdentifier = "com.jterrazz.dns.quad9"

// ScriptCategory groups scripts by their purpose
type ScriptCategory string

const (
	ScriptCategoryTerminal ScriptCategory = "Terminal"
	ScriptCategorySecurity ScriptCategory = "Security"
	ScriptCategoryEditor   ScriptCategory = "Editor"
	ScriptCategorySystem   ScriptCategory = "System"
)

// Script represents a setup/configuration task
// Scripts can be standalone or attached to a Tool via Tool.Scripts
type Script struct {
	Name        string
	Description string
	Category    ScriptCategory

	// Check - verify if already configured (optional)
	// If nil, script is "run-once" with no checkable state
	CheckFn func() CheckResult

	// Run - execute the script
	RunFn func() error

	// ExecArgs - when set, the script runs via tea.ExecProcess (suspends TUI)
	// Use for interactive commands that need full terminal control
	ExecArgs []string

	// Dependencies
	RequiresTool string // Tool that must be installed first (e.g., "openjdk")
}

// Scripts is the single source of truth for all setup/configuration scripts.
var Scripts = []Script{
	// ==========================================================================
	// Terminal
	// ==========================================================================
	{
		Name:        "hushlogin",
		Description: "Silence terminal login message",
		Category:    ScriptCategoryTerminal,
		CheckFn: func() CheckResult {
			hushPath := os.Getenv("HOME") + "/.hushlogin"
			if _, err := os.Stat(hushPath); err == nil {
				return CheckResult{Installed: true, Detail: "~/.hushlogin"}
			}
			return CheckResult{}
		},
		RunFn: runHushlogin,
	},
	{
		Name:        "claude",
		Description: "Install Claude Code settings (MCP servers)",
		Category:    ScriptCategoryEditor,
		CheckFn: func() CheckResult {
			configPath := os.Getenv("HOME") + "/.claude/settings.json"
			if _, err := os.Stat(configPath); err == nil {
				return CheckResult{Installed: true, Detail: "~/.claude/settings.json"}
			}
			return CheckResult{}
		},
		RunFn: runClaudeConfig,
	},
	{
		Name:         "ghostty",
		Description:  "Install Ghostty terminal config",
		Category:     ScriptCategoryTerminal,
		RequiresTool: "ghostty",
		CheckFn: func() CheckResult {
			configPath := os.Getenv("HOME") + "/Library/Application Support/com.mitchellh.ghostty/config"
			if _, err := os.Stat(configPath); err == nil {
				return CheckResult{Installed: true, Detail: "~/Library/Application Support/com.mitchellh.ghostty/config"}
			}
			return CheckResult{}
		},
		RunFn: runGhosttyConfig,
	},
	{
		Name:         "tmux",
		Description:  "Install tmux config",
		Category:     ScriptCategoryTerminal,
		RequiresTool: "tmux",
		CheckFn: func() CheckResult {
			configPath := os.Getenv("HOME") + "/.tmux.conf"
			if _, err := os.Stat(configPath); err == nil {
				return CheckResult{Installed: true, Detail: "~/.tmux.conf"}
			}
			return CheckResult{}
		},
		RunFn: runTmuxConfig,
	},

	// ==========================================================================
	// Security
	// ==========================================================================
	{
		Name:         "gpg",
		Description:  "Configure GPG for commit signing",
		Category:     ScriptCategorySecurity,
		RequiresTool: "gpg",
		CheckFn: func() CheckResult {
			out, _ := exec.Command("git", "config", "--global", "commit.gpgsign").Output()
			if strings.TrimSpace(string(out)) == "true" {
				return CheckResult{Installed: true, Detail: "commit.gpgsign=true"}
			}
			return CheckResult{}
		},
		RunFn: runGPGSetup,
	},
	{
		Name:        "ssh",
		Description: "Generate SSH key with Keychain integration",
		Category:    ScriptCategorySecurity,
		CheckFn: func() CheckResult {
			sshKey := os.Getenv("HOME") + "/.ssh/id_ed25519"
			if _, err := os.Stat(sshKey); err == nil {
				return CheckResult{Installed: true, Detail: "~/.ssh/id_ed25519"}
			}
			return CheckResult{}
		},
		RunFn: runSSHSetup,
	},
	{
		Name:         "gh",
		Description:  "Authenticate GitHub CLI",
		Category:     ScriptCategorySecurity,
		RequiresTool: "gh",
		CheckFn: func() CheckResult {
			if err := exec.Command("gh", "auth", "status").Run(); err != nil {
				return CheckResult{}
			}
			return InstalledWithDetail("authenticated")
		},
		ExecArgs: []string{"gh", "auth", "login", "--hostname", "github.com", "--git-protocol", "ssh", "--web", "--skip-ssh-key"},
	},
	{
		Name:        "spotlight-exclude",
		Description: "Exclude ~/Developer from Spotlight indexing",
		Category:    ScriptCategorySecurity,
		CheckFn: func() CheckResult {
			marker := os.Getenv("HOME") + "/Developer/.metadata_never_index"
			if _, err := os.Stat(marker); err == nil {
				return InstalledWithDetail("~/Developer excluded")
			}
			return CheckResult{}
		},
		RunFn: runSpotlightExclude,
	},
	{
		Name:        "dns",
		Description: "Encrypted DNS via Quad9 (DoH)",
		Category:    ScriptCategorySecurity,
		CheckFn: func() CheckResult {
			if IsDNSProfileInstalled() {
				return InstalledWithDetail("Quad9 DoH")
			}
			return CheckResult{}
		},
		RunFn: runDNSEncrypt,
	},
	// ==========================================================================
	// Editor
	// ==========================================================================
	{
		Name:         "zed",
		Description:  "Install Zed editor config",
		Category:     ScriptCategoryEditor,
		RequiresTool: "zed",
		CheckFn: func() CheckResult {
			configPath := os.Getenv("HOME") + "/.config/zed/settings.json"
			if _, err := os.Stat(configPath); err == nil {
				return CheckResult{Installed: true, Detail: "~/.config/zed/settings.json"}
			}
			return CheckResult{}
		},
		RunFn: runZedConfig,
	},

	// ==========================================================================
	// System
	// ==========================================================================
	{
		Name:         "java",
		Description:  "Configure JAVA_HOME in shell profile",
		Category:     ScriptCategorySystem,
		RequiresTool: "openjdk",
		CheckFn: func() CheckResult {
			javaHome := "/opt/homebrew/opt/openjdk"
			if _, err := os.Stat(javaHome + "/bin/java"); err == nil {
				return CheckResult{Installed: true, Detail: "JAVA_HOME=" + javaHome}
			}
			return CheckResult{}
		},
		RunFn: runJavaHome,
	},
	{
		Name:        "dock-reset",
		Description: "Reset dock to system defaults",
		Category:    ScriptCategorySystem,
		RunFn:       runDockReset,
	},
	{
		Name:        "dock-spacer",
		Description: "Add a small spacer tile to the dock",
		Category:    ScriptCategorySystem,
		RunFn:       runDockSpacer,
	},
}

// =============================================================================
// Script Runners
// =============================================================================

func runHushlogin() error {
	fmt.Println(out.Cyan("Setting up hushlogin..."))

	hushPath := os.Getenv("HOME") + "/.hushlogin"
	if _, err := os.Stat(hushPath); err == nil {
		fmt.Printf("%s .hushlogin already exists\n", out.Green("Done"))
		return nil
	}

	f, err := os.Create(hushPath)
	if err != nil {
		return fmt.Errorf("failed to create .hushlogin: %w", err)
	}
	f.Close()

	fmt.Println(out.Green("Done - terminal login message silenced"))
	return nil
}

// copyRepoConfig copies a config file from the repo to a destination path.
func copyRepoConfig(repoRelPath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	repoConfig, err := GetRepoConfigPath(repoRelPath)
	if err != nil {
		return fmt.Errorf("failed to find repo config: %w", err)
	}

	content, err := os.ReadFile(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", repoConfig, err)
	}

	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", destPath, err)
	}

	return nil
}

func runClaudeConfig() error {
	fmt.Println(out.Cyan("Setting up Claude Code config..."))
	destPath := os.Getenv("HOME") + "/.claude/settings.json"
	if err := copyRepoConfig("dotfiles/applications/claude/settings.json", destPath); err != nil {
		return err
	}
	fmt.Println(out.Green("Done - Claude Code config installed"))
	return nil
}

func runGhosttyConfig() error {
	fmt.Println(out.Cyan("Setting up Ghostty config..."))
	destPath := os.Getenv("HOME") + "/Library/Application Support/com.mitchellh.ghostty/config"
	if err := copyRepoConfig("dotfiles/applications/ghostty/config", destPath); err != nil {
		return err
	}
	fmt.Println(out.Green("Done - Ghostty config installed"))
	return nil
}

func runTmuxConfig() error {
	fmt.Println(out.Cyan("Setting up tmux config..."))
	configPath := os.Getenv("HOME") + "/.tmux.conf"
	if err := copyRepoConfig("dotfiles/applications/tmux/tmux.conf", configPath); err != nil {
		return err
	}
	if err := exec.Command("tmux", "source-file", configPath).Run(); err == nil {
		fmt.Println(out.Green("Done - tmux config installed and reloaded"))
		return nil
	}
	fmt.Println(out.Green("Done - tmux config installed"))
	return nil
}

func runGPGSetup() error {
	fmt.Println(out.Cyan("Setting up GPG for commit signing..."))

	email := UserEmail
	name := UserName

	if !CommandExists("gpg") {
		return fmt.Errorf("GPG not installed. Run: brew install gnupg")
	}

	checkCmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "long", email)
	if output, err := checkCmd.Output(); err == nil && len(output) > 0 {
		fmt.Println(out.Green("GPG key already exists for " + email))
		configureGitGPG(email)
		return nil
	}

	fmt.Println("Generating GPG key...")
	fmt.Println(out.Dimmed("Using ed25519 algorithm"))

	batchConfig := fmt.Sprintf(`%%no-protection
Key-Type: eddsa
Key-Curve: ed25519
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%commit
`, name, email)

	genCmd := exec.Command("gpg", "--batch", "--generate-key")
	genCmd.Stdin = strings.NewReader(batchConfig)
	genCmd.Stdout = os.Stdout
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate GPG key: %w", err)
	}
	fmt.Println(out.Green("GPG key generated"))

	configureGitGPG(email)
	return nil
}

func configureGitGPG(email string) {
	listCmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "long", email)
	output, err := listCmd.Output()
	if err != nil {
		out.Error("Failed to list GPG keys")
		return
	}

	lines := strings.Split(string(output), "\n")
	var keyID string
	for _, line := range lines {
		if strings.Contains(line, "ed25519/") || strings.Contains(line, "rsa") {
			parts := strings.Split(line, "/")
			if len(parts) >= 2 {
				keyID = strings.Fields(parts[1])[0]
				break
			}
		}
	}

	if keyID == "" {
		out.Error("Could not find GPG key ID")
		return
	}

	fmt.Println("Configuring Git to use GPG key...")

	exec.Command("git", "config", "--global", "user.signingkey", keyID).Run()
	exec.Command("git", "config", "--global", "commit.gpgsign", "true").Run()
	exec.Command("git", "config", "--global", "gpg.program", "gpg").Run()

	fmt.Println(out.Green("Git configured for commit signing"))

	fmt.Println()
	fmt.Println("Your GPG public key (add to GitHub):")
	fmt.Println("----------------------------------------")
	exportCmd := exec.Command("gpg", "--armor", "--export", email)
	exportCmd.Stdout = os.Stdout
	exportCmd.Run()
	fmt.Println("----------------------------------------")
	fmt.Println("Add at: https://github.com/settings/gpg/new")

	fmt.Println()
	fmt.Println(out.Green("GPG setup completed"))
	fmt.Println(out.Dimmed("All future commits will be signed automatically"))
}

func runSSHSetup() error {
	fmt.Println(out.Cyan("Setting up SSH..."))

	sshDir := os.Getenv("HOME") + "/.ssh"
	sshKey := sshDir + "/id_ed25519"
	email := UserEmail

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	if _, err := os.Stat(sshKey); err == nil {
		fmt.Printf("%s SSH key already exists at %s\n", out.Green("Done"), sshKey)
	} else {
		fmt.Println("Generating SSH key with macOS Keychain integration...")
		fmt.Println(out.Dimmed("You'll be prompted to create a passphrase"))
		fmt.Println()

		genCmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", sshKey)
		genCmd.Stdin = os.Stdin
		genCmd.Stdout = os.Stdout
		genCmd.Stderr = os.Stderr
		if err := genCmd.Run(); err != nil {
			return fmt.Errorf("failed to generate SSH key: %w", err)
		}
		fmt.Println(out.Green("SSH key generated"))
	}

	fmt.Println("Configuring SSH...")
	sshConfig := sshDir + "/config"

	existingConfig, _ := os.ReadFile(sshConfig)
	if !strings.Contains(string(existingConfig), "AddKeysToAgent yes") {
		configContent := `
Host *
  AddKeysToAgent yes
  UseKeychain yes
  IdentityFile ~/.ssh/id_ed25519
`
		f, err := os.OpenFile(sshConfig, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			f.WriteString(configContent)
			f.Close()
			fmt.Println(out.Green("SSH config updated"))
		}
	} else {
		fmt.Println(out.Green("SSH config already configured"))
	}

	fmt.Println("Adding key to SSH agent with Keychain...")
	fmt.Println(out.Dimmed("Passphrase will be stored in macOS Keychain"))
	fmt.Println()

	addCmd := exec.Command("ssh-add", "--apple-use-keychain", sshKey)
	addCmd.Stdin = os.Stdin
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add key to SSH agent: %w", err)
	}

	fmt.Println()
	fmt.Println("Your public key (add to GitHub):")
	fmt.Println("----------------------------------------")
	pubKey, _ := os.ReadFile(sshKey + ".pub")
	fmt.Println(string(pubKey))
	fmt.Println("----------------------------------------")
	fmt.Println("Add at: https://github.com/settings/ssh/new")

	fmt.Println(out.Green("SSH setup completed"))
	return nil
}

func runSpotlightExclude() error {
	fmt.Println(out.Cyan("Excluding ~/Developer from Spotlight indexing..."))

	devDir := os.Getenv("HOME") + "/Developer"
	marker := devDir + "/.metadata_never_index"

	if _, err := os.Stat(devDir); err != nil {
		return fmt.Errorf("~/Developer directory does not exist")
	}

	if _, err := os.Stat(marker); err == nil {
		fmt.Println(out.Green("Done - already excluded"))
		return nil
	}

	f, err := os.Create(marker)
	if err != nil {
		return fmt.Errorf("failed to create .metadata_never_index: %w", err)
	}
	f.Close()

	fmt.Println(out.Green("Done - ~/Developer excluded from Spotlight"))
	fmt.Println(out.Dimmed("Spotlight will stop indexing this directory"))
	return nil
}

func runZedConfig() error {
	fmt.Println(out.Cyan("Setting up Zed config..."))
	destPath := os.Getenv("HOME") + "/.config/zed/settings.json"
	if err := copyRepoConfig("dotfiles/applications/zed/settings.json", destPath); err != nil {
		return err
	}
	fmt.Println(out.Green("Done - Zed config installed"))
	return nil
}

func runJavaHome() error {
	fmt.Println(out.Cyan("Setting up JAVA_HOME..."))

	javaHome := "/opt/homebrew/opt/openjdk"
	if _, err := os.Stat(javaHome + "/bin/java"); err != nil {
		return fmt.Errorf("OpenJDK not installed. Run: j install openjdk")
	}

	zshrcPath := os.Getenv("HOME") + "/.zshrc"
	existing, _ := os.ReadFile(zshrcPath)
	if strings.Contains(string(existing), "JAVA_HOME") {
		fmt.Println(out.Green("Done - JAVA_HOME already configured in ~/.zshrc"))
		return nil
	}

	f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open ~/.zshrc: %w", err)
	}
	defer f.Close()

	javaConfig := fmt.Sprintf("\n# Java (managed by j)\nexport JAVA_HOME=\"%s\"\nexport PATH=\"$JAVA_HOME/bin:$PATH\"\n", javaHome)
	if _, err := f.WriteString(javaConfig); err != nil {
		return fmt.Errorf("failed to write to ~/.zshrc: %w", err)
	}

	fmt.Println(out.Green("Done - JAVA_HOME configured in ~/.zshrc"))
	fmt.Println(out.Dimmed("Run 'source ~/.zshrc' to apply"))
	return nil
}

func runDockReset() error {
	fmt.Println(out.Cyan("Resetting macOS Dock..."))
	ExecCommand("defaults", "delete", "com.apple.dock")
	ExecCommand("killall", "Dock")
	fmt.Println(out.Green("Done - Dock reset to defaults"))
	return nil
}

func runDockSpacer() error {
	fmt.Println(out.Cyan("Adding spacer to Dock..."))
	ExecCommand("defaults", "write", "com.apple.dock", "persistent-apps", "-array-add", `{"tile-type"="small-spacer-tile";}`)
	ExecCommand("killall", "Dock")
	fmt.Println(out.Green("Done - Dock spacer added"))
	return nil
}

func IsDNSProfileInstalled() bool {
	out, _ := exec.Command("profiles", "-C", "-v").Output()
	return strings.Contains(string(out), dnsProfileIdentifier)
}

func dnsProfilePath() string {
	return filepath.Join(os.Getenv("HOME"), ".jterrazz", "dns", "quad9-dns.mobileconfig")
}

func runDNSEncrypt() error {
	if IsDNSProfileInstalled() {
		exec.Command("open", "x-apple.systempreferences:com.apple.Profiles-Settings.extension").Run()
		return nil
	}

	profilePath := dnsProfilePath()
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(profilePath, []byte(generateDNSProfile()), 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	exec.Command("open", profilePath).Run()

	// Give macOS time to read the file before the TUI resumes
	time.Sleep(2 * time.Second)
	return nil
}

func generateDNSProfile() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>DNSSettings</key>
			<dict>
				<key>DNSProtocol</key>
				<string>HTTPS</string>
				<key>ServerAddresses</key>
				<array>
					<string>2620:fe::fe</string>
					<string>2620:fe::9</string>
					<string>9.9.9.9</string>
					<string>149.112.112.112</string>
				</array>
				<key>ServerURL</key>
				<string>https://dns.quad9.net/dns-query</string>
			</dict>
			<key>PayloadDescription</key>
			<string>Configures device to use Quad9 Encrypted DNS over HTTPS</string>
			<key>PayloadDisplayName</key>
			<string>Quad9 DNS over HTTPS</string>
			<key>PayloadIdentifier</key>
			<string>` + dnsProfileIdentifier + `.doh</string>
			<key>PayloadType</key>
			<string>com.apple.dnsSettings.managed</string>
			<key>PayloadUUID</key>
			<string>B9C4F3A2-5E6D-7F8A-9B0C-1D2E3F4A5B6C</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ProhibitDisablement</key>
			<false/>
		</dict>
	</array>
	<key>PayloadDescription</key>
	<string>Configures encrypted DNS over HTTPS using Quad9 (9.9.9.9)</string>
	<key>PayloadDisplayName</key>
	<string>Quad9 Encrypted DNS</string>
	<key>PayloadIdentifier</key>
	<string>` + dnsProfileIdentifier + `</string>
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>A8B3E2F1-4D5C-6E7F-8A9B-0C1D2E3F4A5B</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`
}

// =============================================================================
// Helper Functions
// =============================================================================

// GetRepoConfigPath returns the full path for a file in the repo
func GetRepoConfigPath(relativePath string) (string, error) {
	root := os.Getenv("HOME") + "/Developer/jterrazz-cli"
	fullPath := root + "/" + relativePath
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}
	return "", fmt.Errorf("config file not found: %s (expected at %s)", relativePath, fullPath)
}

// ExecCommand runs a command with stdout/stderr/stdin attached
func ExecCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// CommandExists checks if a command is available in PATH
var CommandExists = tool.CommandExists

// =============================================================================
// Script Functions
// =============================================================================

// GetAllScripts returns all scripts
func GetAllScripts() []Script {
	return Scripts
}

// GetScriptByName returns a script by name
func GetScriptByName(name string) *Script {
	for i := range Scripts {
		if Scripts[i].Name == name {
			return &Scripts[i]
		}
	}
	return nil
}

// GetScriptsByCategory returns scripts filtered by category
func GetScriptsByCategory(category ScriptCategory) []Script {
	var result []Script
	for _, script := range Scripts {
		if script.Category == category {
			result = append(result, script)
		}
	}
	return result
}

// GetScriptsForTool returns scripts that belong to a tool
func GetScriptsForTool(toolName string) []Script {
	tool := GetToolByName(toolName)
	if tool == nil || len(tool.Scripts) == 0 {
		return nil
	}

	var result []Script
	for _, scriptName := range tool.Scripts {
		if script := GetScriptByName(scriptName); script != nil {
			result = append(result, *script)
		}
	}
	return result
}

// GetStandaloneScripts returns scripts not attached to any tool
func GetStandaloneScripts() []Script {
	// Build set of tool-attached scripts
	attached := make(map[string]bool)
	for _, tool := range Tools {
		for _, scriptName := range tool.Scripts {
			attached[scriptName] = true
		}
	}

	var result []Script
	for _, script := range Scripts {
		if !attached[script.Name] {
			result = append(result, script)
		}
	}
	return result
}

// GetConfigurableScripts returns scripts that have a CheckFn (can be checked)
func GetConfigurableScripts() []Script {
	var result []Script
	for _, script := range Scripts {
		if script.CheckFn != nil {
			result = append(result, script)
		}
	}
	return result
}

// GetUnconfiguredScripts returns scripts that haven't been run yet
func GetUnconfiguredScripts() []Script {
	var result []Script
	for _, script := range Scripts {
		if script.CheckFn != nil {
			check := script.CheckFn()
			if !check.Installed {
				result = append(result, script)
			}
		}
	}
	return result
}

// CheckScript checks if a script has been configured
func CheckScript(script Script) CheckResult {
	if script.CheckFn != nil {
		return script.CheckFn()
	}
	return CheckResult{} // No check = unknown state
}

// RunScript runs a script (placeholder - actual implementation in commands)
func RunScript(script Script) error {
	if script.RunFn != nil {
		return script.RunFn()
	}
	// Scripts without RunFn are invoked via `j setup <RunCmd>`
	return nil
}
