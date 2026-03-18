package config

import (
	"fmt"

	output "github.com/jterrazz/jterrazz-cli/src/internal/presentation/print"
)

// PackageManager represents an upgradable package manager
type PackageManager struct {
	Name        string
	Flag        string // CLI flag name (e.g., "brew" for --brew)
	RequiresCmd string // Command that must exist
	UpgradeFn   func() // Function to run upgrades
}

// PackageManagers is the list of all package managers that can be upgraded
var PackageManagers = []PackageManager{
	{
		Name:        "homebrew",
		Flag:        "brew",
		RequiresCmd: "brew",
		UpgradeFn:   upgradeBrew,
	},
	{
		Name:        "npm",
		Flag:        "npm",
		RequiresCmd: "npm",
		UpgradeFn:   upgradeNpm,
	},
	{
		Name:        "bun",
		Flag:        "bun",
		RequiresCmd: "bun",
		UpgradeFn:   upgradeBun,
	},
}

// GetPackageManagerByFlag returns a package manager by its flag name
func GetPackageManagerByFlag(flag string) *PackageManager {
	for i := range PackageManagers {
		if PackageManagers[i].Flag == flag {
			return &PackageManagers[i]
		}
	}
	return nil
}

// UpgradeAll upgrades all available package managers
func UpgradeAll() {
	for _, pm := range PackageManagers {
		if CommandExists(pm.RequiresCmd) {
			pm.UpgradeFn()
		}
	}
}

// UpgradePackageManager upgrades a specific package manager
func UpgradePackageManager(pm PackageManager) {
	if !CommandExists(pm.RequiresCmd) {
		fmt.Printf("%s %s not found, skipping\n", output.Cyan("Warning:"), pm.RequiresCmd)
		return
	}
	pm.UpgradeFn()
}

// UpgradePackageByName upgrades a specific package by name
func UpgradePackageByName(name string) error {
	// Find package in our tools list
	pkg := GetToolByName(name)
	if pkg != nil {
		switch pkg.Method {
		case InstallBrewFormula:
			if !CommandExists("brew") {
				return fmt.Errorf("Homebrew not found")
			}
			fmt.Printf("  📥 Upgrading %s...\n", name)
			ExecCommand("brew", "upgrade", pkg.Formula)
			fmt.Printf("  %s %s upgraded\n", output.Green("✓"), name)
			return nil
		case InstallBrewCask:
			if !CommandExists("brew") {
				return fmt.Errorf("Homebrew not found")
			}
			fmt.Printf("  📥 Upgrading %s...\n", name)
			ExecCommand("brew", "upgrade", "--cask", pkg.Formula)
			fmt.Printf("  %s %s upgraded\n", output.Green("✓"), name)
			return nil
		case InstallNpm:
			if !CommandExists("npm") {
				return fmt.Errorf("npm not found")
			}
			fmt.Printf("  📥 Upgrading %s...\n", name)
			ExecCommand("npm", "update", "-g", pkg.Formula)
			fmt.Printf("  %s %s upgraded\n", output.Green("✓"), name)
			return nil
		case InstallBun:
			if !CommandExists("bun") {
				return fmt.Errorf("bun not found")
			}
			fmt.Printf("  📥 Upgrading %s...\n", name)
			ExecCommand("bun", "update", "-g", pkg.Formula)
			fmt.Printf("  %s %s upgraded\n", output.Green("✓"), name)
			return nil
		}
	}

	// Try as a direct brew package name
	if CommandExists("brew") {
		fmt.Printf("  📥 Upgrading %s...\n", name)
		ExecCommand("brew", "upgrade", name)
		fmt.Printf("  %s %s upgraded\n", output.Green("✓"), name)
		return nil
	}

	return fmt.Errorf("unknown package: %s", name)
}

// =============================================================================
// Upgrade Functions
// =============================================================================

func upgradeBrew() {
	fmt.Println(output.Cyan("🍺 Upgrading Homebrew packages..."))
	ExecCommand("brew", "update")
	ExecCommand("brew", "upgrade")
	fmt.Println(output.Green("  ✅ Homebrew upgrade completed"))
}

func upgradeNpm() {
	fmt.Println(output.Cyan("📦 Upgrading npm global packages..."))
	ExecCommand("npm", "update", "-g")
	fmt.Println(output.Green("  ✅ npm upgrade completed"))
}

func upgradeBun() {
	fmt.Println(output.Cyan("📦 Upgrading bun global packages..."))
	ExecCommand("bun", "update", "-g")
	fmt.Println(output.Green("  ✅ bun upgrade completed"))
}
