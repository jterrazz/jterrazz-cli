package commands

import (
	"fmt"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	setupview "github.com/jterrazz/jterrazz-cli/src/internal/presentation/views/setup"
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote access connectivity",
	Run: func(cmd *cobra.Command, args []string) {
		runRemoteStatus()
	},
}

var remoteConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Interactive remote access configuration",
	Run: func(cmd *cobra.Command, args []string) {
		setupview.InitRemoteState()
		components.RunOrExit(setupview.RemoteConfig())
	},
}

var remoteUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Connect remote access",
	Run: func(cmd *cobra.Command, args []string) {
		settings, err := config.LoadRemoteSettings()
		if err != nil {
			print.Error(err.Error())
			return
		}

		mode, err := config.RemoteUp(settings)
		if err != nil {
			print.Error(err.Error())
			return
		}

		print.Success(fmt.Sprintf("Remote access connected (%s mode)", mode))
		if mode == config.RemoteModeUserspace && config.CommandExists("caffeinate") {
			if st, statusErr := config.RemoteStatusInfo(settings); statusErr == nil && !st.KeepAwake {
				print.Warning("Connected, but keep-awake is not active")
			}
		}
	},
}

var remoteDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Disconnect remote access",
	Run: func(cmd *cobra.Command, args []string) {
		settings, err := config.LoadRemoteSettings()
		if err != nil {
			print.Error(err.Error())
			return
		}

		mode, err := config.RemoteDown(settings)
		if err != nil {
			print.Error(err.Error())
			return
		}

		print.Success(fmt.Sprintf("Remote access disconnected (%s mode)", mode))
	},
}

var remoteStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show remote access status",
	Run: func(cmd *cobra.Command, args []string) {
		runRemoteStatus()
	},
}

func init() {
	remoteCmd.AddCommand(remoteConfigCmd)
	remoteCmd.AddCommand(remoteUpCmd)
	remoteCmd.AddCommand(remoteDownCmd)
	remoteCmd.AddCommand(remoteStatusCmd)
	rootCmd.AddCommand(remoteCmd)
}

func runRemoteStatus() {
	settings, err := config.LoadRemoteSettings()
	if err != nil {
		print.Error(err.Error())
		return
	}

	status, err := config.RemoteStatusInfo(settings)
	if err != nil {
		print.Warning("Unable to query remote runtime status")
		print.Dim(err.Error())
		print.Linef("Configured mode: %s", settings.Mode)
		print.Linef("Auth method: %s", settings.AuthMethod)
		if settings.Hostname != "" {
			print.Linef("Hostname: %s", settings.Hostname)
		}
		return
	}

	print.Linef("Mode: %s", status.Mode)
	print.Linef("State: %s", status.BackendState)
	if status.Hostname != "" {
		print.Linef("Host: %s", status.Hostname)
	}
	if status.IP != "" {
		print.Linef("IP: %s", status.IP)
	}
	print.Linef("Connected: %t", status.Connected)
	if status.Mode == config.RemoteModeUserspace {
		print.Linef("Keep awake: %t", status.KeepAwake)
	}
}
