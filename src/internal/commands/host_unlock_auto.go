package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
	"github.com/spf13/cobra"
)

var (
	unlockAutoHost           string
	unlockAutoUser           string
	unlockAutoKeychainSvc    string
	unlockAutoKeychainAcct   string
)

var hostUnlockAutoCmd = &cobra.Command{
	Use:   "unlock-auto",
	Short: "Zero-touch FileVault unlock using the Keychain-stored password",
	Long: strings.TrimSpace(`Zero-touch FileVault pre-boot SSH unlock.

The FV password is read from the macOS Keychain at run time and never written to disk:

  security find-generic-password -ws <service> -a <account>

Store it once with:

  security add-generic-password -s mac-mini-fv -a jterrazz -w

Defaults can be overridden via flags or env vars: MAC_HOST, MAC_PREBOOT_USER,
MAC_FV_KEYCHAIN_SERVICE, MAC_FV_KEYCHAIN_ACCOUNT.

Requires /usr/bin/expect (shipped with macOS).`),
	Run: func(cmd *cobra.Command, args []string) { runHostUnlockAuto() },
}

func init() {
	hostUnlockAutoCmd.Flags().StringVar(&unlockAutoHost, "host", defaultUnlockAutoHost(), "target host or IP")
	hostUnlockAutoCmd.Flags().StringVar(&unlockAutoUser, "user", defaultEnv("MAC_PREBOOT_USER", "jterrazz.agent"), "FileVault-enabled macOS user")
	hostUnlockAutoCmd.Flags().StringVar(&unlockAutoKeychainSvc, "keychain-service", defaultEnv("MAC_FV_KEYCHAIN_SERVICE", "mac-mini-fv"), "Keychain generic-password service name")
	hostUnlockAutoCmd.Flags().StringVar(&unlockAutoKeychainAcct, "keychain-account", defaultEnv("MAC_FV_KEYCHAIN_ACCOUNT", "jterrazz"), "Keychain generic-password account name")
	hostCmd.AddCommand(hostUnlockAutoCmd)
}

func runHostUnlockAuto() {
	if _, err := exec.LookPath("/usr/bin/expect"); err != nil {
		failOn(fmt.Errorf("/usr/bin/expect not found — install Tcl/expect or use `j host unlock`"))
	}

	password, err := readKeychainPassword(unlockAutoKeychainSvc, unlockAutoKeychainAcct)
	if err != nil {
		print.Error(err.Error())
		print.Dim("Add the FV password once with:")
		print.Dim("  security add-generic-password -s " + unlockAutoKeychainSvc + " -a " + unlockAutoKeychainAcct + " -w")
		os.Exit(1)
	}

	target := unlockAutoUser + "@" + unlockAutoHost
	print.SectionDivider("FILEVAULT UNLOCK (auto)")
	print.Linef("Target: %s", target)
	print.Dim("Reading password from Keychain (service=" + unlockAutoKeychainSvc + ", account=" + unlockAutoKeychainAcct + ")")
	print.Empty()

	script := buildExpectUnlockScript(target)
	cmd := exec.Command("/usr/bin/expect", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "FV_PW="+password)
	err = cmd.Run()
	// Disconnect-after-unlock is the success signal at preboot; expect's eof
	// branch exits 0, so a non-nil err here means a real failure.
	if err != nil {
		print.Warning("expect exited: " + err.Error())
		print.Dim("If the Mac was at FileVault preboot, a disconnect can still mean the unlock succeeded. Wait 30-90s, then run `j host probe`.")
		os.Exit(1)
	}
}

// readKeychainPassword shells out to `security find-generic-password -ws ...`. The
// `-w` flag prints only the password to stdout; we never log it.
func readKeychainPassword(service, account string) (string, error) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password", "-ws", service, "-a", account)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no FV password in Keychain (service=%s, account=%s)", service, account)
	}
	pw := strings.TrimRight(string(out), "\n")
	if pw == "" {
		return "", fmt.Errorf("Keychain returned empty password (service=%s, account=%s)", service, account)
	}
	return pw, nil
}

// buildExpectUnlockScript renders an expect program that:
//   - Reads the FV password from $env(FV_PW) (set by the parent process)
//   - Spawns ssh with password-auth-only, no-pubkey-auth, no host-key prompt
//   - Sends the password once when prompted, then exits on EOF
//
// The password reaches expect via the environment, not argv, so it never appears
// in `ps`; expect handles it as a quoted Tcl variable.
func buildExpectUnlockScript(target string) string {
	return strings.Join([]string{
		`set timeout 30`,
		`set pw $env(FV_PW)`,
		`spawn ssh -o PreferredAuthentications=password -o PubkeyAuthentication=no -o StrictHostKeyChecking=accept-new -o NumberOfPasswordPrompts=1 ` + target,
		`expect {`,
		`    -re "(?i)password:" { send "$pw\r" }`,
		`    timeout { puts stderr "no password prompt within timeout"; exit 1 }`,
		`    eof { puts stderr "connection closed early"; exit 1 }`,
		`}`,
		`expect {`,
		`    -re "(?i)denied" { puts stderr "FV password rejected"; exit 1 }`,
		`    eof { exit 0 }`,
		`    timeout { exit 0 }`,
		`}`,
		"",
	}, "\n")
}

func defaultUnlockAutoHost() string {
	if v := strings.TrimSpace(os.Getenv("MAC_HOST")); v != "" {
		return v
	}
	return "192.168.1.106"
}

func defaultEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
