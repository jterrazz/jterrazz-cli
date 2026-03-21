package config

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Cleanable represents reclaimable storage that can be cleaned
type Cleanable struct {
	Name        string
	Description string

	// Location
	Path   string        // Static path
	PathFn func() string // Dynamic path (overrides Path)

	// Info
	SizeFn func() int64 // Get current size in bytes

	// Clean
	CleanFn func() error // How to clean it

	// Dependencies
	RequiresCmd string // Only show if this command exists (e.g., "docker")
}

// Cleanables is the list of reclaimable storage (caches, VMs, trash, etc.)
var Cleanables = []Cleanable{
	{
		Name:        "brew",
		Description: "Clean Homebrew cache",
		RequiresCmd: "brew",
		CleanFn: func() error {
			cmd := exec.Command("brew", "cleanup")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		SizeFn: func() int64 {
			return GetDirSize(os.Getenv("HOME") + "/Library/Caches/Homebrew")
		},
	},
	{
		Name:        "docker",
		Description: "Clean Docker containers, images, volumes",
		RequiresCmd: "docker",
		CleanFn: func() error {
			commands := [][]string{
				{"docker", "container", "prune", "-f"},
				{"docker", "image", "prune", "-f"},
				{"docker", "volume", "prune", "-f"},
				{"docker", "network", "prune", "-f"},
				{"docker", "builder", "prune", "-f"},
			}
			var lastErr error
			for _, args := range commands {
				cmd := exec.Command(args[0], args[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					lastErr = err
				}
			}
			return lastErr
		},
	},
	{
		Name:        "multipass",
		Description: "Remove all Multipass instances",
		RequiresCmd: "multipass",
		CleanFn: func() error {
			_ = exec.Command("multipass", "delete", "--all").Run() // ok if no instances
			cmd := exec.Command("multipass", "purge")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		SizeFn: func() int64 {
			return GetDirSize(os.Getenv("HOME") + "/Library/Application Support/multipassd")
		},
	},
	{
		Name:        "trash",
		Description: "Empty system trash",
		CleanFn: func() error {
			trashPath := os.Getenv("HOME") + "/.Trash"
			os.RemoveAll(trashPath)
			return os.MkdirAll(trashPath, 0755)
		},
		SizeFn: func() int64 {
			return GetDirSize(os.Getenv("HOME") + "/.Trash")
		},
	},
}

// =============================================================================
// Cleanable Functions
// =============================================================================

// GetAllCleanables returns all cleanables
func GetAllCleanables() []Cleanable {
	return Cleanables
}

// GetCleanableByName returns a cleanable by name
func GetCleanableByName(name string) *Cleanable {
	for i := range Cleanables {
		if Cleanables[i].Name == name {
			return &Cleanables[i]
		}
	}
	return nil
}

// GetAvailableCleanables returns cleanables where the required command exists
func GetAvailableCleanables() []Cleanable {
	var result []Cleanable
	for _, c := range Cleanables {
		if c.RequiresCmd == "" || CommandExists(c.RequiresCmd) {
			result = append(result, c)
		}
	}
	return result
}

// GetDirSize calculates the total size of a directory
func GetDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
