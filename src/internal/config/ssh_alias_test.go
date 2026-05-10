package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSSHEndpoint(t *testing.T) {
	tests := []struct {
		in       string
		wantUser string
		wantHost string
		wantErr  bool
	}{
		{"agent@192.168.1.106", "agent", "192.168.1.106", false},
		{" admin@host ", "admin", "host", false},
		{"justhost", "", "", true},
		{"@host", "", "", true},
		{"user@", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		u, h, err := parseSSHEndpoint(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseSSHEndpoint(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
		}
		if u != tt.wantUser || h != tt.wantHost {
			t.Errorf("parseSSHEndpoint(%q) = (%q, %q), want (%q, %q)", tt.in, u, h, tt.wantUser, tt.wantHost)
		}
	}
}

func TestWriteSSHAliasNewFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := WriteSSHAlias("mac-mini", Machine{
		Role: RoleHomelab,
		SSH:  "agent@192.168.1.106",
	})
	if err != nil {
		t.Fatalf("WriteSSHAlias error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"# >>> jterrazz machine mac-mini >>>",
		"Host mac-mini",
		"  HostName 192.168.1.106",
		"  User agent",
		"  IdentityFile ~/.ssh/id_ed25519",
		"# <<< jterrazz machine mac-mini <<<",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- got:\n%s", want, got)
		}
	}
}

func TestWriteSSHAliasReplacesExistingBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := WriteSSHAlias("mac-mini", Machine{Role: RoleHomelab, SSH: "old@1.1.1.1"}); err != nil {
		t.Fatalf("first write error = %v", err)
	}
	if err := WriteSSHAlias("mac-mini", Machine{Role: RoleHomelab, SSH: "new@2.2.2.2"}); err != nil {
		t.Fatalf("second write error = %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	got := string(data)
	if strings.Contains(got, "old@") || strings.Contains(got, "1.1.1.1") {
		t.Errorf("expected old block to be replaced; got:\n%s", got)
	}
	if !strings.Contains(got, "  User new") || !strings.Contains(got, "  HostName 2.2.2.2") {
		t.Errorf("expected new block; got:\n%s", got)
	}
	if strings.Count(got, "# >>> jterrazz machine mac-mini >>>") != 1 {
		t.Errorf("expected exactly one managed block; got:\n%s", got)
	}
}

func TestWriteSSHAliasAppendsAlongside(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := WriteSSHAlias("mac-mini", Machine{Role: RoleHomelab, SSH: "a@h1"}); err != nil {
		t.Fatalf("first write error = %v", err)
	}
	if err := WriteSSHAlias("worker", Machine{Role: RoleHomelab, SSH: "b@h2"}); err != nil {
		t.Fatalf("second write error = %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	got := string(data)
	if !strings.Contains(got, "Host mac-mini") || !strings.Contains(got, "Host worker") {
		t.Errorf("expected both managed blocks; got:\n%s", got)
	}
}

func TestWriteSSHAliasRefusesForeignHost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgPath := filepath.Join(home, ".ssh", "config")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	preExisting := "Host mac-mini\n  HostName legacy\n  User legacy\n"
	if err := os.WriteFile(cfgPath, []byte(preExisting), 0o600); err != nil {
		t.Fatal(err)
	}

	err := WriteSSHAlias("mac-mini", Machine{Role: RoleHomelab, SSH: "agent@1.2.3.4"})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "outside any managed block") {
		t.Errorf("error doesn't mention conflict: %v", err)
	}
	data, _ := os.ReadFile(cfgPath)
	if string(data) != preExisting {
		t.Errorf("file should be unchanged; got:\n%s", string(data))
	}
}

func TestWriteSSHAliasRejectsBadEndpoint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := WriteSSHAlias("a", Machine{Role: RoleHomelab, SSH: "no-at-sign"}); err == nil {
		t.Fatal("expected error on malformed endpoint")
	}
}

func TestRemoveSSHAlias(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_ = WriteSSHAlias("mac-mini", Machine{Role: RoleHomelab, SSH: "a@h"})
	_ = WriteSSHAlias("worker", Machine{Role: RoleHomelab, SSH: "b@h2"})
	if err := RemoveSSHAlias("mac-mini"); err != nil {
		t.Fatalf("RemoveSSHAlias error = %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	got := string(data)
	if strings.Contains(got, "mac-mini") {
		t.Errorf("expected mac-mini block removed; got:\n%s", got)
	}
	if !strings.Contains(got, "Host worker") {
		t.Errorf("expected worker block preserved; got:\n%s", got)
	}
}

func TestRemoveSSHAliasMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := RemoveSSHAlias("nope"); err != nil {
		t.Fatalf("expected no-op on missing file, got %v", err)
	}
}

func TestWriteSSHAliasCustomIdentity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := WriteSSHAlias("mac-mini", Machine{
		Role:     RoleHomelab,
		SSH:      "agent@h",
		Identity: "~/.ssh/custom_key",
	})
	if err != nil {
		t.Fatalf("WriteSSHAlias error = %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if !strings.Contains(string(data), "  IdentityFile ~/.ssh/custom_key") {
		t.Errorf("expected custom identity in output; got:\n%s", string(data))
	}
}
