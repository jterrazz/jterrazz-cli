package configview

import (
	"testing"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// fixture builds a small set of scripts spanning categories and roles.
// Helpers below create CheckFn closures that report a fixed Installed value.
func fixture() []config.Script {
	return []config.Script{
		{Name: "ssh", Category: config.ScriptCategorySecurity, CheckFn: checkInstalled(true), InstallFn: noop},
		{Name: "gpg", Category: config.ScriptCategorySecurity, CheckFn: checkInstalled(false), InstallFn: noop},
		{Name: "tmux", Category: config.ScriptCategoryTerminal, CheckFn: checkInstalled(true), InstallFn: noop},
		{Name: "ghostty", Category: config.ScriptCategoryTerminal, CheckFn: checkInstalled(false), InstallFn: noop},
		{Name: "zed", Category: config.ScriptCategoryEditor, CheckFn: checkInstalled(true), InstallFn: noop},
		{Name: "autologin", Category: config.ScriptCategoryServer, Role: config.RoleServer, CheckFn: checkInstalled(false), InstallFn: noop, UninstallFn: noopErr},
		{Name: "power", Category: config.ScriptCategoryServer, Role: config.RoleServer, CheckFn: checkInstalled(true), InstallFn: noop, UninstallFn: noopErr},
		{Name: "no-category", Category: "", InstallFn: noop}, // dropped by buildSections
	}
}

func checkInstalled(b bool) func() config.CheckResult {
	return func() config.CheckResult { return config.CheckResult{Installed: b} }
}

func noop(_ config.InputValues) error { return nil }
func noopErr() error                  { return nil }

func TestBuildSectionsRoleFilterDev(t *testing.T) {
	sections := buildSectionsFrom(fixture(), config.RoleClient)
	for _, s := range sections {
		if s.Category == config.ScriptCategoryServer {
			t.Errorf("Server section visible for client role")
		}
	}
}

func TestBuildSectionsRoleFilterServer(t *testing.T) {
	sections := buildSectionsFrom(fixture(), config.RoleServer)
	var foundServer bool
	for _, s := range sections {
		if s.Category == config.ScriptCategoryServer {
			foundServer = true
			if len(s.Scripts) != 2 {
				t.Errorf("Server section has %d scripts, want 2", len(s.Scripts))
			}
		}
	}
	if !foundServer {
		t.Error("Server section missing for server role")
	}
}

func TestBuildSectionsCanonicalOrder(t *testing.T) {
	sections := buildSectionsFrom(fixture(), config.RoleServer)
	want := []config.ScriptCategory{
		config.ScriptCategoryTerminal,
		config.ScriptCategorySecurity,
		config.ScriptCategoryEditor,
		config.ScriptCategoryServer,
	}
	if len(sections) != len(want) {
		t.Fatalf("got %d sections, want %d", len(sections), len(want))
	}
	for i, s := range sections {
		if s.Category != want[i] {
			t.Errorf("sections[%d] = %q, want %q", i, s.Category, want[i])
		}
	}
}

func TestBuildSectionsItemsSortedByName(t *testing.T) {
	sections := buildSectionsFrom(fixture(), config.RoleClient)
	for _, s := range sections {
		for i := 1; i < len(s.Scripts); i++ {
			if s.Scripts[i-1].Name > s.Scripts[i].Name {
				t.Errorf("section %s not sorted: %s > %s",
					s.Category, s.Scripts[i-1].Name, s.Scripts[i].Name)
			}
		}
	}
}

func TestBuildSectionsDropsScriptsWithoutCategory(t *testing.T) {
	sections := buildSectionsFrom(fixture(), config.RoleServer)
	for _, s := range sections {
		for _, sc := range s.Scripts {
			if sc.Name == "no-category" {
				t.Error("script with empty Category was included")
			}
		}
	}
}

func TestSectionInstalledCount(t *testing.T) {
	sections := buildSectionsFrom(fixture(), config.RoleServer)
	for _, s := range sections {
		switch s.Category {
		case config.ScriptCategoryTerminal:
			installed, total := s.installedCount()
			if installed != 1 || total != 2 {
				t.Errorf("Terminal: got %d/%d, want 1/2", installed, total)
			}
		case config.ScriptCategoryServer:
			installed, total := s.installedCount()
			if installed != 1 || total != 2 {
				t.Errorf("Server: got %d/%d, want 1/2", installed, total)
			}
		}
	}
}
