package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

// requireDarwin fails fast on non-macOS hosts. Local-host configuration commands
// only make sense on macOS; everything else is a misuse.
func requireDarwin() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("this command only runs on macOS (got %s)", runtime.GOOS)
	}
	return nil
}

// requireRoot returns an error if the process is not effectively root.
// Print a hint with the original argv so the user can re-run with sudo.
func requireRoot() error {
	if os.Geteuid() == 0 {
		return nil
	}
	return errors.New("requires root — re-run with: sudo " + strings.Join(os.Args, " "))
}

// run executes a command, streams output to the user, and returns the error.
// Used for state-changing commands where the user benefits from seeing live output.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runQuiet captures combined output without streaming. Used when we want to
// inspect or reformat the result before showing the user.
func runQuiet(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// failOn prints the error and exits with status 1. Use after every required
// step in state-changing commands so the user sees a clear failure point.
func failOn(err error) {
	if err == nil {
		return
	}
	print.Error(err.Error())
	os.Exit(1)
}
