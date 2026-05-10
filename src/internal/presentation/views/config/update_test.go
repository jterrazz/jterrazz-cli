package configview

import (
	"testing"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// modelWithSections constructs a Model directly from a synthetic section list,
// bypassing the package-level config.Scripts.
func modelWithSections(sections []Section) Model {
	return Model{
		sections: sections,
		cursor:   firstItemCursor(sections),
		expanded: map[string]bool{},
	}
}

func makeSection(cat config.ScriptCategory, names ...string) Section {
	scripts := make([]*config.Script, len(names))
	for i, n := range names {
		scripts[i] = &config.Script{Name: n, Category: cat, CheckFn: checkInstalled(false), InstallFn: noop}
	}
	return Section{Category: cat, Scripts: scripts}
}

func TestCursorMoveDownWithinSection(t *testing.T) {
	m := modelWithSections([]Section{
		makeSection(config.ScriptCategoryTerminal, "a", "b", "c"),
	})
	m.moveCursorDown()
	if m.cursor.item != 1 {
		t.Errorf("cursor.item = %d, want 1", m.cursor.item)
	}
	m.moveCursorDown()
	if m.cursor.item != 2 {
		t.Errorf("cursor.item = %d, want 2", m.cursor.item)
	}
	m.moveCursorDown()
	if m.cursor.item != 2 {
		t.Errorf("cursor.item = %d after bottom, want 2 (no-op at end)", m.cursor.item)
	}
}

func TestCursorMoveDownAcrossSections(t *testing.T) {
	m := modelWithSections([]Section{
		makeSection(config.ScriptCategoryTerminal, "a"),
		makeSection(config.ScriptCategorySecurity, "b", "c"),
	})
	m.moveCursorDown()
	if m.cursor.section != 1 || m.cursor.item != 0 {
		t.Errorf("cursor = (%d, %d), want (1, 0)", m.cursor.section, m.cursor.item)
	}
}

func TestCursorMoveDownSkipsCollapsedSection(t *testing.T) {
	m := modelWithSections([]Section{
		makeSection(config.ScriptCategoryTerminal, "a"),
		makeSection(config.ScriptCategorySecurity, "b", "c"),
		makeSection(config.ScriptCategoryEditor, "d"),
	})
	m.sections[1].Collapsed = true
	m.moveCursorDown()
	if m.cursor.section != 2 || m.cursor.item != 0 {
		t.Errorf("cursor = (%d, %d), want (2, 0) — should skip collapsed section",
			m.cursor.section, m.cursor.item)
	}
}

func TestCursorMoveUpAcrossSections(t *testing.T) {
	m := modelWithSections([]Section{
		makeSection(config.ScriptCategoryTerminal, "a", "b"),
		makeSection(config.ScriptCategorySecurity, "c"),
	})
	m.cursor = cursorPos{section: 1, item: 0}
	m.moveCursorUp()
	if m.cursor.section != 0 || m.cursor.item != 1 {
		t.Errorf("cursor = (%d, %d), want (0, 1)", m.cursor.section, m.cursor.item)
	}
}

func TestCursorMoveUpSkipsCollapsedSection(t *testing.T) {
	m := modelWithSections([]Section{
		makeSection(config.ScriptCategoryTerminal, "a"),
		makeSection(config.ScriptCategorySecurity, "b"),
		makeSection(config.ScriptCategoryEditor, "c"),
	})
	m.sections[1].Collapsed = true
	m.cursor = cursorPos{section: 2, item: 0}
	m.moveCursorUp()
	if m.cursor.section != 0 || m.cursor.item != 0 {
		t.Errorf("cursor = (%d, %d), want (0, 0) — should skip collapsed section",
			m.cursor.section, m.cursor.item)
	}
}

func TestToggleCurrentSection(t *testing.T) {
	m := modelWithSections([]Section{
		makeSection(config.ScriptCategoryTerminal, "a"),
	})
	if m.sections[0].Collapsed {
		t.Fatal("section starts collapsed, want expanded")
	}
	m.toggleCurrentSection()
	if !m.sections[0].Collapsed {
		t.Error("toggle didn't collapse")
	}
	m.toggleCurrentSection()
	if m.sections[0].Collapsed {
		t.Error("second toggle didn't expand")
	}
}

func TestStartInstallNoOpWhenAlreadyInstalled(t *testing.T) {
	called := false
	scripts := []*config.Script{{
		Name:      "x",
		Category:  config.ScriptCategoryTerminal,
		CheckFn:   checkInstalled(true),
		InstallFn: func(_ config.InputValues) error { called = true; return nil },
	}}
	m := modelWithSections([]Section{{Category: config.ScriptCategoryTerminal, Scripts: scripts}})
	_, cmd := m.startInstall()
	if cmd != nil {
		t.Error("expected nil cmd when already installed")
	}
	if called {
		t.Error("InstallFn should not be invoked")
	}
}

func TestStartInstallNoOpWithoutInstallFn(t *testing.T) {
	scripts := []*config.Script{{
		Name:     "x",
		Category: config.ScriptCategoryTerminal,
		CheckFn:  checkInstalled(false),
	}}
	m := modelWithSections([]Section{{Category: config.ScriptCategoryTerminal, Scripts: scripts}})
	_, cmd := m.startInstall()
	if cmd != nil {
		t.Error("expected nil cmd when InstallFn missing")
	}
}

func TestStartInstallOpensModalWhenInputsDeclared(t *testing.T) {
	scripts := []*config.Script{{
		Name:      "needs-password",
		Category:  config.ScriptCategoryServer,
		CheckFn:   checkInstalled(false),
		InstallFn: noop,
		Inputs: []config.ScriptInput{
			{Name: "password", Label: "Password", Kind: config.InputPassword},
		},
	}}
	m := modelWithSections([]Section{{Category: config.ScriptCategoryServer, Scripts: scripts}})
	updated, cmd := m.startInstall()
	if cmd == nil {
		t.Fatal("expected modal init cmd")
	}
	mm := updated.(Model)
	if !mm.modalActive() {
		t.Error("modal should be active after startInstall on a script with Inputs")
	}
	if mm.formScript == nil || mm.formScript.Name != "needs-password" {
		t.Error("formScript not set correctly")
	}
}

func TestStartUninstallNoOpWhenNotInstalled(t *testing.T) {
	called := false
	scripts := []*config.Script{{
		Name:        "x",
		Category:    config.ScriptCategoryTerminal,
		CheckFn:     checkInstalled(false),
		InstallFn:   noop,
		UninstallFn: func() error { called = true; return nil },
	}}
	m := modelWithSections([]Section{{Category: config.ScriptCategoryTerminal, Scripts: scripts}})
	_, cmd := m.startUninstall()
	if cmd != nil {
		t.Error("expected nil cmd when not installed")
	}
	if called {
		t.Error("UninstallFn should not be invoked")
	}
}

func TestStartUninstallNoOpWithoutUninstallFn(t *testing.T) {
	scripts := []*config.Script{{
		Name:      "x",
		Category:  config.ScriptCategoryTerminal,
		CheckFn:   checkInstalled(true),
		InstallFn: noop,
		// No UninstallFn — item is install-only.
	}}
	m := modelWithSections([]Section{{Category: config.ScriptCategoryTerminal, Scripts: scripts}})
	_, cmd := m.startUninstall()
	if cmd != nil {
		t.Error("expected nil cmd when UninstallFn missing")
	}
}

func TestCollectModalValuesEmpty(t *testing.T) {
	m := Model{}
	got := m.collectModalValues()
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestCollectModalValues(t *testing.T) {
	pw := "secret123"
	user := "alice"
	m := Model{
		formScript: &config.Script{Inputs: []config.ScriptInput{
			{Name: "password"}, {Name: "user"},
		}},
		formBindings: []*string{&pw, &user},
	}
	got := m.collectModalValues()
	if got.Get("password") != "secret123" || got.Get("user") != "alice" {
		t.Errorf("unexpected values: %v", got)
	}
}

func TestRebuildSectionsClampsCursor(t *testing.T) {
	// Cursor starts at (0, 5) but section only has 3 scripts after rebuild.
	m := Model{
		sections: []Section{makeSection(config.ScriptCategoryTerminal, "a", "b", "c")},
		cursor:   cursorPos{section: 0, item: 5},
		expanded: map[string]bool{},
	}
	m.clampCursor()
	if m.cursor.item != 2 {
		t.Errorf("cursor.item = %d, want 2 (clamped to last)", m.cursor.item)
	}
}

func TestRebuildSectionsHandlesEmpty(t *testing.T) {
	m := Model{sections: nil, expanded: map[string]bool{}}
	m.clampCursor()
	if m.cursor.section != -1 || m.cursor.item != -1 {
		t.Errorf("cursor = (%d, %d), want (-1, -1)", m.cursor.section, m.cursor.item)
	}
}
