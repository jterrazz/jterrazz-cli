package config

import (
	"os"
	"os/exec"
	"strings"
)

// IdentityCheck represents a developer identity verification
type IdentityCheck struct {
	Name        string
	Description string
	CheckFn     func() CheckResult
	GoodWhen    bool // true = check passes when Installed=true
}

// IdentityChecks is the list of developer identity checks
var IdentityChecks = []IdentityCheck{
	{
		Name:        "git-email",
		Description: "Git commit email",
		CheckFn: func() CheckResult {
			out, err := exec.Command("git", "config", "--global", "user.email").Output()
			if err != nil {
				return NotInstalled()
			}
			email := strings.TrimSpace(string(out))
			return CheckResult{Installed: email == UserEmail(), Detail: email}
		},
		GoodWhen: true,
	},
	{
		Name:        "git-name",
		Description: "Git commit author name",
		CheckFn: func() CheckResult {
			out, err := exec.Command("git", "config", "--global", "user.name").Output()
			if err != nil {
				return NotInstalled()
			}
			name := strings.TrimSpace(string(out))
			return CheckResult{Installed: name != "", Detail: name}
		},
		GoodWhen: true,
	},
	{
		Name:        "git-signing",
		Description: "Git commit signature",
		CheckFn: func() CheckResult {
			out, err := exec.Command("git", "config", "--global", "commit.gpgsign").Output()
			if err != nil {
				return NotInstalled()
			}
			return CheckResult{Installed: strings.TrimSpace(string(out)) == "true"}
		},
		GoodWhen: true,
	},
	{
		Name:        "gpg-key",
		Description: "GPG key for signing",
		CheckFn: func() CheckResult {
			out, err := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "long").Output()
			if err != nil || len(out) == 0 {
				return NotInstalled()
			}
			return InstalledWithDetail("~/.gnupg")
		},
		GoodWhen: true,
	},
	{
		Name:        "github",
		Description: "GitHub CLI authentication",
		CheckFn: func() CheckResult {
			if _, err := exec.LookPath("gh"); err != nil {
				return NotInstalled()
			}
			out, err := exec.Command("gh", "auth", "status").CombinedOutput()
			if err != nil {
				return NotInstalled()
			}
			// Parse "Logged in to github.com account username (...)"
			text := string(out)
			if idx := strings.Index(text, "account "); idx >= 0 {
				rest := text[idx+len("account "):]
				if sp := strings.IndexByte(rest, ' '); sp > 0 {
					return InstalledWithDetail(rest[:sp])
				}
			}
			return Installed()
		},
		GoodWhen: true,
	},
	{
		Name:        "ssh-key",
		Description: "SSH key for authentication",
		CheckFn: func() CheckResult {
			sshKey := os.Getenv("HOME") + "/.ssh/id_ed25519"
			if _, err := os.Stat(sshKey); err == nil {
				return InstalledWithDetail("~/.ssh/id_ed25519")
			}
			return NotInstalled()
		},
		GoodWhen: true,
	},
}
