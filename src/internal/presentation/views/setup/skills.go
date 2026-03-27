package setup

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jterrazz/jterrazz-cli/src/internal/config"
	"github.com/jterrazz/jterrazz-cli/src/internal/domain/skill"
	"github.com/jterrazz/jterrazz-cli/src/internal/presentation/components"
)

// skillAction represents action items in skills UI
type skillAction string

const (
	skillActionInstallRepo skillAction = "install-repo"
)

type skillsState struct {
	expanded     map[string]bool
	installed    []string
	repoSkills   map[string][]string
	loadingRepo  string
	loadingSkill string // Skill currently being installed/removed
	itemData     []skillItemData
}

type skillItemData struct {
	repo   string
	skill  string
	action skillAction
}

var skills skillsState

// InitSkillsState initializes the skills state
func InitSkillsState() {
	skills = skillsState{
		expanded:   make(map[string]bool),
		installed:  skill.ListInstalled(),
		repoSkills: make(map[string][]string),
		itemData:   nil,
	}
}

// BuildSkillsItems builds the skills menu items
func BuildSkillsItems() []components.Item {
	var items []components.Item
	skills.itemData = nil

	// Studio skills section (@jterrazz)
	studioSkills := config.GetStudioSkills()
	if len(studioSkills) > 0 {
		items = append(items, components.Item{Kind: components.KindHeader, Label: "Studio"})
		skills.itemData = append(skills.itemData, skillItemData{})

		for _, s := range studioSkills {
			state := components.StateUnchecked
			if skills.loadingSkill == s.Skill {
				state = components.StateLoading
			} else if isSkillInstalled(s.Skill) {
				state = components.StateChecked
			}
			items = append(items, components.Item{
				Kind:  components.KindToggle,
				Label: s.Skill,
				State: state,
			})
			skills.itemData = append(skills.itemData, skillItemData{repo: s.Repo, skill: s.Skill})
		}
	}

	// Community skills section
	communitySkills := config.GetCommunitySkills()
	if len(communitySkills) > 0 {
		items = append(items, components.Item{Kind: components.KindHeader, Label: "Community"})
		skills.itemData = append(skills.itemData, skillItemData{})

		for _, s := range communitySkills {
			state := components.StateUnchecked
			if skills.loadingSkill == s.Skill {
				state = components.StateLoading
			} else if isSkillInstalled(s.Skill) {
				state = components.StateChecked
			}
			items = append(items, components.Item{
				Kind:  components.KindToggle,
				Label: s.Skill,
				State: state,
			})
			skills.itemData = append(skills.itemData, skillItemData{repo: s.Repo, skill: s.Skill})
		}
	}

	// Installed section (skills not in Favorites)
	var otherInstalled []string
	for _, s := range skills.installed {
		if !config.IsFavoriteSkill("", s) {
			otherInstalled = append(otherInstalled, s)
		}
	}
	if len(otherInstalled) > 0 {
		items = append(items, components.Item{Kind: components.KindHeader, Label: "Installed"})
		skills.itemData = append(skills.itemData, skillItemData{})

		for _, s := range otherInstalled {
			repo := findRepoForSkill(s)
			state := components.StateChecked
			if skills.loadingSkill == s {
				state = components.StateLoading
			}
			items = append(items, components.Item{
				Kind:        components.KindToggle,
				Label:       s,
				Description: repo,
				State:       state,
			})
			skills.itemData = append(skills.itemData, skillItemData{repo: repo, skill: s})
		}
	}

	// Browse section
	items = append(items, components.Item{Kind: components.KindHeader, Label: "Repositories"})
	skills.itemData = append(skills.itemData, skillItemData{})

	for _, repo := range config.GetAllSkillRepos() {
		expanded := skills.expanded[repo.Name]
		repoSkills := skills.repoSkills[repo.Name]
		isLoading := skills.loadingRepo == repo.Name

		desc := repo.Description
		if isLoading {
			desc = "Loading..."
		} else if len(repoSkills) > 0 {
			installedCount := 0
			for _, s := range repoSkills {
				if isSkillInstalled(s) {
					installedCount++
				}
			}
			desc = strconv.Itoa(len(repoSkills)) + " skills"
			if installedCount > 0 {
				desc = strconv.Itoa(installedCount) + "/" + strconv.Itoa(len(repoSkills)) + " installed"
			}
		}

		items = append(items, components.Item{
			Kind:        components.KindExpandable,
			Label:       repo.Name,
			Description: desc,
			Expanded:    expanded,
		})
		skills.itemData = append(skills.itemData, skillItemData{repo: repo.Name})

		if expanded && repoSkills != nil {
			items = append(items, components.Item{
				Kind:   components.KindAction,
				Label:  "install all",
				Indent: 1,
			})
			skills.itemData = append(skills.itemData, skillItemData{repo: repo.Name, action: skillActionInstallRepo})

			for _, s := range repoSkills {
				state := components.StateUnchecked
				if skills.loadingSkill == s {
					state = components.StateLoading
				} else if isSkillInstalled(s) {
					state = components.StateChecked
				}
				items = append(items, components.Item{
					Kind:   components.KindToggle,
					Label:  s,
					State:  state,
					Indent: 1,
				})
				skills.itemData = append(skills.itemData, skillItemData{repo: repo.Name, skill: s})
			}
		}
	}

	return items
}

