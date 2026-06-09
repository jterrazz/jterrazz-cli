# j

A single CLI to bootstrap and manage a macOS development machine — tools, configs, templates, and remote access. No sudo required.

## Install

**Fresh machine** (no Go needed):

```sh
xcode-select --install
curl -fsSL https://raw.githubusercontent.com/jterrazz/jterrazz-cli/main/scripts/install.sh | sh
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

Full-screen TUI dashboard, organised into 4 tabs (`←/→` to cycle, `1..4` to jump directly):

- **System** — live CPU/Memory/GPU/Network sparklines, top processes, network, Tailscale peers, and system health (firewall, DNS, etc.)
- **Workspace** — tracked git repos, Docker containers, project dependencies
- **Applications** — 100+ tracked tools with versions, by category
- **Configuration** — every `j config` item with its current state, grouped by category (Terminal / Security / Editor / System / Server / Network / Identity). Server subsection only shows on a server-registered machine.

Everything loads in parallel with a progress bar; the System tab's live readings refresh every second.

### `j machine`

Manages a small registry of the machines you own — typically a client box (your laptop) and one or more servers — and runs status checks, remote actions, and server-only configuration.

#### Registry

Every machine has an alias, a role (`client` or `server`), and an optional SSH endpoint. The registry lives in `~/.jterrazz/config.json` and is the single source of truth — adding a machine also writes a managed `Host` block in `~/.ssh/config`.

```sh
j machine init                                                    # Bootstrap THIS machine (interactive)
j machine list                                                    # Table of registered machines (* marks self)
j machine add mac-mini --role server --ssh agent@192.168.1.106   # Add a remote
j machine add macbook  --role client                                 # Add a local-only entry
j machine remove mac-mini                                         # Refuses if alias is self
```

The role decides what `j machine status` reports and which items `j config` exposes for this box.

#### Inspect

```sh
j machine status              # FileVault, SSH, plus services (server role only)
j machine probe <alias>       # ping + ssh + OpenClaw gateway port + console owner
j machine restart <alias> -y  # FileVault-aware authrestart, waits for SSH to come back
j machine unlock <alias>      # Pre-boot SSH session to enter the FileVault password
```

`status` runs locally and adapts to the role:
- **client**: Machine state only — FileVault, SSH (port 22).
- **server**: Machine state + Services — OpenClaw runtime, OpenClaw config, channel health (Slack/Telegram/BlueBubbles), OrbStack.

`probe`/`restart`/`unlock` resolve the SSH endpoint from the registry. They refuse to act on the alias marked as self.

To configure the local machine (terminal, security, editor, system, server services), use `j config`.

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

### `j config`

Interactive TUI for configuring the local machine, organised into 3 tabs (`←/→` to cycle, `1..3` to jump directly):

- **Configuration** — installable items grouped by category. Sections are collapsible; items show their current state.
- **Skills** — install / list / remove AI agent skills (requires the `skills` CLI on PATH).
- **Remote** — read-only summary of the Tailscale endpoint; press `i` to open a form that rewrites `~/.jterrazz/config.json`.

```
 j config                                                self: mac-mini · server
 [Configuration]  Skills  Remote
 ──────────────────────────────────────────────────────────────────────────────
 ▾ Terminal               3/3
   ✓ ghostty
   ✓ tmux
   ✓ hushlogin

 ▸ Security               4/5
 ▾ Editor                 1/1
   ✓ zed

 ▸ System                 2/4
 ▾ Server                2/4
   ✓ autologin
 ▶ ✗ power
   ✓ lock-after-login
   ✗ sshd
 ──────────────────────────────────────────────────────────────────────────────
 ▶ power |  i install   space details
```

Categories on the Configuration tab (Server only appears when the current machine is registered as `server`):

- **Terminal** — ghostty, tmux, hushlogin
- **Security** — GPG commit signing, SSH keygen, GitHub CLI auth, encrypted DNS (Quad9), Spotlight exclusion
- **Editor** — Zed config
- **System** — JAVA_HOME, nvm, dock reset/spacer
- **Server** — autologin, power policy, lock-after-login, sshd

Keys:

| Key | Action |
|---|---|
| `←` `→` `1..3` | switch tab |
| `↑` `↓` `j` `k` | navigate |
| `tab` | collapse/expand current section |
| `space` | toggle the inline detail panel (Configuration tab) |
| `i` | install the current item (or open the reconfigure form on the Remote tab) |
| `u` | uninstall (only for toggleable items that are currently installed) |
| `q` `esc` | quit |

Items that need extra inputs (e.g. autologin's password) open a modal form before installing — built on [Charm's huh](https://github.com/charmbracelet/huh). Set `AGENT_PASSWORD` in your environment to pre-fill the autologin password field.

### `j remote`

```sh
j remote up       # Connect (userspace mode, SSH enabled, keep-awake)
j remote down     # Disconnect and stop daemon
j remote status   # Show connection state
```

Supports `auto`/`userspace` mode and `oauth`/`authkey` authentication. To change the endpoint settings, open `j config` and switch to the Remote tab.

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
├── config.json    # Runtime config (remote/Tailscale, machine registry)
├── tailscale/     # Userspace daemon state
└── dns/           # Generated DNS profiles
```

Schema of `config.json`:

```jsonc
{
  "remote":    { "mode": "userspace", "auth_method": "oauth", ... },
  "self":      "macbook",
  "machines": {
    "macbook":  { "role": "client" },
    "mac-mini": { "role": "server", "ssh": "agent@192.168.1.106" }
  }
}
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
└── applications/             # App configs (ghostty, tmux, zed, zsh)
tests/e2e/                    # End-to-end tests
```

## License

MIT
