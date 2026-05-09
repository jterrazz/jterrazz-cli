package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

const (
	sshConfigBlockBegin = "# >>> jterrazz host ssh-config (mac-mini) >>>"
	sshConfigBlockEnd   = "# <<< jterrazz host ssh-config (mac-mini) <<<"
)

var (
	sshConfigAlias    string
	sshConfigHostname string
	sshConfigUser     string
	sshConfigKey      string
	sshConfigPath     string
)

var hostSSHConfigCmd = &cobra.Command{
	Use:   "ssh-config",
	Short: "Manage the ~/.ssh/config block for the homelab Mac",
	Long: strings.TrimSpace(`Idempotently install or update an SSH config block in ~/.ssh/config so the
homelab Mac is reachable via a friendly alias from the MacBook.

If a Host block with the same alias already exists outside of the managed delimiters,
the command refuses to overwrite it and exits non-zero — the SSH config is too
load-bearing to silently replace.`),
}

var hostSSHConfigInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Insert or update the managed SSH config block",
	Run:   func(cmd *cobra.Command, args []string) { runHostSSHConfigInstall() },
}

var hostSSHConfigStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the managed block is present and what `ssh -G` resolves",
	Run:   func(cmd *cobra.Command, args []string) { runHostSSHConfigStatus() },
}

func init() {
	for _, c := range []*cobra.Command{hostSSHConfigInstallCmd, hostSSHConfigStatusCmd} {
		c.Flags().StringVar(&sshConfigAlias, "alias", "mac-mini", "Host alias to define")
		c.Flags().StringVar(&sshConfigHostname, "hostname", "192.168.1.106", "HostName value")
		c.Flags().StringVar(&sshConfigUser, "user", "jterrazz.agent", "User value")
		c.Flags().StringVar(&sshConfigKey, "identity", "~/.ssh/id_ed25519", "IdentityFile value")
		c.Flags().StringVar(&sshConfigPath, "path", defaultSSHConfigPath(), "ssh_config file to manage")
	}
	hostSSHConfigCmd.AddCommand(hostSSHConfigInstallCmd, hostSSHConfigStatusCmd)
	hostCmd.AddCommand(hostSSHConfigCmd)
}

func runHostSSHConfigInstall() {
	path := sshConfigPath
	if path == "" {
		path = defaultSSHConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		failOn(err)
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		failOn(err)
	}

	block := buildSSHConfigBlock()
	updated, conflict := updateSSHConfig(existing, block, sshConfigAlias)
	if conflict != "" {
		print.SectionDivider("SSH-CONFIG INSTALL")
		print.Warning("Refusing to modify " + path + ":")
		print.Dim("  " + conflict)
		print.Dim("Resolve manually, then re-run.")
		os.Exit(1)
	}

	if bytes.Equal(updated, existing) {
		print.SectionDivider("SSH-CONFIG INSTALL")
		print.Success("Block already up to date in " + path)
		return
	}

	if err := os.WriteFile(path, updated, 0o600); err != nil {
		failOn(err)
	}

	print.SectionDivider("SSH-CONFIG INSTALL")
	print.Success("Updated " + path)
	print.Dim("Verify: `ssh -G " + sshConfigAlias + "` or `j host probe --host " + sshConfigAlias + "`")
}

func runHostSSHConfigStatus() {
	path := sshConfigPath
	if path == "" {
		path = defaultSSHConfigPath()
	}

	print.SectionDivider("SSH-CONFIG STATUS")
	print.Linef("File: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		print.Linef("  state: %s", err.Error())
		return
	}
	if managedBlockPresent(data) {
		print.Linef("  managed block: present")
	} else {
		print.Linef("  managed block: absent")
	}
	if hostBlockHasAlias(data, sshConfigAlias) {
		print.Linef("  Host %s: defined", sshConfigAlias)
	} else {
		print.Linef("  Host %s: not defined", sshConfigAlias)
	}

	out, err := runQuiet("ssh", "-G", sshConfigAlias)
	if err != nil {
		print.Linef("  ssh -G: error %s", err.Error())
		return
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "hostname", "user", "identityfile", "serveraliveinterval":
			print.Linef("  ssh -G %s: %s", fields[0], strings.Join(fields[1:], " "))
		}
	}
}

func buildSSHConfigBlock() string {
	return strings.Join([]string{
		sshConfigBlockBegin,
		"Host " + sshConfigAlias,
		"  HostName " + sshConfigHostname,
		"  User " + sshConfigUser,
		"  IdentityFile " + sshConfigKey,
		"  ServerAliveInterval 30",
		"  ServerAliveCountMax 3",
		sshConfigBlockEnd,
		"",
	}, "\n")
}

// updateSSHConfig returns the new file contents and a non-empty conflict string if
// a foreign Host block with the same alias exists outside the managed delimiters.
func updateSSHConfig(existing []byte, block, alias string) ([]byte, string) {
	text := string(existing)

	beginIdx := strings.Index(text, sshConfigBlockBegin)
	endIdx := strings.Index(text, sshConfigBlockEnd)

	if beginIdx >= 0 && endIdx > beginIdx {
		// Replace the managed block in place.
		endLine := endIdx + len(sshConfigBlockEnd)
		// Consume the trailing newline if any.
		if endLine < len(text) && text[endLine] == '\n' {
			endLine++
		}
		updated := text[:beginIdx] + block + text[endLine:]
		return []byte(updated), ""
	}

	// No managed block. Check for a foreign Host alias entry.
	if conflict := findForeignHostBlock(text, alias); conflict != "" {
		return existing, conflict
	}

	// Append the managed block, ensuring a separating newline.
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" && !strings.HasSuffix(text, "\n\n") {
		text += "\n"
	}
	return []byte(text + block), ""
}

// findForeignHostBlock returns a human-readable conflict description if a Host line
// with the given alias exists in the (un-managed portion of the) file.
func findForeignHostBlock(text, alias string) string {
	scanner := strings.Split(text, "\n")
	inManaged := false
	for i, line := range scanner {
		trimmed := strings.TrimSpace(line)
		if trimmed == sshConfigBlockBegin {
			inManaged = true
			continue
		}
		if trimmed == sshConfigBlockEnd {
			inManaged = false
			continue
		}
		if inManaged {
			continue
		}
		if !strings.HasPrefix(trimmed, "Host ") {
			continue
		}
		fields := strings.Fields(trimmed)
		for _, name := range fields[1:] {
			if name == alias {
				return fmt.Sprintf("Host %s already defined at line %d (outside managed block)", alias, i+1)
			}
		}
	}
	return ""
}

func managedBlockPresent(data []byte) bool {
	text := string(data)
	return strings.Contains(text, sshConfigBlockBegin) && strings.Contains(text, sshConfigBlockEnd)
}

func hostBlockHasAlias(data []byte, alias string) bool {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "Host ") {
			continue
		}
		for _, name := range strings.Fields(trimmed)[1:] {
			if name == alias {
				return true
			}
		}
	}
	return false
}

func defaultSSHConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}
