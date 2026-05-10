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

// SecurityChecks is the list of macOS security checks. Each CheckFn populates
// Detail with a short state word ("on", "off", "unknown") so the j status
// Configuration tab can render it as a right-aligned second column.
var SecurityChecks = []SecurityCheck{
	{
		Name:        "filevault",
		Description: "Full disk encryption",
		CheckFn: func() CheckResult {
			out, err := exec.Command("fdesetup", "status").Output()
			if err != nil {
				return CheckResult{Detail: "unknown"}
			}
			on := strings.Contains(string(out), "FileVault is On")
			return CheckResult{Installed: on, Detail: onOff(on)}
		},
		GoodWhen: true,
	},
	{
		Name:        "firewall",
		Description: "Block incoming connections",
		CheckFn: func() CheckResult {
			out, err := exec.Command("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate").Output()
			if err != nil {
				return CheckResult{Detail: "unknown"}
			}
			on := strings.Contains(string(out), "enabled")
			return CheckResult{Installed: on, Detail: onOff(on)}
		},
		GoodWhen: true,
	},
	{
		Name:        "sip",
		Description: "System Integrity Protection",
		CheckFn: func() CheckResult {
			out, err := exec.Command("csrutil", "status").Output()
			if err != nil {
				return CheckResult{Detail: "unknown"}
			}
			on := strings.Contains(string(out), "enabled")
			return CheckResult{Installed: on, Detail: onOff(on)}
		},
		GoodWhen: true,
	},
	{
		Name:        "gatekeeper",
		Description: "App signature verification",
		CheckFn: func() CheckResult {
			out, err := exec.Command("spctl", "--status").Output()
			if err != nil {
				return CheckResult{Detail: "unknown"}
			}
			on := strings.Contains(string(out), "enabled")
			return CheckResult{Installed: on, Detail: onOff(on)}
		},
		GoodWhen: true,
	},
	{
		Name:        "remote-login",
		Description: "SSH server disabled",
		CheckFn: func() CheckResult {
			out, err := exec.Command("launchctl", "list").Output()
			if err != nil {
				return CheckResult{Detail: "unknown"}
			}
			sshRunning := strings.Contains(string(out), "com.openssh.sshd")
			// GoodWhen=true means we want SSH off, so Installed=!sshRunning.
			// Detail reports the actual sshd state for clarity.
			return CheckResult{Installed: !sshRunning, Detail: onOff(sshRunning)}
		},
		GoodWhen: true,
	},
	{
		Name:        "encrypted-dns",
		Description: "DNS over HTTPS/TLS",
		CheckFn: func() CheckResult {
			on := IsDNSProfileInstalled()
			return CheckResult{Installed: on, Detail: onOff(on)}
		},
		GoodWhen: true,
	},
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}
