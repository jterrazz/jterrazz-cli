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
	ScriptCategoryServer   ScriptCategory = "Server"
)

// Script represents a setup/configuration task
// Scripts can be standalone or attached to a Tool via Tool.Scripts
type Script struct {
	Name        string
	Description string
	Category    ScriptCategory

	// Help is the long-form description shown in the j config detail panel.
	// Wraps freely; aim for a short paragraph describing what the install does
	// and why someone would want it.
	Help string

	// Role - if set, the script only applies to machines registered with that
	// role. Items with no Role apply to every machine. Used to gate server-only
	// configuration from the TUI on a client box.
	Role Role

	// Check - verify if already configured (optional)
	// If nil, script is "run-once" with no checkable state
	CheckFn func() CheckResult

	// InstallFn - perform the installation/configuration action. Receives the
	// values collected via the Inputs modal (empty map if Inputs is nil).
	// For scripts that don't need inputs, wrap a plain func() error with
	// config.NoInputs(...) at assignment time.
	InstallFn func(InputValues) error

	// UninstallFn - inverse of InstallFn. When set, the item is toggleable in
	// the TUI: pressing 'u' on an installed item runs this instead of InstallFn.
	UninstallFn func() error

	// Inputs - if non-empty, the TUI collects these values via a modal form
	// before calling InstallFn. The script reads them via a side channel
	// (typically env vars or a closure-captured map).
	Inputs []ScriptInput

	// ExecArgs - when set, the script runs via tea.ExecProcess (suspends TUI)
	// Use for interactive commands that need full terminal control
	ExecArgs []string

	// Interactive - when true, InstallFn is wrapped in tea.Exec so the TUI is
	// released while it runs. Required for any InstallFn that prompts the user
	// (passphrase entry, GPG key generation, etc.).
	Interactive bool

	// Dependencies
	RequiresTool string // Tool that must be installed first (e.g., "openjdk")
}

// InputValues maps Input.Name to the user-provided value for that input.
// Always non-nil when passed to InstallFn (use Get to read safely).
type InputValues map[string]string

// Get returns the value for `name` or "" if missing.
func (v InputValues) Get(name string) string {
	if v == nil {
		return ""
	}
	return v[name]
}

// NoInputs wraps a plain `func() error` so it satisfies the Script.InstallFn
// signature. Convenience for the common case where a script needs no inputs.
func NoInputs(fn func() error) func(InputValues) error {
	return func(_ InputValues) error { return fn() }
}

// InputKind enumerates the supported modal input field types.
type InputKind int

const (
	InputText     InputKind = iota // single-line text input
	InputPassword                  // masked single-line input
	InputSelect                    // pick one from Options
	InputConfirm                   // yes/no
)

// ScriptInput describes a single value collected from the user before an
// InstallFn runs. Rendered in a modal form by the j config TUI.
type ScriptInput struct {
	Name     string // key the InstallFn uses to look up the value
	Label    string // primary prompt
	Help     string // sub-text explaining the field
	Kind     InputKind
	Options  []string // values for InputSelect
	Default  string   // pre-filled value (e.g. from an env var)
	Validate func(string) error
}

// MatchesRole reports whether the script applies to the given machine role.
// Scripts with no Role match every role.
func (s Script) MatchesRole(role Role) bool {
	return s.Role == "" || s.Role == role
}

