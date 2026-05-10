package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRemoteSettings(t *testing.T) {
	tests := []struct {
		name    string
		input   RemoteSettings
		wantErr bool
	}{
		{
			name:    "defaults are valid",
			input:   RemoteSettings{Mode: RemoteModeAuto, AuthMethod: RemoteAuthOAuth},
			wantErr: false,
		},
		{
			name:    "userspace oauth with secret is valid",
			input:   RemoteSettings{Mode: RemoteModeUserspace, AuthMethod: RemoteAuthOAuth, Secret: "secret"},
			wantErr: false,
		},
		{
			name:    "oauth without secret is valid",
			input:   RemoteSettings{Mode: RemoteModeUserspace, AuthMethod: RemoteAuthOAuth},
			wantErr: false,
		},
		{
			name:    "authkey without secret is invalid",
			input:   RemoteSettings{Mode: RemoteModeUserspace, AuthMethod: RemoteAuthAuthKey},
			wantErr: true,
		},
		{
			name:    "invalid mode is rejected",
			input:   RemoteSettings{Mode: "bad", AuthMethod: RemoteAuthOAuth},
			wantErr: true,
		},
		{
			name:    "invalid auth method is rejected",
			input:   RemoteSettings{Mode: RemoteModeAuto, AuthMethod: "bad"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemoteSettings(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateRemoteSettings() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoadRemoteSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	given := RemoteSettings{
		Mode:       RemoteModeUserspace,
		AuthMethod: RemoteAuthAuthKey,
		Secret:     "tskey-auth-abc",
		Hostname:   "worker",
	}

	if err := SaveRemoteSettings(given); err != nil {
		t.Fatalf("SaveRemoteSettings() error = %v", err)
	}

	got, err := LoadRemoteSettings()
	if err != nil {
		t.Fatalf("LoadRemoteSettings() error = %v", err)
	}

	if got.Mode != given.Mode {
		t.Fatalf("Mode got=%q want=%q", got.Mode, given.Mode)
	}
	if got.AuthMethod != given.AuthMethod {
		t.Fatalf("AuthMethod got=%q want=%q", got.AuthMethod, given.AuthMethod)
	}
	if got.Secret != given.Secret {
		t.Fatalf("Secret got=%q want=%q", got.Secret, given.Secret)
	}
	if got.Hostname != given.Hostname {
		t.Fatalf("Hostname got=%q want=%q", got.Hostname, given.Hostname)
	}
}

func TestParseSuggestedUpFlags(t *testing.T) {
	output := `Warning: ...
Error: changing settings via 'tailscale up' requires mentioning all
non-default flags. To proceed, either re-run your command with --reset or
use the command below to explicitly mention the current value of
all non-default settings:

        tailscale up --ssh --accept-routes --hostname old-host
`

	flags := parseSuggestedUpFlags(output)
	if len(flags) == 0 {
		t.Fatalf("expected suggested flags to be parsed")
	}

	want := []string{"--ssh", "--accept-routes", "--hostname", "old-host"}
	if len(flags) != len(want) {
		t.Fatalf("flags length got=%d want=%d (%v)", len(flags), len(want), flags)
	}
	for i := range want {
		if flags[i] != want[i] {
			t.Fatalf("flags[%d] got=%q want=%q", i, flags[i], want[i])
		}
	}
}

func TestMergeUpArgsWithSuggestedFlags(t *testing.T) {
	desired := []string{"up", "--ssh", "--hostname", "worker-new", "--auth-key", "tskey-abc"}
	suggested := []string{"--ssh", "--accept-routes", "--hostname", "worker-old"}

	merged := mergeUpArgsWithSuggestedFlags(desired, suggested)
	want := []string{"up", "--ssh", "--accept-routes", "--hostname", "worker-new", "--auth-key", "tskey-abc"}

	if len(merged) != len(want) {
		t.Fatalf("merged length got=%d want=%d (%v)", len(merged), len(want), merged)
	}
	for i := range want {
		if merged[i] != want[i] {
			t.Fatalf("merged[%d] got=%q want=%q", i, merged[i], want[i])
		}
	}
}

func TestShouldRetryWithSuggestedFlagsNewline(t *testing.T) {
	output := "Error: changing settings via 'tailscale up' requires mentioning all\nnon-default flags."
	if !shouldRetryWithSuggestedFlags(output) {
		t.Fatalf("expected retry detection to handle newline-wrapped message")
	}
}

func TestIsKeepAwakeRunningWithCurrentProcessPID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(userspaceDir(), 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(keepAwakePIDPath(), []byte("1"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// PID 1 is always running on unix systems in our supported environments.
	if !isKeepAwakeRunning() {
		t.Fatalf("expected keep-awake to be reported as running")
	}
}

func TestIsKeepAwakeRunningRemovesStalePIDFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(userspaceDir(), 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(keepAwakePIDPath(), []byte("99999999"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if isKeepAwakeRunning() {
		t.Fatalf("expected keep-awake to be reported as not running")
	}

	if _, err := os.Stat(filepath.Clean(keepAwakePIDPath())); !os.IsNotExist(err) {
		t.Fatalf("expected stale keep-awake pid file to be removed")
	}
}
