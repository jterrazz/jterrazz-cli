# j

A single CLI to bootstrap and manage a macOS development machine — tools, configs, templates, and remote access. No sudo required.

## Install

**Fresh machine** (no Go needed):

```sh
xcode-select --install
curl -fsSL https://raw.githubusercontent.com/jterrazz/jterrazz-cli/main/install.sh | sh
source ~/.zshrc
```

**From source** (requires Go 1.24+):

```sh
git clone https://github.com/jterrazz/jterrazz-cli.git ~/Developer/jterrazz-cli
cd ~/Developer/jterrazz-cli
make install
source ~/.zshrc
```

The binary lives at `~/.jterrazz/bin/j`. All user data goes under `~/.jterrazz/`.

## Commands

### `j status`

Full-screen TUI showing system state at a glance: setup scripts, security checks, developer identity, 100+ tracked tools with versions, top processes, network info, and disk cache sizes. Everything loads in parallel.

### `j install [tool...]`

```sh
j install                          # List all tracked tools with status
j install homebrew go node         # Install specific tools
j install claude codex ollama rtk  # AI tools
j install ghostty tmux zed         # Terminal + editor
```

100+ tools across 7 categories (package managers, runtimes, devops, AI, terminal, GUI apps, Mac App Store). Each tool knows its install method (brew, cask, npm, bun, manual), dependencies, version detection, and optional post-install scripts.

### `j upgrade [package...]`

```sh
j upgrade --all          # Upgrade all package managers (brew, npm, bun)
j upgrade --brew         # Upgrade Homebrew only
j upgrade node claude    # Upgrade specific packages
```

### `j clean [item...]`

```sh
j clean --all            # Clean everything (brew cache, docker, multipass, trash)
j clean docker trash     # Clean specific items
```

### `j setup`

Interactive TUI to run configuration scripts:

- **Terminal** — ghostty, tmux, hushlogin
- **Security** — GPG commit signing, SSH keygen, GitHub CLI auth, encrypted DNS (Quad9), Spotlight exclusion
- **Editor** — Zed config
- **System** — JAVA_HOME, dock reset/spacer
- **Remote** — Tailscale configuration
- **Skills** — AI skills management

### `j remote`

```sh
j remote setup    # Configure Tailscale in ~/.jterrazz/config.json
j remote up       # Connect (userspace mode, SSH enabled, keep-awake)
j remote down     # Disconnect and stop daemon
j remote status   # Show connection state
```

Supports `auto`/`userspace` mode and `oauth`/`authkey` authentication.

### `j sync`

Sync project scaffolding across repos using [Copier](https://github.com/copier-org/copier) templates in `dotfiles/blueprints/`.

```sh
j sync init       # Scaffold a project (auto-detects Go/TypeScript)
j sync            # Pull template updates into current project
j sync diff       # Preview changes
j sync --all      # Update all projects in ~/Developer
```

Templates generate: `.editorconfig`, `.gitignore`, `LICENSE`, CI workflows, Docker/deploy configs, and Claude Code skill files — conditional on language and project type.

### `j run`

```sh
j run git feat "message"    # git add . && commit "feat: message"
j run git fix "message"     # git add . && commit "fix: message"
j run git wip               # git add --all && commit "WIP"
j run git unwip             # Undo last commit
j run git push              # Push current branch
j run git sync              # Fetch + pull
j run docker reset          # Remove all containers + images
j run docker clean          # System prune
```

### Shell shortcuts

Sourced via `dotfiles/applications/zsh/zshrc.sh`:

| Command | Action |
|---------|--------|
| `jj` | Attach tmux session `main` |
| `jc` | Open Claude in tmux |
| `jo` | Open Codex in tmux |
| `jg` | Open Gemini in tmux |

## User data

Everything lives under `~/.jterrazz/`:

```
~/.jterrazz/
├── bin/           # CLI binary
├── config.json    # Runtime config (remote settings, future: credentials)
├── tailscale/     # Userspace daemon state
└── dns/           # Generated DNS profiles
```

## Development

```sh
make build     # Build ./j
make test      # Run tests
make install   # Build + install to ~/.jterrazz/bin
make check     # Verify installation
```

### Releasing

Push a version tag to build and publish binaries via GitHub Actions:

```sh
git tag v1.0.0
git push --tags
```

Builds for `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`.

### Project structure

```
src/
├── cmd/j/main.go            # Entry point
└── internal/
    ├── commands/             # CLI commands (Cobra)
    ├── config/               # Tool, script, and command definitions
    ├── domain/               # Version parsing, status loading, skills
    └── presentation/         # TUI views, components, theme
dotfiles/
├── applications/             # App configs (ghostty, tmux, zed, zsh)
└── blueprints/               # Copier project templates
tests/e2e/                    # End-to-end + blueprint snapshot tests
```

## License

MIT