// checkFileExists returns a CheckFn that reports installed if path exists.
func checkFileExists(path, displayPath string) func() CheckResult {
	return func() CheckResult {
		if _, err := os.Stat(path); err == nil {
			return CheckResult{Installed: true, Detail: displayPath}
		}
		return CheckResult{}
	}
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
		Help:        "Creates ~/.hushlogin so macOS suppresses the 'Last login:' banner each time you open a new terminal.",
		CheckFn:     checkFileExists(os.Getenv("HOME")+"/.hushlogin", "~/.hushlogin"),
		InstallFn:   NoInputs(runHushlogin),
	},
	{
		Name:         "ghostty",
		Description:  "Install Ghostty terminal config",
		Category:     ScriptCategoryTerminal,
		RequiresTool: "ghostty",
		Help:         "Drops the repo's Ghostty config and two Catppuccin themes (espresso, latte) into ~/.config/ghostty.",
		CheckFn: checkFileExists(
			os.Getenv("HOME")+"/.config/ghostty/config",
			"~/.config/ghostty/config"),
		InstallFn: NoInputs(runGhosttyConfig),
	},
	{
		Name:         "tmux",
		Description:  "Install tmux config",
		Category:     ScriptCategoryTerminal,
		RequiresTool: "tmux",
		Help:         "Installs ~/.tmux.conf with sensible bindings; reloads any running tmux session.",
		CheckFn:      checkFileExists(os.Getenv("HOME")+"/.tmux.conf", "~/.tmux.conf"),
		InstallFn:    NoInputs(runTmuxConfig),
	},

	// ==========================================================================
	// Security
	// ==========================================================================
	{
		Name:         "gpg",
		Description:  "Configure GPG for commit signing",
		Category:     ScriptCategorySecurity,
		RequiresTool: "gpg",
		Interactive:  true,
		Help:         "Generates an ed25519 GPG key for the configured git email and wires it into git so all future commits are signed.",
		CheckFn: func() CheckResult {
			out, err := exec.Command("git", "config", "--global", "commit.gpgsign").Output()
			if err != nil {
				return CheckResult{}
			}
			if strings.TrimSpace(string(out)) == "true" {
				return CheckResult{Installed: true, Detail: "commit.gpgsign=true"}
			}
			return CheckResult{}
		},
		InstallFn: NoInputs(runGPGSetup),
	},
	{
		Name:        "ssh",
		Description: "Generate SSH key with Keychain integration",
		Category:    ScriptCategorySecurity,
		Interactive: true,
		Help:        "Generates an ed25519 SSH key, stores its passphrase in macOS Keychain, and prints the public key for you to paste into GitHub.",
		CheckFn:     checkFileExists(os.Getenv("HOME")+"/.ssh/id_ed25519", "~/.ssh/id_ed25519"),
		InstallFn:   NoInputs(runSSHSetup),
	},
	{
		Name:         "gh",
		Description:  "Authenticate GitHub CLI",
		Category:     ScriptCategorySecurity,
		RequiresTool: "gh",
		Help:         "Runs `gh auth login` against github.com via the browser, configures git to push over SSH.",
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
		Description: "Exclude ~/Developer from Spotlight (guided — opens Settings)",
		Category:    ScriptCategorySecurity,
		Help:        "macOS Tahoe stores the Spotlight Privacy list in a root-only plist (/System/Volumes/Data/.Spotlight-V100/VolumeConfiguration.plist), so we can't read it from the TUI without sudo on every render. This opens System Settings → Spotlight → Search Privacy and you drop ~/Developer onto the list manually. Then we touch ~/.jterrazz/spotlight-exclude.done so future checks see ✓ — delete that file if you ever remove the exclusion.",
		CheckFn: func() CheckResult {
			if isSpotlightExcluded(os.Getenv("HOME") + "/Developer") {
				return InstalledWithDetail("~/Developer in Spotlight Privacy list")
			}
			return CheckResult{}
		},
		InstallFn: NoInputs(runSpotlightExclude),
	},
	{
		Name:        "dns",
		Description: "Encrypted DNS via Quad9 (DoH)",
		Category:    ScriptCategorySecurity,
		Help:        "Generates a configuration profile for DNS-over-HTTPS via Quad9 (9.9.9.9) and opens it in System Settings for installation.",
		CheckFn: func() CheckResult {
			if IsDNSProfileInstalled() {
				return InstalledWithDetail("Quad9 DoH")
			}
			return CheckResult{}
		},
		InstallFn: NoInputs(runDNSEncrypt),
	},
	// ==========================================================================
	// Editor
	// ==========================================================================
	{
		Name:         "zed",
		Description:  "Install Zed editor config",
		Category:     ScriptCategoryEditor,
		RequiresTool: "zed",
		Help:         "Installs ~/.config/zed/settings.json from the repo (themes, keymaps, font).",
		CheckFn:      checkFileExists(os.Getenv("HOME")+"/.config/zed/settings.json", "~/.config/zed/settings.json"),
		InstallFn:    NoInputs(runZedConfig),
	},

	// ==========================================================================
	// System
	// ==========================================================================
	{
		Name:         "java",
		Description:  "Configure JAVA_HOME in shell profile",
		Category:     ScriptCategorySystem,
		RequiresTool: "openjdk",
		Help:         "Appends a JAVA_HOME export to ~/.zshrc pointing at /opt/homebrew/opt/openjdk so java/javac are on PATH.",
		CheckFn: func() CheckResult {
			javaHome := "/opt/homebrew/opt/openjdk"
			if _, err := os.Stat(javaHome + "/bin/java"); err == nil {
				return CheckResult{Installed: true, Detail: "JAVA_HOME=" + javaHome}
			}
			return CheckResult{}
		},
		InstallFn: NoInputs(runJavaHome),
	},
	{
		Name:         "nvm",
		Description:  "Initialize per-user nvm state (~/.nvm)",
		Category:     ScriptCategorySystem,
		RequiresTool: "nvm",
		Help:         "Creates the ~/.nvm directory so the shell loader can manage Node versions per user.",
		CheckFn: func() CheckResult {
			if _, err := os.Stat(os.Getenv("HOME") + "/.nvm"); err == nil {
				return InstalledWithDetail("~/.nvm exists")
			}
			return CheckResult{}
		},
		InstallFn: NoInputs(runNvmSetup),
	},
	{
		Name:        "dock-reset",
		Description: "Reset dock to system defaults",
		Category:    ScriptCategorySystem,
		Help:        "`defaults delete com.apple.dock` + killall Dock — restores the macOS default dock layout.",
		InstallFn:   NoInputs(runDockReset),
	},
	{
		Name:        "dock-spacer",
		Description: "Add a small spacer tile to the dock",
		Category:    ScriptCategorySystem,
		Help:        "Adds a small spacer tile (transparent gap) to the Dock for visual grouping.",
		InstallFn:   NoInputs(runDockSpacer),
	},

	// ==========================================================================
	// Server — only visible on machines registered with role=server
	// ==========================================================================
	// CheckFn / InstallFn / UninstallFn live in src/internal/commands/machine_config_*.go
	// and are wired up via config.RegisterServerActions at init time so this file
	// stays free of macOS-specific logic.
}

