package commands

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var machineRestartConfirmed bool

var machineRestartCmd = &cobra.Command{
	Use:   "restart <alias>",
	Short: "Software-reboot a remote homelab Mac (FileVault-aware authrestart)",
	Long: strings.TrimSpace(`Issue a FileVault-aware software reboot of the homelab Mac via SSH.

The remote runs ` + "`sudo fdesetup authrestart -delayminutes 0`" + ` which captures the
FileVault unlock token in memory; the next boot skips FV and auto-login lands the
agent session ~60s later.

Requires --yes — this is a destructive remote action.`),
	Args: cobra.ExactArgs(1),
	Run:  func(cmd *cobra.Command, args []string) { runMachineRestart(args[0]) },
}

func init() {
	machineRestartCmd.Flags().BoolVarP(&machineRestartConfirmed, "yes", "y", false, "skip the interactive confirmation prompt")
	machineCmd.AddCommand(machineRestartCmd)
}

func runMachineRestart(alias string) {
	target := resolveRemoteSSH(alias)

	print.Header("j machine restart "+alias, target)
	print.Dim(" Will issue: sudo fdesetup authrestart -delayminutes 0")
	print.Empty()

	if !machineRestartConfirmed {
		failOn(fmt.Errorf("refusing to reboot without --yes; re-run with `j machine restart %s --yes`", alias))
	}

	cmd := exec.Command("ssh", target, "sudo fdesetup authrestart -delayminutes 0")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Connection drop is expected the moment the Mac begins to restart.
		print.Dim("ssh exited (expected — the Mac is restarting): " + strings.TrimSpace(string(out)))
	}

	ip := resolveSSHHostname(target)
	print.Linef("Waiting for %s to come back…", alias)
	if ip != "" {
		waitForPing(ip, 60)
	}
	if waitForSSH(target, 60) {
		print.Success("SSH ready after restart")
		return
	}
	failOn(fmt.Errorf("SSH did not come back within budget — check `j machine probe %s`", alias))
}

func resolveSSHHostname(target string) string {
	out, err := runQuiet("ssh", "-G", target)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "hostname" {
			return fields[1]
		}
	}
	return ""
}

func waitForPing(ip string, attempts int) bool {
	for i := 0; i < attempts; i++ {
		if err := exec.Command("ping", "-c", "1", "-W", "1000", ip).Run(); err == nil {
			print.Success(fmt.Sprintf("ping reachable after ~%ds", i*2))
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

func waitForSSH(target string, attempts int) bool {
	for i := 0; i < attempts; i++ {
		err := exec.Command("ssh",
			"-o", "ConnectTimeout=2",
			"-o", "BatchMode=yes",
			target,
			"true",
		).Run()
		if err == nil {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// dialPort is a small helper used by machine_probe.
func dialPort(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
