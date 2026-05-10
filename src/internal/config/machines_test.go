package config

import (
	"testing"
)

func TestMachineValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   Machine
		wantErr bool
	}{
		{"local dev is valid", Machine{Role: RoleDev}, false},
		{"remote homelab is valid", Machine{Role: RoleHomelab, SSH: "user@host"}, false},
		{"unknown role is rejected", Machine{Role: "server"}, true},
		{"empty role is rejected", Machine{}, true},
		{"ssh without @ is rejected", Machine{Role: RoleHomelab, SSH: "host"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.input.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestAddListGetRemove(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := AddMachine("macbook", Machine{Role: RoleDev}); err != nil {
		t.Fatalf("AddMachine(macbook) error = %v", err)
	}
	if err := AddMachine("mac-mini", Machine{Role: RoleHomelab, SSH: "agent@192.168.1.106"}); err != nil {
		t.Fatalf("AddMachine(mac-mini) error = %v", err)
	}

	list := ListMachines()
	if len(list) != 2 {
		t.Fatalf("ListMachines() len = %d, want 2", len(list))
	}
	if list[0].Alias != "mac-mini" || list[1].Alias != "macbook" {
		t.Fatalf("ListMachines() not sorted: %+v", list)
	}

	got, ok := GetMachine("mac-mini")
	if !ok || got.SSH != "agent@192.168.1.106" {
		t.Fatalf("GetMachine(mac-mini) = (%+v, %v)", got, ok)
	}

	if err := RemoveMachine("mac-mini"); err != nil {
		t.Fatalf("RemoveMachine(mac-mini) error = %v", err)
	}
	if _, ok := GetMachine("mac-mini"); ok {
		t.Fatal("GetMachine(mac-mini) still found after RemoveMachine")
	}
	if err := RemoveMachine("does-not-exist"); err != nil {
		t.Fatalf("RemoveMachine(missing) should be no-op, got %v", err)
	}
}

func TestAddRefusesDuplicate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := AddMachine("macbook", Machine{Role: RoleDev}); err != nil {
		t.Fatalf("first AddMachine error = %v", err)
	}
	if err := AddMachine("macbook", Machine{Role: RoleHomelab}); err == nil {
		t.Fatal("AddMachine of duplicate should fail, got nil")
	}
}

func TestSetSelfAndSelfMachine(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := SetSelf("macbook"); err == nil {
		t.Fatal("SetSelf on unknown alias should fail, got nil")
	}

	_ = AddMachine("macbook", Machine{Role: RoleDev})
	if err := SetSelf("macbook"); err != nil {
		t.Fatalf("SetSelf(macbook) error = %v", err)
	}

	alias, m, ok := SelfMachine()
	if !ok || alias != "macbook" || m.Role != RoleDev {
		t.Fatalf("SelfMachine() = (%q, %+v, %v)", alias, m, ok)
	}
}

func TestRemoveSelfRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_ = AddMachine("macbook", Machine{Role: RoleDev})
	_ = SetSelf("macbook")

	if err := RemoveMachine("macbook"); err == nil {
		t.Fatal("RemoveMachine(self) should fail, got nil")
	}
}

func TestUpdateMachine(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := UpdateMachine("ghost", Machine{Role: RoleDev}); err == nil {
		t.Fatal("UpdateMachine on unknown alias should fail, got nil")
	}

	_ = AddMachine("mac-mini", Machine{Role: RoleHomelab, SSH: "old@host"})
	if err := UpdateMachine("mac-mini", Machine{Role: RoleHomelab, SSH: "new@host"}); err != nil {
		t.Fatalf("UpdateMachine error = %v", err)
	}
	got, _ := GetMachine("mac-mini")
	if got.SSH != "new@host" {
		t.Fatalf("UpdateMachine didn't take, got SSH=%q", got.SSH)
	}
}

func TestMachineIsLocal(t *testing.T) {
	if !(Machine{Role: RoleDev}).IsLocal() {
		t.Fatal("dev without SSH should be local")
	}
	if (Machine{Role: RoleHomelab, SSH: "u@h"}).IsLocal() {
		t.Fatal("machine with SSH should not be local")
	}
}

func TestRoundTripPersistence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	in := Machine{Role: RoleHomelab, SSH: "agent@192.168.1.106", Identity: "~/.ssh/id_ed25519"}
	if err := AddMachine("mac-mini", in); err != nil {
		t.Fatalf("AddMachine error = %v", err)
	}

	cfg, err := LoadJRC()
	if err != nil {
		t.Fatalf("LoadJRC error = %v", err)
	}
	out, ok := cfg.Machines["mac-mini"]
	if !ok {
		t.Fatal("mac-mini not in reloaded config")
	}
	if out != in {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}