// ServerActions is the set of install/uninstall/check functions provided by the
// commands package for the four server-only scripts. The commands package wires
// these in via RegisterServerActions so this file (and the config package as a
// whole) stays free of macOS-specific imports.
type ServerActions struct {
	AutologinInstall        func(InputValues) error
	AutologinUninstall      func() error
	AutologinCheck          func() CheckResult
	PowerInstall            func() error
	PowerUninstall          func() error
	PowerCheck              func() CheckResult
	LockAfterLoginInstall   func() error
	LockAfterLoginUninstall func() error
	LockAfterLoginCheck     func() CheckResult
	SshdInstall             func() error
	SshdUninstall           func() error
	SshdCheck               func() CheckResult
}

// RegisterServerActions appends the four server Scripts using the provided
// action set. Called from the commands package init().
func RegisterServerActions(a ServerActions) {
	Scripts = append(Scripts,
		Script{
			Name:        "autologin",
			Description: "GUI auto-login for jterrazz.agent (FileVault-aware)",
			Category:    ScriptCategoryServer,
			Role:        RoleServer,
			Interactive: true,
			Help:        "Bypasses loginwindow at boot/restart so an agent can drive the Aqua session without anyone at the keyboard. Writes /etc/kcpassword via the public XOR cipher (the password is never logged).",
			Inputs: []ScriptInput{
				{
					Name:    "password",
					Label:   "Agent password for jterrazz.agent",
					Help:    "Encoded into /etc/kcpassword via the public XOR cipher. Never logged or stored elsewhere.",
					Kind:    InputPassword,
					Default: os.Getenv("AGENT_PASSWORD"),
				},
			},
			CheckFn:     a.AutologinCheck,
			InstallFn:   a.AutologinInstall,
			UninstallFn: a.AutologinUninstall,
		},
		Script{
			Name:        "power",
			Description: "Always-on power policy (no sleep, autorestart, wake)",
			Category:    ScriptCategoryServer,
			Role:        RoleServer,
			Interactive: true,
			Help:        "Applies a server pmset profile: never sleep, restart on power return, no hibernate, wake on LAN. Uninstall resets pmset to macOS defaults.",
			CheckFn:     a.PowerCheck,
			InstallFn:   NoInputs(a.PowerInstall),
			UninstallFn: a.PowerUninstall,
		},
		Script{
			Name:        "lock-after-login",
			Description: "LaunchAgent that locks the screen ~20s after auto-login",
			Category:    ScriptCategoryServer,
			Role:        RoleServer,
			Interactive: true,
			Help:        "Per-user LaunchAgent that locks the screen ~20s after auto-login. Keeps the GUI session alive (so agent runtimes work) while the screen stays physically protected.",
			CheckFn:     a.LockAfterLoginCheck,
			InstallFn:   NoInputs(a.LockAfterLoginInstall),
			UninstallFn: a.LockAfterLoginUninstall,
		},
		Script{
			Name:        "sshd",
			Description: "Remote Login (sshd) + FileVault pre-boot SSH unlock group",
			Category:    ScriptCategoryServer,
			Role:        RoleServer,
			Interactive: true,
			Help:        "Enables Remote Login (sshd) and adds jterrazz.agent to the access_ssh group. The FileVault remote-unlock toggle still has to be flipped manually in System Settings → Privacy & Security.",
			CheckFn:     a.SshdCheck,
			InstallFn:   NoInputs(a.SshdInstall),
			UninstallFn: a.SshdUninstall,
		},
	)
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

// makeConfigInstaller creates a InstallFn that copies a repo config file to a destination.
func makeConfigInstaller(label, repoRelPath, destPath string) func() error {
	return func() error {
		fmt.Println(out.Cyan("Setting up " + label + " config..."))
		if err := copyRepoConfig(repoRelPath, destPath); err != nil {
			return err
		}
		fmt.Println(out.Green("Done - " + label + " config installed"))
		return nil
	}
}

func runGhosttyConfig() error {
	ghosttyDir := os.Getenv("HOME") + "/.config/ghostty"
	install := makeConfigInstaller("Ghostty",
		"dotfiles/applications/ghostty/config",
		ghosttyDir+"/config")
	if err := install(); err != nil {
		return err
	}

	themes := []string{"catppuccin-espresso-blur", "catppuccin-latte-blur"}
	for _, name := range themes {
		if err := copyRepoConfig("dotfiles/applications/ghostty/themes/"+name, ghosttyDir+"/themes/"+name); err != nil {
			return fmt.Errorf("failed to install Ghostty theme %s: %w", name, err)
		}
	}
	fmt.Println(out.Green("Done - Ghostty themes installed"))
	return nil
}

func runTmuxConfig() error {
	configPath := os.Getenv("HOME") + "/.tmux.conf"
	install := makeConfigInstaller("tmux", "dotfiles/applications/tmux/tmux.conf", configPath)
	if err := install(); err != nil {
		return err
	}
	// Try to reload tmux if running
	if err := exec.Command("tmux", "source-file", configPath).Run(); err == nil {
		fmt.Println(out.Green("  tmux config reloaded"))
	}
	return nil
}

func runGPGSetup() error {
	fmt.Println(out.Cyan("Setting up GPG for commit signing..."))

	email := UserEmail()
	name := UserName()

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

	if err := exec.Command("git", "config", "--global", "user.signingkey", keyID).Run(); err != nil {
		out.Error("Failed to set git signing key: " + err.Error())
		return
	}
	if err := exec.Command("git", "config", "--global", "commit.gpgsign", "true").Run(); err != nil {
		out.Error("Failed to enable git commit signing: " + err.Error())
		return
	}
	if err := exec.Command("git", "config", "--global", "gpg.program", "gpg").Run(); err != nil {
		out.Error("Failed to set git gpg program: " + err.Error())
		return
	}

	fmt.Println(out.Green("Git configured for commit signing"))

	fmt.Println()
	fmt.Println("Your GPG public key (add to GitHub):")
	fmt.Println("----------------------------------------")
	exportCmd := exec.Command("gpg", "--armor", "--export", email)
	exportCmd.Stdout = os.Stdout
	if err := exportCmd.Run(); err != nil {
		out.Error("Failed to export GPG key: " + err.Error())
	}
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
	email := UserEmail()

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

	existingConfig, _ := os.ReadFile(sshConfig) // ok if file doesn't exist yet
	if !strings.Contains(string(existingConfig), "AddKeysToAgent yes") {
		configContent := `
Host *
  AddKeysToAgent yes
  UseKeychain yes
  IdentityFile ~/.ssh/id_ed25519
`
		f, err := os.OpenFile(sshConfig, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("failed to open SSH config: %w", err)
		}
		if _, err := f.WriteString(configContent); err != nil {
			f.Close()
			return fmt.Errorf("failed to write SSH config: %w", err)
		}
		f.Close()
		fmt.Println(out.Green("SSH config updated"))
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
	pubKey, err := os.ReadFile(sshKey + ".pub")
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}
	fmt.Println(string(pubKey))
	fmt.Println("----------------------------------------")
	fmt.Println("Add at: https://github.com/settings/ssh/new")

	fmt.Println(out.Green("SSH setup completed"))
	return nil
}

// spotlightExcludeMarkerPath is the sentinel file written by the install
// script.
//
// On Tahoe (26+), the Spotlight Privacy list lives in
// /System/Volumes/Data/.Spotlight-V100/VolumeConfiguration.plist — root-only,
// so a TUI CheckFn can't read it without prompting for sudo on every render.
// And the legacy `defaults read com.apple.Spotlight Exclusions` returns
// "does not exist" (the key was moved out of that domain).
//
// So we fall back to "did the user run the install at least once?":
// the script touches this file after opening System Settings, on the
// assumption that the user followed through with the manual step.
// If they ever remove ~/Developer from Privacy, they can rm this marker.
func spotlightExcludeMarkerPath() string {
	return filepath.Join(os.Getenv("HOME"), ".jterrazz", "spotlight-exclude.done")
}

// isSpotlightExcluded reports whether the user has marked Spotlight Privacy
// as configured for ~/Developer. Sentinel-based — see spotlightExcludeMarkerPath.
func isSpotlightExcluded(_ string) bool {
	_, err := os.Stat(spotlightExcludeMarkerPath())
	return err == nil
}

func runSpotlightExclude() error {
	devDir := os.Getenv("HOME") + "/Developer"
	if _, err := os.Stat(devDir); err != nil {
		return fmt.Errorf("~/Developer directory does not exist")
	}

	if isSpotlightExcluded(devDir) {
		fmt.Println(out.Green("Done - ~/Developer already marked as excluded"))
		return nil
	}

	// Clean up the stale .metadata_never_index marker — it never worked
	// inside a sub-folder, only at volume root.
	staleMarker := devDir + "/.metadata_never_index"
	if _, err := os.Stat(staleMarker); err == nil {
		_ = os.Remove(staleMarker)
	}

	fmt.Println(out.Cyan("Add ~/Developer to Spotlight Privacy"))
	fmt.Println(out.Dimmed("macOS has no public API for this — it must be done in System Settings."))
	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("  1. System Settings opens at Spotlight → Search Privacy")
	fmt.Println("  2. Click + and pick ~/Developer (or drag the folder onto the list)")
	fmt.Println("  3. Re-run `j status` to confirm")
	fmt.Println()

	if err := exec.Command("open", "x-apple.systempreferences:com.apple.Spotlight-Settings.extension").Run(); err != nil {
		fmt.Println(out.Dimmed("Could not open Settings automatically — open System Settings → Spotlight → Search Privacy manually."))
	}

	// Touch the sentinel so subsequent CheckFn calls report ✓. We trust the
	// user followed the manual step — if they didn't, they can rm
	// ~/.jterrazz/spotlight-exclude.done and re-run.
	markerPath := spotlightExcludeMarkerPath()
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o700); err == nil {
		_ = os.WriteFile(markerPath, []byte("manual\n"), 0o644)
	}

	// Brief pause so System Settings is on screen before the TUI redraws.
	time.Sleep(1 * time.Second)
	return nil
}

