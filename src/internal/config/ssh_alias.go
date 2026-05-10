package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SSH config blocks managed by `j machine` are wrapped in a per-alias delimiter
// pair. Multiple aliases produce multiple independent blocks in ~/.ssh/config.
//
// The format intentionally embeds the alias in both the begin and end markers
// so the file is self-describing if a human ever opens it.
const (
	sshAliasMarkerPrefix = "# >>> jterrazz machine "
	sshAliasMarkerSuffix = " >>>"
	sshAliasEndPrefix    = "# <<< jterrazz machine "
	sshAliasEndSuffix    = " <<<"
)

func sshAliasBegin(alias string) string { return sshAliasMarkerPrefix + alias + sshAliasMarkerSuffix }
func sshAliasEnd(alias string) string   { return sshAliasEndPrefix + alias + sshAliasEndSuffix }

// sshConfigPath returns ~/.ssh/config or empty string if HOME is unset.
func sshConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// WriteSSHAlias inserts or refreshes a managed `Host <alias>` block in
// ~/.ssh/config. Refuses (returns an error) if a conflicting `Host <alias>`
// line exists outside of any managed block.
//
// Requires m.SSH to be non-empty and parseable as user@host.
func WriteSSHAlias(alias string, m Machine) error {
	if strings.TrimSpace(alias) == "" {
		return fmt.Errorf("alias is required")
	}
	user, host, err := parseSSHEndpoint(m.SSH)
	if err != nil {
		return err
	}

	path := sshConfigPath()
	if path == "" {
		return fmt.Errorf("cannot resolve ~/.ssh/config: HOME unset")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	block := buildSSHAliasBlock(alias, host, user, m.Identity)
	updated, conflict := upsertSSHAliasBlock(existing, alias, block)
	if conflict != "" {
		return fmt.Errorf("refusing to modify %s: %s", path, conflict)
	}
	if bytes.Equal(updated, existing) {
		return nil
	}
	return os.WriteFile(path, updated, 0o600)
}

// RemoveSSHAlias removes the managed `Host <alias>` block from ~/.ssh/config.
// No-op if the file or the block does not exist.
func RemoveSSHAlias(alias string) error {
	path := sshConfigPath()
	if path == "" {
		return nil
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	updated := removeSSHAliasBlock(existing, alias)
	if bytes.Equal(updated, existing) {
		return nil
	}
	return os.WriteFile(path, updated, 0o600)
}

func parseSSHEndpoint(endpoint string) (user, host string, err error) {
	endpoint = strings.TrimSpace(endpoint)
	at := strings.Index(endpoint, "@")
	if at <= 0 || at == len(endpoint)-1 {
		return "", "", fmt.Errorf("ssh endpoint %q must be user@host", endpoint)
	}
	return endpoint[:at], endpoint[at+1:], nil
}

func buildSSHAliasBlock(alias, host, user, identity string) string {
	if identity == "" {
		identity = "~/.ssh/id_ed25519"
	}
	return strings.Join([]string{
		sshAliasBegin(alias),
		"Host " + alias,
		"  HostName " + host,
		"  User " + user,
		"  IdentityFile " + identity,
		"  ServerAliveInterval 30",
		"  ServerAliveCountMax 3",
		sshAliasEnd(alias),
		"",
	}, "\n")
}

// upsertSSHAliasBlock replaces this alias's managed block if it exists, or
// appends if not. Returns the new file contents and a non-empty conflict
// description if a foreign `Host <alias>` line exists outside any managed block.
func upsertSSHAliasBlock(existing []byte, alias, block string) ([]byte, string) {
	text := string(existing)
	begin := sshAliasBegin(alias)
	end := sshAliasEnd(alias)

	beginIdx := strings.Index(text, begin)
	endIdx := strings.Index(text, end)
	if beginIdx >= 0 && endIdx > beginIdx {
		endLine := endIdx + len(end)
		if endLine < len(text) && text[endLine] == '\n' {
			endLine++
		}
		return []byte(text[:beginIdx] + block + text[endLine:]), ""
	}

	if conflict := findForeignHostAlias(text, alias); conflict != "" {
		return existing, conflict
	}

	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" && !strings.HasSuffix(text, "\n\n") {
		text += "\n"
	}
	return []byte(text + block), ""
}

func removeSSHAliasBlock(existing []byte, alias string) []byte {
	text := string(existing)
	begin := sshAliasBegin(alias)
	end := sshAliasEnd(alias)

	beginIdx := strings.Index(text, begin)
	endIdx := strings.Index(text, end)
	if beginIdx < 0 || endIdx <= beginIdx {
		return existing
	}
	endLine := endIdx + len(end)
	if endLine < len(text) && text[endLine] == '\n' {
		endLine++
	}
	return []byte(text[:beginIdx] + text[endLine:])
}

// findForeignHostAlias returns a human-readable description if a `Host <alias>`
// line exists outside of any managed block. "Managed" means inside any
// `# >>> jterrazz machine X >>>` ... `# <<< jterrazz machine X <<<` pair, not
// just this alias's block — neighbouring managed blocks belong to other aliases
// and shouldn't be mistaken for foreign.
func findForeignHostAlias(text, alias string) string {
	inManaged := false
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, sshAliasMarkerPrefix) && strings.HasSuffix(trimmed, sshAliasMarkerSuffix) {
			inManaged = true
			continue
		}
		if strings.HasPrefix(trimmed, sshAliasEndPrefix) && strings.HasSuffix(trimmed, sshAliasEndSuffix) {
			inManaged = false
			continue
		}
		if inManaged {
			continue
		}
		if !strings.HasPrefix(trimmed, "Host ") {
			continue
		}
		for _, name := range strings.Fields(trimmed)[1:] {
			if name == alias {
				return fmt.Sprintf("Host %s already defined at line %d (outside any managed block)", alias, i+1)
			}
		}
	}
	return ""
}