func isSkillInstalled(name string) bool {
	for _, s := range skills.installed {
		if s == name {
			return true
		}
	}
	return false
}

func findRepoForSkill(name string) string {
	for repoName, repoSkills := range skills.repoSkills {
		for _, s := range repoSkills {
			if s == name {
				return repoName
			}
		}
	}
	for _, s := range config.FavoriteSkills {
		if s.Skill == name {
			return s.Repo
		}
	}
	return ""
}

// HandleSkillsSelect handles item selection in the skills menu
func HandleSkillsSelect(index int, item components.Item) tea.Cmd {
	if index >= len(skills.itemData) {
		return nil
	}
	data := skills.itemData[index]

	switch item.Kind {
	case components.KindExpandable:
		if skills.expanded[data.repo] {
			skills.expanded[data.repo] = false
			return func() tea.Msg { return components.RefreshMsg{} }
		} else {
			skills.expanded[data.repo] = true
			if skills.repoSkills[data.repo] == nil {
				skills.loadingRepo = data.repo
				return fetchSkillsCmd(data.repo)
			}
			return func() tea.Msg { return components.RefreshMsg{} }
		}

	case components.KindAction:
		if data.action == skillActionInstallRepo {
			return installRepoCmd(data.repo)
		}

	case components.KindToggle:
		skills.loadingSkill = data.skill
		if item.State == components.StateChecked {
			return removeSkillCmd(data.skill)
		}
		return installSkillCmd(data.repo, data.skill)
	}

	return nil
}

// SkillsFetchedMsg is sent when skills are fetched from a repo
type SkillsFetchedMsg struct {
	Repo   string
	Skills []string
	Err    error
}

// HandleSkillsMessage handles messages for the skills view
func HandleSkillsMessage(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SkillsFetchedMsg:
		skills.loadingRepo = ""
		if msg.Err == nil {
			skills.repoSkills[msg.Repo] = msg.Skills
		} else {
			skills.repoSkills[msg.Repo] = []string{}
		}
		return nil

	case components.ActionDoneMsg:
		skills.loadingSkill = ""
		skills.installed = skill.ListInstalled()
		return nil
	}
	return nil
}

func fetchSkillsCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		repoSkills, err := skill.ListFromRepo(repo)
		return SkillsFetchedMsg{Repo: repo, Skills: repoSkills, Err: err}
	}
}

func installSkillCmd(repo, name string) tea.Cmd {
	return func() tea.Msg {
		if err := skill.Install(repo, name); err != nil {
			return components.ActionDoneMsg{Message: "Error: " + err.Error(), Err: err}
		}
		return components.ActionDoneMsg{Message: "Installed " + name}
	}
}

func installRepoCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		if err := skill.InstallAll(repo); err != nil {
			return components.ActionDoneMsg{Message: "Error: " + err.Error(), Err: err}
		}
		return components.ActionDoneMsg{Message: "Installed all from " + repo}
	}
}

func removeSkillCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := skill.Remove(name); err != nil {
			return components.ActionDoneMsg{Message: "Error: " + err.Error(), Err: err}
		}
		return components.ActionDoneMsg{Message: "Removed " + name}
	}
}

// SkillsConfig returns the TUI config for the skills view
func SkillsConfig() components.AppConfig {
	return components.AppConfig{
		Title:      "Skills",
		BuildItems: BuildSkillsItems,
		OnSelect:   HandleSkillsSelect,
		OnMessage:  HandleSkillsMessage,
	}
}
