package commands

import (
	"fmt"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote access connectivity",
	Long: "Connect / disconnect / inspect the Tailscale endpoint for remote\n" +
		"access. To configure the endpoint itself (mode, auth, hostname, secret),\n" +
		"use `j config` and switch to the Remote tab.",
	Run: func(cmd *cobra.Command, args []string) {
		runRemoteStatus()
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

		daemon, err := config.RemoteUp(settings)
		if err != nil {
			print.Error(err.Error())
			return
		}

		print.Success(fmt.Sprintf("Remote access connected (%s daemon)", daemon))
		if daemon == config.RemoteDaemonUserspace && config.CommandExists("caffeinate") {
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

		result, err := config.RemoteDown(settings)
		if err != nil {
			print.Error(err.Error())
			return
		}

		if len(result.Stopped) == 0 {
			print.Success("Remote access already disconnected")
			return
		}
		names := make([]string, len(result.Stopped))
		for i, daemon := range result.Stopped {
			names[i] = string(daemon)
		}
		print.Success(fmt.Sprintf("Remote access disconnected (%s daemon)", strings.Join(names, " + ")))
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

	print.Linef("Connected: %t", status.Connected)
	print.Linef("State: %s", status.BackendState)
	print.Linef("Daemon: %s", status.Daemon.Describe())
	if status.Hostname != "" {
		print.Linef("Host: %s", status.Hostname)
	}
	if status.IP != "" {
		print.Linef("IP: %s", status.IP)
	}
	if status.Daemon == config.RemoteDaemonUserspace {
		print.Linef("Keep awake: %t", status.KeepAwake)
	}
}
