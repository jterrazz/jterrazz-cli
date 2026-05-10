package commands

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var (
	machineProbeTarget      string
	machineProbeGatewayPort int
)

const defaultGatewayPort = 18789

var machineProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Probe a remote homelab Mac (ping, ssh, gateway port, console owner)",
	Long: strings.TrimSpace(`Quick health probe of a remote homelab Mac.

Checks: ICMP reachability, SSH (BatchMode), the OpenClaw gateway port, and the
console owner reported by stat -f %Su /dev/console. Useful right after `+"`j machine restart`"+`
to confirm auto-login succeeded and lock-after-login fired.`),
	Run: func(cmd *cobra.Command, args []string) { runMachineProbe() },
}

func init() {
	machineProbeCmd.Flags().StringVar(&machineProbeTarget, "host", defaultRemoteHost(), "ssh target (host alias or user@host)")
	machineProbeCmd.Flags().IntVar(&machineProbeGatewayPort, "gateway-port", defaultGatewayPortFromEnv(), "OpenClaw gateway TCP port to probe")
	machineCmd.AddCommand(machineProbeCmd)
}

func runMachineProbe() {
	target := machineProbeTarget
	if target == "" {
		target = "mac-mini"
	}

	ip := resolveSSHHostname(target)
	if ip == "" {
		ip = target
	}

	print.SectionDivider("MACHINE PROBE")
	print.Linef("Target: %s → %s", target, ip)

	if exec.Command("ping", "-c", "1", "-W", "1000", ip).Run() == nil {
		print.Row(true, "ping", ip)
	} else {
		print.Row(false, "ping", ip)
	}

	sshErr := exec.Command("ssh",
		"-o", "ConnectTimeout=2",
		"-o", "BatchMode=yes",
		target, "true",
	).Run()
	if sshErr == nil {
		print.Row(true, "ssh", "BatchMode auth ok")
	} else {
		print.Row(false, "ssh", "auth failed (or pre-boot — try `j machine unlock`)")
	}

	if dialPort(ip, machineProbeGatewayPort, 2*time.Second) {
		print.Row(true, "gateway", "port "+strconv.Itoa(machineProbeGatewayPort)+" open")
	} else {
		print.Row(false, "gateway", "port "+strconv.Itoa(machineProbeGatewayPort)+" closed")
	}

	if sshErr == nil {
		owner, err := runQuiet("ssh",
			"-o", "ConnectTimeout=2",
			"-o", "BatchMode=yes",
			target, "stat -f %Su /dev/console",
		)
		if err == nil {
			owner = strings.TrimSpace(owner)
			label := owner
			if owner == "root" {
				label = "root (loginwindow / no GUI session)"
			}
			print.Linef("  console: %s", label)
		}
	}
}

func defaultGatewayPortFromEnv() int {
	if v := os.Getenv("OPENCLAW_GATEWAY_PORT"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return n
		}
	}
	return defaultGatewayPort
}
