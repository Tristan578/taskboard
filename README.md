# Player2 Kanban

**Player2 Kanban** is an agent-native, local-first project management tool. It bridges the gap between fast "vibe coding" and professional team workflows by syncing local tasks directly with GitHub Issues.

This project is a fork of the excellent [tcarac/taskboard](https://github.com/tcarac/taskboard), enhanced with deep AI agent integration, strict engineering standards enforcement, and bidirectional GitHub synchronization.

## Screenshots

![Kanban Board](screenshots/board.png)

![Ticket Detail](screenshots/ticket-detail.png)

## Key Enhancements

- **Agent-Native Instructions** — Built-in commands to install `.cursorrules`, `.clauderules`, and `.gemini/GEMINI.md` to align AI assistants with your workflow.
- **Hard-Enforced Standards** — CI/CD gates for 75%+ test coverage, `gosec` security scans, and mandatory linear git history.
- **GitHub Sync Engine** — Bidirectional sync between local SQLite and GitHub Issues. Metadata is stored in hidden HTML comments.
- **Strict Mode** — Enforce professional standards. When enabled, tickets *must* have a User Story and Acceptance Criteria (Gherkin) before they can be created or updated.
- **Git Hooks** — Automated sync on `git push` and `git pull` via installed `pre-push` and `post-merge` hooks.
- **Embedded Terminal** — Full interactive shell in the web UI. Uses ConPTY on Windows (PowerShell/cmd.exe) and PTY on macOS/Linux (bash/zsh). Run commands, AI agents, or debug directly from the board.

## Requirements

- **Go 1.24+** and **Node.js 22+** (for building from source)
- **Git** (for hook integration)
- Works on **Windows**, **macOS**, and **Linux**

## Installation

### Via NPM

```bash
npm install -g player2-kanban
```

This downloads a pre-built binary for your platform from GitHub Releases.

### From Source

```bash
git clone https://github.com/Tristan578/taskboard.git
cd taskboard
make build
# Binary is at ./player2-kanban (add to your PATH)
```

### Manual Steps (if `make` is unavailable)

```bash
cd web && npm ci && npm run build && cd ..
mkdir -p cmd/kanban/web/dist && cp -r web/dist/* cmd/kanban/web/dist/
go build -o player2-kanban ./cmd/kanban
```

## Setup & Usage

### 1. Start the Server
```bash
player2-kanban start
# => Player2 Kanban running at http://localhost:3010 (pid 12345)
```

To run in the foreground (useful for development):
```bash
player2-kanban start --foreground
```

### 2. Connect to GitHub
```bash
# Set your GitHub Personal Access Token
export GITHUB_TOKEN="ghp_your_token_here"

# Link a project to a repository
player2-kanban project link <project_id> owner/repo

# Sync issues
player2-kanban project sync <project_id>

# Check sync status
player2-kanban project sync-status
```

### 3. Install Auto-Sync Hooks
Keep your board updated automatically whenever you push or pull code.
```bash
player2-kanban hook install <project_id>

# To remove hooks later:
player2-kanban hook uninstall
```

### 4. Configure your AI Agent
```bash
player2-kanban agent-config install cursor  # or claude, gemini, windsurf, antigravity, copilot, codex
```

## Strict Mode Enforcement

When a project is in **Strict Mode**, Player2 Kanban blocks any attempt (by human or agent) to create a non-draft ticket without:
1. **User Story**: `As a [role] I want [feature] So that [benefit]`
2. **Acceptance Criteria**: Gherkin format (`Given [context] When [action] Then [result]`)

Create tickets with strict mode fields from the CLI:
```bash
player2-kanban ticket create \
  --project <id> \
  --title "Add SSO login" \
  --user-story "As a user I want to log in with SSO So that I don't need another password" \
  --acceptance-criteria "Given a valid SSO session When I visit /login Then I am authenticated" \
  --priority high
```

Or create as a draft first and fill in details later:
```bash
player2-kanban ticket create --project <id> --title "Investigate flaky test" --draft
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `start [--port N] [--foreground]` | Start the web UI server (default: port 3010) |
| `stop` | Stop the running server |
| `mcp` | Start the MCP server (stdin/stdout) |
| `project create <name> --prefix <PFX>` | Create a project |
| `project list` | List all projects |
| `project link <id> <owner/repo>` | Link project to GitHub |
| `project sync <id> [--async]` | Sync with GitHub |
| `project sync-status` | Show sync queue status |
| `project delete <id>` | Delete a project |
| `ticket create --project <id> --title "..."` | Create a ticket |
| `ticket list [--project <id>] [--status todo\|in_progress\|done] [--priority urgent\|high\|medium\|low]` | List tickets |
| `ticket move <id> --status <status>` | Move ticket to status |
| `ticket delete <id>` | Delete a ticket |
| `ticket subtask add <ticket_id> "title"` | Add subtask |
| `ticket subtask toggle <id>` | Toggle subtask completion |
| `team create <name>` | Create a team |
| `team list` | List teams |
| `hook install <project_id>` | Install git sync hooks |
| `hook uninstall` | Remove git sync hooks |
| `agent-config install <agent>` | Install AI agent rules |
| `clear [--force]` | Delete all data |

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `GITHUB_TOKEN` | GitHub Personal Access Token for sync | (none) |
| `LOG_LEVEL` | Log verbosity: `debug`, `info`, `warn`, `error` | `info` |

Data is stored in your OS config directory (`%APPDATA%` on Windows, `~/.config` on Linux/macOS) as a SQLite database.

## Troubleshooting

**"player2-kanban is not running"** when stopping
— The server isn't started, or the PID file is stale. Run `player2-kanban start` first.

**"GITHUB_TOKEN environment variable not set"** during sync
— Export your token: `export GITHUB_TOKEN="ghp_..."`. You need a GitHub PAT with `repo` scope.

**Port already in use**
— Another instance may be running. Try `player2-kanban stop` first, or use `--port` to pick a different port: `player2-kanban start --port 3011`

**Strict mode blocking ticket creation**
— Non-draft tickets require `--user-story` and `--acceptance-criteria` flags. Use `--draft` to skip validation temporarily.

**Sync fails silently**
— Check the sync queue: `player2-kanban project sync-status`. Failed jobs show in the "Failed jobs" count. Check server logs for details (`LOG_LEVEL=debug`).

**Database location**
— Default: `~/.config/player2-kanban/` (Linux/macOS) or `%APPDATA%/player2-kanban/` (Windows). Override with `--db /path/to/file.db`.

## Tech Stack

| Layer        | Technology                                          |
| ------------ | --------------------------------------------------- |
| Backend      | Go (Cobra, Chi)                                     |
| Database     | SQLite (Pure Go)                                    |
| Frontend     | React, TypeScript, Tailwind CSS v4, dnd-kit         |
| Sync         | GitHub GraphQL & REST APIs                          |
| AI Integration| Model Context Protocol (MCP)                        |
| Distribution | Single binary with embedded assets                  |

## Acknowledgments

Based on [Taskboard](https://github.com/tcarac/taskboard) by tcarac.

## License

[MIT](LICENSE)
