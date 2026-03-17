package config

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunCommand represents a group of related subcommands under `j run`
type RunCommand struct {
	Name        string
	Description string
	Subcommands []RunSubcommand
}

// RunSubcommand represents a single executable subcommand
type RunSubcommand struct {
	Name        string
	Description string
	MinArgs     int
	RunFn       func(args []string) error
}

// RunCommands is the list of all `j run` command groups
var RunCommands = []RunCommand{
	// =========================================================================
	// Docker
	// =========================================================================
	{
		Name:        "docker",
		Description: "Docker container and image management",
		Subcommands: []RunSubcommand{
			{
				Name:        "rm",
				Description: "Remove all containers",
				RunFn: func(args []string) error {
					out, err := exec.Command("docker", "ps", "-aq").Output()
					if err != nil || strings.TrimSpace(string(out)) == "" {
						fmt.Println("No containers to remove")
						return nil
					}
					containers := strings.Fields(string(out))
					return ExecCommand("docker", append([]string{"rm", "-vf"}, containers...)...)
				},
			},
			{
				Name:        "rmi",
				Description: "Remove all images",
				RunFn: func(args []string) error {
					out, err := exec.Command("docker", "images", "-aq").Output()
					if err != nil || strings.TrimSpace(string(out)) == "" {
						fmt.Println("No images to remove")
						return nil
					}
					images := strings.Fields(string(out))
					return ExecCommand("docker", append([]string{"rmi", "-f"}, images...)...)
				},
			},
			{
				Name:        "clean",
				Description: "Clean up Docker system (prune)",
				RunFn: func(args []string) error {
					return ExecCommand("docker", "system", "prune", "-af")
				},
			},
			{
				Name:        "reset",
				Description: "Remove all containers and images",
				RunFn: func(args []string) error {
					// Remove containers (ignore errors, continue cleaning)
					out, _ := exec.Command("docker", "ps", "-aq").Output()
					if strings.TrimSpace(string(out)) != "" {
						containers := strings.Fields(string(out))
						_ = ExecCommand("docker", append([]string{"rm", "-vf"}, containers...)...)
					}
					// Remove images (ignore errors, continue cleaning)
					out, _ = exec.Command("docker", "images", "-aq").Output()
					if strings.TrimSpace(string(out)) != "" {
						images := strings.Fields(string(out))
						_ = ExecCommand("docker", append([]string{"rmi", "-f"}, images...)...)
					}
					return nil
				},
			},
		},
	},
	// =========================================================================
	// Git
	// =========================================================================
	{
		Name:        "git",
		Description: "Git workflow shortcuts",
		Subcommands: []RunSubcommand{
			{
				Name:        "feat",
				Description: "Add all and commit with 'feat:' prefix",
				MinArgs:     1,
				RunFn:       func(args []string) error { return gitCommit("feat", args) },
			},
			{
				Name:        "fix",
				Description: "Add all and commit with 'fix:' prefix",
				MinArgs:     1,
				RunFn:       func(args []string) error { return gitCommit("fix", args) },
			},
			{
				Name:        "chore",
				Description: "Add all and commit with 'chore:' prefix",
				MinArgs:     1,
				RunFn:       func(args []string) error { return gitCommit("chore", args) },
			},
			{
				Name:        "push",
				Description: "Push current branch to origin",
				RunFn: func(args []string) error {
					return ExecCommand("git", "push", "-u", "origin", "HEAD")
				},
			},
			{
				Name:        "sync",
				Description: "Fetch and pull from remote",
				RunFn: func(args []string) error {
					// fetch -p can fail if no remote configured, continue anyway
					_ = ExecCommand("git", "fetch", "-p")
					return ExecCommand("git", "pull")
				},
			},
			{
				Name:        "wip",
				Description: "Add all and commit as 'WIP'",
				RunFn: func(args []string) error {
					if err := ExecCommand("git", "add", "--all"); err != nil {
						return fmt.Errorf("git add failed: %w", err)
					}
					return ExecCommand("git", "commit", "-m", "WIP")
				},
			},
			{
				Name:        "unwip",
				Description: "Undo last commit and unstage",
				RunFn: func(args []string) error {
					if err := ExecCommand("git", "reset", "--soft", "HEAD~1"); err != nil {
						return fmt.Errorf("git reset failed: %w", err)
					}
					return ExecCommand("git", "reset", "HEAD")
				},
			},
			{
				Name:        "status",
				Description: "Show git status",
				RunFn: func(args []string) error {
					return ExecCommand("git", "status")
				},
			},
			{
				Name:        "log",
				Description: "Show recent commits",
				RunFn: func(args []string) error {
					return ExecCommand("git", "log", "--oneline", "-10")
				},
			},
			{
				Name:        "branches",
				Description: "List local branches",
				RunFn: func(args []string) error {
					return ExecCommand("git", "branch")
				},
			},
		},
	},
}

// gitCommit stages all changes and commits with a prefixed message
func gitCommit(prefix string, args []string) error {
	if err := ExecCommand("git", "add", "."); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	message := fmt.Sprintf("%s: %s", prefix, strings.Join(args, " "))
	return ExecCommand("git", "commit", "-m", message)
}
