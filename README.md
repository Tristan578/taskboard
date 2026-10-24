# Player2 Kanban

> Preview: this branch adds guided setup and native desktop installers. It is not yet published on npm. PR artifacts are unsigned test builds; public distribution still needs signing and release validation.

**Player2** is a local board for your plans, tasks, and AI work. It helps you choose
a next step, work with an AI tool you already use, and see what is ready for review.
It does not include an AI service or build and host an app by itself.

Built on [tcarac/taskboard](https://github.com/tcarac/taskboard).

## Start with something useful

Pick a goal: build a first app, organize shifts and tips, plan a household week,
move your career forward, test a business idea, or ship technical work. Player2
creates an editable starter board. You can use your existing AI chat through
copy/paste, then add agent automation when you need it.

- Two-step setup with no required account, repository, prefix, or API key.
- Everyday boards let you review and complete tasks; software boards require
  verified merge/deployment evidence.
- Optional CLI/MCP agents, heartbeats, stale-work detection, and conflict-aware
  GitHub issue synchronization.
- Local storage with downloadable database backups.

[Install and create your first board](docs/getting-started.md) ·
[Workflow and migration details](docs/rebuild.md) · [Usability scope](docs/usability.md)

## Requirements

- **Go 1.24+** and **Node.js 22+** (for building from source)
- **Git** (for hook integration)
- Works on **Windows**, **macOS**, and **Linux**

## Installation

Desktop installers are built for Windows (`.exe`), macOS (`.pkg`) and Ubuntu/Debian
(`.deb`), plus portable binaries for other Linux distributions. No Node.js or Go is
required for desktop installation. See the [install guide](docs/getting-started.md)
for the preview/release status and the correct download for your computer.


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

### 1. Open Player2
```bash
player2-kanban open
# => Player2 Kanban running at http://localhost:3010 (pid 12345)
```

To run in the foreground (useful for development):
```bash
player2-kanban start --foreground
```

### 2. Optional: Connect to GitHub
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

### 3. Optional: Install Auto-Sync Hooks
Keep your board updated automatically whenever you push or pull code.
```bash
player2-kanban hook install <project_id>

# To remove hooks later:
player2-kanban hook uninstall
```

### 4. Configure your AI Agent
```bash
player2-kanban agent-config install all     # install for every supported IDE
# Or pick one: cursor, claude, gemini, windsurf, antigravity, copilot, codex
player2-kanban agent-config install cursor
```

| Agent | File Created |
|-------|-------------|
| cursor | `.cursor/rules/player2.mdc` |
| claude | `CLAUDE.md` |
| gemini | `.gemini/GEMINI.md` |
| windsurf | `.windsurfrules` |
| antigravity | `.agent/rules/player2.md` |
| copilot | `.github/copilot-instructions.md` |
| codex | `AGENTS.md` |

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
| `open [--port N] [--no-browser]` | Start/reuse Player2 and open your browser |
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

## Observable agent workflow (rebuild)

```bash
player2-kanban start
player2-kanban status
player2-kanban agent-config install all
player2-kanban activity TICKET_ID --agent Codex --session RUN_ID --summary "Investigating"
player2-kanban ticket move TICKET_ID --status in_progress
# Heartbeat at milestones and at least every five minutes.
player2-kanban ticket move TICKET_ID --status review
# Only after merge:
player2-kanban ticket complete TICKET_ID --evidence https://github.com/OWNER/REPO/pull/NUMBER
```

Use the actual ticket ID, not its display key. Configure `player2-kanban mcp` in
agent clients. Set `GITHUB_TOKEN` in the server environment for background sync and
in CLI/MCP environments when verifying completion. All processes must share the
same `--db` when a custom database is used. The board refreshes every five seconds.
The server is local-only; remote agents need a separately configured workspace.

Conflicts preserve both versions until you choose one in the ticket panel. Failed
sync appears on the board and can be retried. Stale activity does not automatically
close a task. See [supported sync fields and limitations](docs/rebuild.md).

To prevent a locally launched agent from skipping tracking, wrap its command:
```bash
player2-kanban run --ticket TICKET_ID --agent Codex -- codex
# Or pass another installed agent command and its arguments after --.
```
The launcher claims the task, sends process heartbeats every minute, and moves
In Progress to Review on successful exit or Blocked on failure. It never marks
Done from an exit code. Remote/cloud agents still need their own MCP/CLI connection.
