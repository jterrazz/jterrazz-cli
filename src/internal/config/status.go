package config

// StatusSection represents a section in the status display
type StatusSection struct {
	Title    string
	SubTitle string // Optional subsection title
	RenderFn func() // Function to render this section
}

// StatusSections defines all status sections in display order
var StatusSections = []StatusSection{
	// Activity — live performance
	{Title: "Activity", SubTitle: "CPU", RenderFn: nil},
	{Title: "Activity", SubTitle: "Memory", RenderFn: nil},

	// Environment — network, services, system health
	{Title: "Environment", SubTitle: "Network", RenderFn: nil},
	{Title: "Environment", SubTitle: "Services", RenderFn: nil},
	{Title: "Environment", SubTitle: "System", RenderFn: nil},

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
	{Title: "Tools", SubTitle: "DevOps", RenderFn: nil},
	{Title: "Tools", SubTitle: "AI", RenderFn: nil},
	{Title: "Tools", SubTitle: "Terminal & Git", RenderFn: nil},
	{Title: "Tools", SubTitle: "GUI Apps", RenderFn: nil},
	{Title: "Tools", SubTitle: "Mac App Store", RenderFn: nil},
}

// ToolCategories defines the order of tool categories in status display
var ToolCategories = []ToolCategory{
	CategoryPackageManager,
	CategoryRuntimes,
	CategoryDevOps,
	CategoryAI,
	CategoryTerminalGit,
	CategoryGUIApps,
	CategoryMacAppStore,
}
