# Player2 Kanban

**Player2 Kanban** is an agent-native, local-first project management tool. It bridges the gap between fast "vibe coding" and professional team workflows by syncing local tasks directly with GitHub Issues.

This project is a fork of the excellent [tcarac/taskboard](https://github.com/tcarac/taskboard), enhanced with deep AI agent integration, strict engineering standards enforcement, and bidirectional GitHub synchronization.

## Screenshots

![Kanban Board](screenshots/board.png)

![Ticket Detail](screenshots/ticket-detail.png)

## Key Enhancements

- **Agent-Native Instructions** — Built-in commands to install `.cursorrules`, `.clauderules`, and `.gemini/GEMINI.md` to align AI assistants with your workflow.
- **GitHub Sync Engine** — Bidirectional sync between local SQLite and GitHub Issues. Metadata like User Stories and Gherkin ACs are stored in YAML frontmatter within the Issue body.
- **Strict Mode** — Enforce professional standards. When enabled, tickets *must* have a User Story and Acceptance Criteria (Gherkin) before they can be created or updated.
- **Git Hooks** — Automated sync on `git push` and `git pull` via installed `pre-push` and `post-merge` hooks.
- **Embedded Terminal** — Run AI coding agents directly from the web UI.

## Installation

### From Source

Requires Go 1.24+ and Node.js 22+.

```bash
git clone https://github.com/Tristan578/taskboard.git
cd taskboard
# Build frontend and backend
./scripts/build.sh # or follow manual steps in Makefile
```

### Via NPM (Beta)

```bash
npm install -g player2-kanban
```

## Setup & Usage

### 1. Start the Server
```bash
player2-kanban start
# Open http://localhost:3010
```

### 2. Connect to GitHub
```bash
# Set your token
export GITHUB_TOKEN="your_pat_token"

# Link a project to a repo
player2-kanban project link <project_id> https://github.com/owner/repo

# Perform initial sync
player2-kanban project sync <project_id>
```

### 3. Install Auto-Sync Hooks
Keep your board updated automatically whenever you push or pull code.
```bash
player2-kanban hook install <project_id>
```

### 4. Configure your AI Agent
```bash
player2-kanban agent-config install cursor  # or claude, gemini, windsurf, antigravity
```

## Strict Mode Enforcement

When a project is in **Strict Mode**, Player2 Kanban blocks any attempt (by human or agent) to create a ticket without:
1. **User Story**: `As a... I want... So that...`
2. **Acceptance Criteria**: Gherkin format (`Given... When... Then...`)

This ensures that tasks are well-defined before implementation begins, preventing "context drift" in agentic workflows.

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
