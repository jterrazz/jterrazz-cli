package commands

import (
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

// machineContext returns the canonical "<alias> · <role>" context string for
// the machine this CLI is running on. Used as the right-hand context of
// every local command's header.
//
// Returns a muted "(unregistered)" placeholder when no self alias is set in
// ~/.jterrazz/config.json — typically a brand-new install before
// `j machine init` ran.
func machineContext() string {
	alias, m, ok := config.SelfMachine()
	if !ok {
		return print.MutedText("(unregistered)")
	}
	return alias + " · " + print.RenderRole(string(m.Role))
}

// targetContext returns the canonical "<alias> · <role>" context string for
// a remote target. Falls back to bare alias if the registry doesn't know it
// (shouldn't normally happen — the alias-aware verbs validate first).
func targetContext(alias string) string {
	m, ok := config.GetMachine(alias)
	if !ok {
		return alias
	}
	return alias + " · " + print.RenderRole(string(m.Role))
}
