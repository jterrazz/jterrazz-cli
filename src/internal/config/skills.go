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

// StudioSkills are @jterrazz skills — the foundation of every project
var StudioSkills = []Skill{
	{"jterrazz/jterrazz-studio", "jterrazz-stack"},
	{"jterrazz/jterrazz-studio", "jterrazz-new-project"},
	{"jterrazz/jterrazz-infra", "jterrazz-infra"},
	{"jterrazz/package-typescript", "jterrazz-typescript"},
	{"jterrazz/package-codestyle", "jterrazz-codestyle"},
	{"jterrazz/package-broadcast", "jterrazz-broadcast"},
	{"jterrazz/jterrazz-workflows", "jterrazz-workflows"},
}

// CommunitySkills are third-party skills worth having
var CommunitySkills = []Skill{
	{"anthropics/skills", "frontend-design"},
	{"expo/skills", "upgrading-expo"},
	{"shadcn-ui/ui", "shadcn"},
	{"sickn33/antigravity-awesome-skills", "last30days"},
	{"tobi/qmd", "qmd"},
	{"vercel-labs/agent-skills", "vercel-react-best-practices"},
	{"vercel-labs/agent-skills", "vercel-react-native-skills"},
}

// FavoriteSkills is all pinned skills combined (studio + community)
var FavoriteSkills = append(StudioSkills, CommunitySkills...)

// StudioRepos are @jterrazz skill repositories
var StudioRepos = []SkillRepo{
	{"jterrazz/jterrazz-studio", "Dev toolkit, conventions, and scaffolding"},
	{"jterrazz/jterrazz-infra", "Infrastructure and deployment (K3s, Helm, Traefik)"},
	{"jterrazz/jterrazz-workflows", "Shared CI/CD workflows (validate, release)"},
	{"jterrazz/package-typescript", "TypeScript build tooling (tsdown)"},
	{"jterrazz/package-codestyle", "Linting and formatting (oxlint, oxfmt, tsgo)"},
	{"jterrazz/package-broadcast", "Multi-channel announcements (App Store, push)"},
}

// CommunityRepos are third-party skill repositories
var CommunityRepos = []SkillRepo{
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

// SkillRepos is all repositories combined (studio + community)
var SkillRepos = append(StudioRepos, CommunityRepos...)

// GetAllSkillRepos returns all skill repositories
func GetAllSkillRepos() []SkillRepo {
	return SkillRepos
}

// GetStudioSkills returns @jterrazz skills
func GetStudioSkills() []Skill {
	return StudioSkills
}

// GetCommunitySkills returns third-party skills
func GetCommunitySkills() []Skill {
	return CommunitySkills
}

// GetStudioRepos returns @jterrazz skill repositories
func GetStudioRepos() []SkillRepo {
	return StudioRepos
}

// GetCommunityRepos returns third-party skill repositories
func GetCommunityRepos() []SkillRepo {
	return CommunityRepos
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