var runZedConfig = makeConfigInstaller("Zed",
	"dotfiles/applications/zed/settings.json",
	os.Getenv("HOME")+"/.config/zed/settings.json")

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

func runNvmSetup() error {
	fmt.Println(out.Cyan("Setting up nvm..."))

	nvmSh := "/opt/homebrew/opt/nvm/nvm.sh"
	if _, err := os.Stat(nvmSh); err != nil {
		return fmt.Errorf("nvm not installed via brew (expected %s). Run: j install nvm", nvmSh)
	}

	nvmDir := os.Getenv("HOME") + "/.nvm"
	if err := os.MkdirAll(nvmDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", nvmDir, err)
	}

	fmt.Println(out.Green("Done - " + nvmDir + " ready"))
	fmt.Println(out.Dimmed("Shell loader is in dotfiles/applications/zsh/zshrc.sh — open a new shell or `source ~/.zshrc`"))
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
	out, err := exec.Command("profiles", "-C", "-v").Output()
	if err != nil {
		return false
	}
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
	for _, root := range repoRootCandidates() {
		fullPath := root + "/" + relativePath
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	candidates := repoRootCandidates()
	return "", fmt.Errorf("config file not found: %s (searched: %s)", relativePath, strings.Join(candidates, ", "))
}

// repoRootCandidates returns candidate filesystem locations of the source repo,
// in priority order. Set J_REPO_PATH to override.
func repoRootCandidates() []string {
	home := os.Getenv("HOME")
	var roots []string
	if env := os.Getenv("J_REPO_PATH"); env != "" {
		roots = append(roots, env)
	}
	roots = append(roots,
		home+"/Developer/jterrazz-cli",
		home+"/Developer/jterrazz/jterrazz-cli",
	)
	return roots
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

