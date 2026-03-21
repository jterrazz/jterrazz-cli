package config

import (
	"os/exec"
	"strings"
)

// SecurityCheck represents a system security verification
type SecurityCheck struct {
	Name        string
	Description string
	CheckFn     func() CheckResult
	GoodWhen    bool // true = check passes when Installed=true, false = check passes when Installed=false
}

// SecurityChecks is the list of macOS security checks
var SecurityChecks = []SecurityCheck{
	{
		Name:        "filevault",
		Description: "Full disk encryption",
		CheckFn: func() CheckResult {
			out, err := exec.Command("fdesetup", "status").Output()
			if err != nil {
				return NotInstalled()
			}
			return CheckResult{Installed: strings.Contains(string(out), "FileVault is On")}
		},
		GoodWhen: true,
	},
	{
		Name:        "firewall",
		Description: "Block incoming connections",
		CheckFn: func() CheckResult {
			out, err := exec.Command("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate").Output()
			if err != nil {
				return NotInstalled()
			}
			return CheckResult{Installed: strings.Contains(string(out), "enabled")}
		},
		GoodWhen: true,
	},
	{
		Name:        "sip",
		Description: "System Integrity Protection",
		CheckFn: func() CheckResult {
			out, err := exec.Command("csrutil", "status").Output()
			if err != nil {
				return NotInstalled()
			}
			return CheckResult{Installed: strings.Contains(string(out), "enabled")}
		},
		GoodWhen: true,
	},
	{
		Name:        "gatekeeper",
		Description: "App signature verification",
		CheckFn: func() CheckResult {
			out, err := exec.Command("spctl", "--status").Output()
			if err != nil {
				return NotInstalled()
			}
			return CheckResult{Installed: strings.Contains(string(out), "enabled")}
		},
		GoodWhen: true,
	},
	{
		Name:        "remote-login",
		Description: "SSH server disabled",
		CheckFn: func() CheckResult {
			out, err := exec.Command("launchctl", "list").Output()
			if err != nil {
				return NotInstalled()
			}
			sshRunning := strings.Contains(string(out), "com.openssh.sshd")
			return CheckResult{Installed: !sshRunning}
		},
		GoodWhen: true,
	},
	{
		Name:        "encrypted-dns",
		Description: "DNS over HTTPS/TLS",
		CheckFn: func() CheckResult {
			return CheckResult{Installed: IsDNSProfileInstalled()}
		},
		GoodWhen: true,
	},
}
