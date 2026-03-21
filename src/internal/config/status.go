package config

// StatusSection represents a section in the status display
type StatusSection struct {
	Title    string
	SubTitle string // Optional subsection title
	RenderFn func() // Function to render this section
}

// StatusSections defines all status sections in display order
var StatusSections = []StatusSection{
	// System — live performance
	{Title: "System", SubTitle: "CPU", RenderFn: nil},
	{Title: "System", SubTitle: "Memory", RenderFn: nil},

	// Environment — network, services, system health
	{Title: "Environment", SubTitle: "Network", RenderFn: nil},
	{Title: "Environment", SubTitle: "Services", RenderFn: nil},
	{Title: "Environment", SubTitle: "Health", RenderFn: nil},

	// Workspace — dev state
	{Title: "Workspace", SubTitle: "Git", RenderFn: nil},
	{Title: "Workspace", SubTitle: "Disk", RenderFn: nil},

	// Setup — config checks (flat)
	{Title: "Setup", SubTitle: "Setup", RenderFn: nil},
	{Title: "Setup", SubTitle: "Security", RenderFn: nil},
	{Title: "Setup", SubTitle: "Identity", RenderFn: nil},

	// Tools — inventory (cards)
	{Title: "Tools", SubTitle: "Package Managers", RenderFn: nil},
	{Title: "Tools", SubTitle: "Runtimes", RenderFn: nil},
	{Title: "Tools", SubTitle: "Terminal", RenderFn: nil},
	{Title: "Tools", SubTitle: "Git", RenderFn: nil},
	{Title: "Tools", SubTitle: "System", RenderFn: nil},
	{Title: "Tools", SubTitle: "Deploy", RenderFn: nil},
	{Title: "Tools", SubTitle: "AI Agents", RenderFn: nil},
	{Title: "Tools", SubTitle: "AI Tooling", RenderFn: nil},
	{Title: "Tools", SubTitle: "Development", RenderFn: nil},
	{Title: "Tools", SubTitle: "Creative", RenderFn: nil},
	{Title: "Tools", SubTitle: "Communication", RenderFn: nil},
	{Title: "Tools", SubTitle: "Productivity", RenderFn: nil},
	{Title: "Tools", SubTitle: "AI Apps", RenderFn: nil},
	{Title: "Tools", SubTitle: "Browse", RenderFn: nil},
	{Title: "Tools", SubTitle: "Security", RenderFn: nil},
	{Title: "Tools", SubTitle: "Entertainment", RenderFn: nil},
	{Title: "Tools", SubTitle: "Utilities", RenderFn: nil},
}

// ToolCategories defines the order of tool categories in status display
var ToolCategories = []ToolCategory{
	CategoryPackageManager,
	CategoryRuntimes,
	CategoryTerminal,
	CategoryGit,
	CategorySystem,
	CategoryDeploy,
	CategoryAIAgents,
	CategoryAITooling,
	CategoryDevelopment,
	CategoryCreative,
	CategoryCommunication,
	CategoryProductivity,
	CategoryAIApps,
	CategoryBrowse,
	CategorySecurity,
	CategoryEntertainment,
	CategoryUtilities,
}
