package commands

import "github.com/jterrazz/jterrazz-cli/src/internal/config"

// init registers the four homelab Scripts with the config package so the
// j config TUI can list and toggle them. The CheckFn / RunFn / DisableFn
// implementations live in homelab_*.go.
func init() {
	config.RegisterHomelabActions(config.HomelabActions{
		AutologinEnable:       enableAutologin,
		AutologinDisable:      disableAutologin,
		AutologinCheck:        checkAutologinEnabled,
		PowerEnable:           enablePowerHarden,
		PowerDisable:          disablePowerHarden,
		PowerCheck:            checkPowerHardened,
		LockAfterLoginEnable:  enableLockAfterLogin,
		LockAfterLoginDisable: disableLockAfterLogin,
		LockAfterLoginCheck:   checkLockAfterLoginInstalled,
		SshdEnable:            enableSshd,
		SshdDisable:           disableSshd,
		SshdCheck:             checkSshdEnabled,
	})
}
