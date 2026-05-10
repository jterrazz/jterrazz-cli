package config

import (
	"fmt"
	"sort"
	"strings"
)

// Role is the function a machine plays in the user's setup. The role drives which
// status checks run, which `j config` Server items are visible, and how the
// TUI groups things.
type Role string

const (
	RoleClient Role = "client"
	RoleServer Role = "server"
)

// Machine is a single entry in the registry. A machine is "local" (the one this
// CLI is running on) when SSH is empty.
type Machine struct {
	Role     Role   `json:"role"`
	SSH      string `json:"ssh,omitempty"`
	Identity string `json:"identity,omitempty"`
}

// IsLocal reports whether the machine has no remote endpoint.
func (m Machine) IsLocal() bool { return strings.TrimSpace(m.SSH) == "" }

// Validate checks that the role is known and, if SSH is set, that it looks like user@host.
func (m Machine) Validate() error {
	switch m.Role {
	case RoleClient, RoleServer:
	default:
		return fmt.Errorf("invalid role %q (expected client or server)", m.Role)
	}
	if m.SSH != "" && !strings.Contains(m.SSH, "@") {
		return fmt.Errorf("ssh endpoint %q must be user@host", m.SSH)
	}
	return nil
}

// SelfMachine returns this machine's alias and entry, or false if no self alias is set.
func SelfMachine() (string, Machine, bool) {
	cfg, err := LoadJRC()
	if err != nil {
		return "", Machine{}, false
	}
	if cfg.Self == "" {
		return "", Machine{}, false
	}
	m, ok := cfg.Machines[cfg.Self]
	if !ok {
		return cfg.Self, Machine{}, false
	}
	return cfg.Self, m, true
}

// GetMachine returns the registry entry for the given alias.
func GetMachine(alias string) (Machine, bool) {
	cfg, err := LoadJRC()
	if err != nil {
		return Machine{}, false
	}
	m, ok := cfg.Machines[alias]
	return m, ok
}

// ListMachines returns the registry sorted by alias.
func ListMachines() []struct {
	Alias   string
	Machine Machine
} {
	cfg, err := LoadJRC()
	if err != nil {
		return nil
	}
	aliases := make([]string, 0, len(cfg.Machines))
	for a := range cfg.Machines {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)
	out := make([]struct {
		Alias   string
		Machine Machine
	}, 0, len(aliases))
	for _, a := range aliases {
		out = append(out, struct {
			Alias   string
			Machine Machine
		}{a, cfg.Machines[a]})
	}
	return out
}

// AddMachine inserts a new machine. Refuses to overwrite an existing alias.
func AddMachine(alias string, m Machine) error {
	if strings.TrimSpace(alias) == "" {
		return fmt.Errorf("alias is required")
	}
	if err := m.Validate(); err != nil {
		return err
	}
	cfg, err := LoadJRC()
	if err != nil {
		return err
	}
	if cfg.Machines == nil {
		cfg.Machines = map[string]Machine{}
	}
	if _, exists := cfg.Machines[alias]; exists {
		return fmt.Errorf("machine %q already exists (use update instead)", alias)
	}
	cfg.Machines[alias] = m
	return SaveJRC(cfg)
}

// UpdateMachine replaces an existing machine entry. Errors if the alias is unknown.
func UpdateMachine(alias string, m Machine) error {
	if err := m.Validate(); err != nil {
		return err
	}
	cfg, err := LoadJRC()
	if err != nil {
		return err
	}
	if _, exists := cfg.Machines[alias]; !exists {
		return fmt.Errorf("machine %q does not exist", alias)
	}
	cfg.Machines[alias] = m
	return SaveJRC(cfg)
}

// RemoveMachine deletes the entry for the given alias. No-op if it doesn't exist.
// Refuses to remove the alias currently marked as self.
func RemoveMachine(alias string) error {
	cfg, err := LoadJRC()
	if err != nil {
		return err
	}
	if cfg.Self == alias {
		return fmt.Errorf("refusing to remove %q while it is set as self; use `j machine init` to point self elsewhere first", alias)
	}
	if _, exists := cfg.Machines[alias]; !exists {
		return nil
	}
	delete(cfg.Machines, alias)
	return SaveJRC(cfg)
}

// SetSelf marks an existing machine alias as self. The alias must already exist.
func SetSelf(alias string) error {
	cfg, err := LoadJRC()
	if err != nil {
		return err
	}
	if _, exists := cfg.Machines[alias]; !exists {
		return fmt.Errorf("machine %q does not exist; add it first", alias)
	}
	cfg.Self = alias
	return SaveJRC(cfg)
}
