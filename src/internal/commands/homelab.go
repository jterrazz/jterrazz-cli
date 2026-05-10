package commands

import "github.com/jterrazz/jterrazz-cli/src/internal/config"

// init registers the four homelab Scripts with the config package so the
// j config TUI can list and toggle them. The CheckFn / InstallFn / UninstallFn
// implementations live in homelab_*.go.
func init() {
	config.RegisterHomelabActions(config.HomelabActions{
		AutologinInstall:        installAutologin,
		AutologinUninstall:      uninstallAutologin,
		AutologinCheck:          checkAutologinInstalled,
		PowerInstall:            installPower,
		PowerUninstall:          uninstallPower,
		PowerCheck:              checkPowerInstalled,
		LockAfterLoginInstall:   installLockAfterLogin,
		LockAfterLoginUninstall: uninstallLockAfterLogin,
		LockAfterLoginCheck:     checkLockAfterLoginInstalled,
		SshdInstall:             installSshd,
		SshdUninstall:           uninstallSshd,
		SshdCheck:               checkSshdInstalled,
	})
}
