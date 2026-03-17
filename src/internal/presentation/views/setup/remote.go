package setup

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
)

type remoteAction string

const (
	remoteActionMode     remoteAction = "mode"
	remoteActionAuth     remoteAction = "auth"
	remoteActionHostname remoteAction = "hostname"
	remoteActionSecret   remoteAction = "secret"
	remoteActionSave     remoteAction = "save"
)

type remoteItemData struct {
	action remoteAction
}

type remoteFieldUpdatedMsg struct {
	field string
	value string
	err   error
}

type remoteState struct {
	settings config.RemoteSettings
	itemData []remoteItemData
}

var remote remoteState

// InitRemoteState initializes remote setup state.
func InitRemoteState() {
	settings, err := config.LoadRemoteSettings()
	if err != nil {
		settings = config.RemoteSettings{
			Mode:       config.RemoteModeUserspace,
			AuthMethod: config.RemoteAuthOAuth,
		}
	}

	remote = remoteState{
		settings: settings,
		itemData: nil,
	}
}

// BuildRemoteItems builds the remote setup menu items.
func BuildRemoteItems() []components.Item {
	var items []components.Item
	remote.itemData = nil

	items = append(items, components.Item{Kind: components.KindHeader, Label: "Remote"})
	remote.itemData = append(remote.itemData, remoteItemData{})

	items = append(items, components.Item{
		Kind:        components.KindNavigation,
		Label:       "mode",
		Description: string(remote.settings.Mode),
	})
	remote.itemData = append(remote.itemData, remoteItemData{action: remoteActionMode})

	items = append(items, components.Item{
		Kind:        components.KindNavigation,
		Label:       "auth method",
		Description: string(remote.settings.AuthMethod),
	})
	remote.itemData = append(remote.itemData, remoteItemData{action: remoteActionAuth})

	hostDesc := remote.settings.Hostname
	if hostDesc == "" {
		hostDesc = "-"
	}
	items = append(items, components.Item{
		Kind:        components.KindNavigation,
		Label:       "hostname",
		Description: hostDesc,
	})
	remote.itemData = append(remote.itemData, remoteItemData{action: remoteActionHostname})

	secretDesc := "-"
	if remote.settings.AuthMethod != config.RemoteAuthAuthKey {
		secretDesc = "not used"
	} else if remote.settings.Secret != "" {
		secretDesc = "configured"
	}
	items = append(items, components.Item{
		Kind:        components.KindNavigation,
		Label:       "secret",
		Description: secretDesc,
	})
	remote.itemData = append(remote.itemData, remoteItemData{action: remoteActionSecret})

	items = append(items, components.Item{Kind: components.KindHeader, Label: "Actions"})
	remote.itemData = append(remote.itemData, remoteItemData{})

	items = append(items, components.Item{
		Kind:        components.KindNavigation,
		Label:       "save",
		Description: "~/.jterrazz/config.json",
	})
	remote.itemData = append(remote.itemData, remoteItemData{action: remoteActionSave})

	return items
}

// HandleRemoteSelect handles item selection in the remote menu.
func HandleRemoteSelect(index int, item components.Item) tea.Cmd {
	if index >= len(remote.itemData) {
		return nil
	}
	data := remote.itemData[index]

	switch data.action {
	case remoteActionMode:
		remote.settings.Mode = nextRemoteMode(remote.settings.Mode)
		return func() tea.Msg { return components.RefreshMsg{} }

	case remoteActionAuth:
		remote.settings.AuthMethod = nextRemoteAuthMethod(remote.settings.AuthMethod)
		if remote.settings.AuthMethod == config.RemoteAuthOAuth {
			remote.settings.Secret = ""
		}
		return func() tea.Msg { return components.RefreshMsg{} }

	case remoteActionHostname:
		return promptRemoteFieldCmd(
			"hostname",
			"Hostname (empty to clear): ",
			false,
		)

	case remoteActionSecret:
		if remote.settings.AuthMethod != config.RemoteAuthAuthKey {
			return func() tea.Msg {
				return components.ActionDoneMsg{Message: "Auth key is only used with auth method = authkey"}
			}
		}
		return promptRemoteFieldCmd(
			"secret",
			"Auth key (empty to clear): ",
			true,
		)

	case remoteActionSave:
		return func() tea.Msg {
			if err := config.SaveRemoteSettings(remote.settings); err != nil {
				return components.ActionDoneMsg{Message: "Error: " + err.Error(), Err: err}
			}
			return components.ActionDoneMsg{Message: "Saved remote config"}
		}
	}

	return nil
}

// HandleRemoteMessage handles messages for the remote view.
func HandleRemoteMessage(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case remoteFieldUpdatedMsg:
		if msg.err != nil {
			return func() tea.Msg {
				return components.ActionDoneMsg{Message: "Error: " + msg.err.Error(), Err: msg.err}
			}
		}

		value := strings.TrimSpace(msg.value)
		switch msg.field {
		case "hostname":
			remote.settings.Hostname = value
		case "secret":
			remote.settings.Secret = value
		}
		return func() tea.Msg { return components.RefreshMsg{} }
	}
	return nil
}

// RemoteConfig returns the TUI config for the remote view.
func RemoteConfig() components.AppConfig {
	return components.AppConfig{
		Title:      "Remote",
		BuildItems: BuildRemoteItems,
		OnSelect:   HandleRemoteSelect,
		OnMessage:  HandleRemoteMessage,
	}
}

func nextRemoteMode(mode config.RemoteMode) config.RemoteMode {
	order := []config.RemoteMode{
		config.RemoteModeAuto,
		config.RemoteModeUserspace,
	}
	for i := range order {
		if order[i] == mode {
			return order[(i+1)%len(order)]
		}
	}
	return config.RemoteModeUserspace
}

func nextRemoteAuthMethod(method config.RemoteAuthMethod) config.RemoteAuthMethod {
	order := []config.RemoteAuthMethod{
		config.RemoteAuthOAuth,
		config.RemoteAuthAuthKey,
	}
	for i := range order {
		if order[i] == method {
			return order[(i+1)%len(order)]
		}
	}
	return config.RemoteAuthOAuth
}

func promptRemoteFieldCmd(field, prompt string, hidden bool) tea.Cmd {
	tmpFile, err := os.CreateTemp("", "jremote-"+field+"-*")
	if err != nil {
		return func() tea.Msg {
			return remoteFieldUpdatedMsg{field: field, err: err}
		}
	}
	path := tmpFile.Name()
	tmpFile.Close()

	script := ""
	if hidden {
		script = "printf " + shellQuote(prompt) + "; stty -echo; IFS= read -r v; stty echo; printf '\\n'; printf '%s' \"$v\" > \"$1\""
	} else {
		script = "printf " + shellQuote(prompt) + "; IFS= read -r v; printf '%s' \"$v\" > \"$1\""
	}

	cmd := exec.Command("sh", "-c", script, "sh", path)
	return tea.ExecProcess(cmd, func(execErr error) tea.Msg {
		defer os.Remove(path)
		if execErr != nil {
			return remoteFieldUpdatedMsg{field: field, err: execErr}
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return remoteFieldUpdatedMsg{field: field, err: readErr}
		}
		return remoteFieldUpdatedMsg{field: field, value: string(data)}
	})
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
