package config

// SkillRepo represents a repository containing AI agent skills
type SkillRepo struct {
	Name        string // Repository name (owner/repo format)
	Description string
}

// Skill represents a favorite/installed skill
type Skill struct {
	Repo  string // Repository (owner/repo format)
	Skill string // Skill name within the repo
}

// FavoriteSkills is the list of skills you want installed on your machine
// The "Install All" action will install these skills
var FavoriteSkills = []Skill{
	{"anthropics/skills", "frontend-design"},
	{"expo/skills", "upgrading-expo"},
	{"shadcn-ui/ui", "shadcn"},
	{"sickn33/antigravity-awesome-skills", "last30days"},
	{"tobi/qmd", "qmd"},
	{"vercel-labs/agent-skills", "vercel-react-best-practices"},
	{"vercel-labs/agent-skills", "vercel-react-native-skills"},
}

// SkillRepos is the list of recommended skill repositories
// Skills are fetched dynamically when a repo is expanded in the TUI
var SkillRepos = []SkillRepo{
	{"anthropics/skills", "Official Anthropic skills for Claude"},
	{"better-auth/skills", "Authentication best practices"},
	{"code-with-beto/skills", "Beto's development skills"},
	{"coreyhaines31/marketingskills", "Marketing and SEO skills"},
	{"expo/skills", "Expo and React Native mobile development"},
	{"firecrawl/cli", "Web content extraction for AI agents"},
	{"shadcn-ui/ui", "Official shadcn/ui components and patterns"},
	{"obra/superpowers", "Development workflow and productivity skills"},
	{"remotion-dev/skills", "Remotion video creation skills"},
	{"resend/email-best-practices", "Email development best practices"},
	{"supabase/agent-skills", "Supabase database and backend skills"},
	{"tobi/qmd", "Local search engine for docs and knowledge bases"},
	{"vercel-labs/agent-skills", "Vercel React and web development skills"},
}

// GetAllSkillRepos returns all skill repositories
func GetAllSkillRepos() []SkillRepo {
	return SkillRepos
}

// GetSkillRepoByName returns a skill repo by name
func GetSkillRepoByName(name string) *SkillRepo {
	for i := range SkillRepos {
		if SkillRepos[i].Name == name {
			return &SkillRepos[i]
		}
	}
	return nil
}

// GetFavoriteSkills returns all favorite skills
func GetFavoriteSkills() []Skill {
	return FavoriteSkills
}

// IsFavoriteSkill checks if a skill is in the favorites list
// If repo is empty, only the skill name is checked
func IsFavoriteSkill(repo, skill string) bool {
	for _, fav := range FavoriteSkills {
		if fav.Skill == skill {
			if repo == "" || fav.Repo == repo {
				return true
			}
		}
	}
	return false
}
