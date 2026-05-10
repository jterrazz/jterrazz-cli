package configview

import (
	"sort"

	"github.com/jterrazz/jterrazz-cli/src/internal/config"
)

// Section is a category of scripts grouped under one collapsible header in
// the j config TUI. The order of sections in a Model.sections slice is the
// vertical order they're rendered in.
type Section struct {
	Category  config.ScriptCategory
	Scripts   []*config.Script
	Collapsed bool
}

// sectionOrder defines how the standard categories stack. Categories not in
// this list (future ones) get appended after, in alphabetical order.
var sectionOrder = []config.ScriptCategory{
	config.ScriptCategoryTerminal,
	config.ScriptCategorySecurity,
	config.ScriptCategoryEditor,
	config.ScriptCategorySystem,
	config.ScriptCategoryHomelab,
}

// buildSections groups Scripts by Category, applies role filtering, sorts
// items within each section by name, and returns sections in the canonical
// order (sectionOrder first, then any extras alphabetically).
//
// Scripts without a Category are dropped — every config item must declare one.
func buildSections(role config.Role) []Section {
	groups := map[config.ScriptCategory][]*config.Script{}
	for i := range config.Scripts {
		s := &config.Scripts[i]
		if !s.MatchesRole(role) {
			continue
		}
		if s.Category == "" {
			continue
		}
		groups[s.Category] = append(groups[s.Category], s)
	}

	sortByName := func(scripts []*config.Script) {
		sort.Slice(scripts, func(i, j int) bool {
			return scripts[i].Name < scripts[j].Name
		})
	}

	var sections []Section
	seen := map[config.ScriptCategory]bool{}
	for _, cat := range sectionOrder {
		if scripts, ok := groups[cat]; ok && len(scripts) > 0 {
			sortByName(scripts)
			sections = append(sections, Section{Category: cat, Scripts: scripts})
			seen[cat] = true
		}
	}

	// Append any extra categories not in sectionOrder, alphabetically.
	var extras []config.ScriptCategory
	for cat := range groups {
		if !seen[cat] {
			extras = append(extras, cat)
		}
	}
	sort.Slice(extras, func(i, j int) bool { return extras[i] < extras[j] })
	for _, cat := range extras {
		scripts := groups[cat]
		sortByName(scripts)
		sections = append(sections, Section{Category: cat, Scripts: scripts})
	}

	return sections
}

// installedCount returns how many of the section's scripts are currently
// installed according to their CheckFn. Scripts without a CheckFn are
// ignored (they have no checkable state).
func (s Section) installedCount() (installed, total int) {
	for _, sc := range s.Scripts {
		if sc.CheckFn == nil {
			continue
		}
		total++
		if sc.CheckFn().Installed {
			installed++
		}
	}
	return
}
