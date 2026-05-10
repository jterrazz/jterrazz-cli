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

var machineProbeGatewayPort int

const defaultGatewayPort = 18789

var machineProbeCmd = &cobra.Command{
	Use:   "probe <alias>",
	Short: "Probe a remote machine (ping, ssh, gateway port, console owner)",
	Long: strings.TrimSpace(`Quick health probe of a remote machine, looked up via the registry.

Checks: ICMP reachability, SSH (BatchMode), the OpenClaw gateway port, and the
console owner reported by stat -f %Su /dev/console. Useful right after ` + "`j machine restart`" + `
to confirm auto-login succeeded and lock-after-login fired.`),
	Args: cobra.ExactArgs(1),
	Run:  func(cmd *cobra.Command, args []string) { runMachineProbe(args[0]) },
}

func init() {
	machineProbeCmd.Flags().IntVar(&machineProbeGatewayPort, "gateway-port", defaultGatewayPortFromEnv(), "OpenClaw gateway TCP port to probe")
	machineCmd.AddCommand(machineProbeCmd)
}

func runMachineProbe(alias string) {
	target := resolveRemoteSSH(alias)

	ip := resolveSSHHostname(target)
	if ip == "" {
		// fall back to the host part of user@host so the gateway-port probe still works.
		if at := strings.Index(target, "@"); at > 0 && at < len(target)-1 {
			ip = target[at+1:]
		} else {
			ip = target
		}
	}

	print.Header("j machine probe "+alias, "→ "+targetContext(alias))

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
		print.Row(false, "ssh", "auth failed (or pre-boot — try `j machine unlock "+alias+"`)")
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
