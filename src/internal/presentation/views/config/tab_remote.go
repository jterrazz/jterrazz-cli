package configview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// renderRemoteBody renders the Remote tab — a read-only summary of the
// current Tailscale settings, plus a hint to press 'i' to reconfigure.
//
// The reconfigure flow opens a generic huh form modal (buildFormModal)
// that collects mode / auth / hostname / secret and persists via
// config.SaveRemoteSettings.
func (m Model) renderRemoteBody() string {
	settings, err := config.LoadRemoteSettings()
	if err != nil {
		return contextStyle.Render(" Failed to read remote settings: " + err.Error())
	}

	hostname := settings.Hostname
	if hostname == "" {
		hostname = stateMissingStyle.Render("(unset)")
	}
	secret := stateMissingStyle.Render("(not used)")
	if settings.AuthMethod == config.RemoteAuthAuthKey {
		if settings.Secret != "" {
			secret = stateInstalledStyle.Render("configured")
		} else {
			secret = stateMissingStyle.Render("(missing)")
		}
	}

	var b strings.Builder
	b.WriteString(" " + sectionHeaderStyle.Render("Tailscale endpoint") + "\n")
	b.WriteString("\n")
	b.WriteString(remoteRow("mode", string(settings.Mode)))
	b.WriteString(remoteRow("auth", string(settings.AuthMethod)))
	b.WriteString(remoteRow("hostname", hostname))
	b.WriteString(remoteRow("secret", secret))
	b.WriteString("\n")
	b.WriteString(detailTextStyle.Render(" Press i to reconfigure (opens a form)."))
	return b.String()
}

func remoteRow(label, value string) string {
	return fmt.Sprintf("   %s   %s\n",
		itemNameMutedStyle.Render(fmt.Sprintf("%-10s", label)),
		itemNameStyle.Render(value))
}

// remoteStartConfigure opens the reconfigure modal — a 4-field huh form
// that mutates a working copy of RemoteSettings, then saves on submit.
func (m Model) remoteStartConfigure() (tea.Model, tea.Cmd) {
	current, err := config.LoadRemoteSettings()
	if err != nil {
		return m, nil
	}

	mode := string(current.Mode)
	auth := string(current.AuthMethod)
	host := current.Hostname
	secret := current.Secret

	fields := []huh.Field{
		huh.NewSelect[string]().
			Title("Mode").
			Description("auto = also covers the system Tailscale app; userspace = j's own daemon only").
			Options(
				huh.NewOption(string(config.RemoteModeAuto), string(config.RemoteModeAuto)),
				huh.NewOption(string(config.RemoteModeUserspace), string(config.RemoteModeUserspace)),
			).
			Value(&mode),
		huh.NewSelect[string]().
			Title("Auth method").
			Description("oauth = browser flow; authkey = pre-shared key").
			Options(
				huh.NewOption(string(config.RemoteAuthOAuth), string(config.RemoteAuthOAuth)),
				huh.NewOption(string(config.RemoteAuthAuthKey), string(config.RemoteAuthAuthKey)),
			).
			Value(&auth),
		huh.NewInput().
			Title("Hostname").
			Description("Tailscale hostname (leave empty to use system default)").
			Value(&host),
		huh.NewInput().
			Title("Auth key").
			Description("Required when auth method is authkey; ignored otherwise").
			EchoMode(huh.EchoModePassword).
			Value(&secret),
	}

	m.buildFormModal(
		"reconfigure remote",
		"Tailscale endpoint config — written to ~/.jterrazz/config.json on submit.",
		fields,
		func() tea.Cmd {
			next := config.RemoteSettings{
				Mode:       config.RemoteMode(mode),
				AuthMethod: config.RemoteAuthMethod(auth),
				Hostname:   strings.TrimSpace(host),
				Secret:     secret,
			}
			if next.AuthMethod != config.RemoteAuthAuthKey {
				// Don't persist a stale secret when the auth method doesn't use it.
				next.Secret = ""
			}
			return runAction("remote", "save", func() error {
				return config.SaveRemoteSettings(next)
			})
		},
	)
	return m, m.form.Init()
}

// renderRemoteFooter shows the contextual key hint for the Remote tab.
func (m Model) renderRemoteFooter() string {
	prefix := footerLabelStyle.Render(" ▶ remote  ")
	hint := footerKey("i", "reconfigure")
	return prefix + hint
}
