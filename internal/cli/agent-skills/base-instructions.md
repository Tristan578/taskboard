# Player2 Kanban: Agent Lifecycle Protocol

You are an AI agent assisting a developer. You MUST follow this lifecycle for every task.

## The Player2 Loop

### 1. Pre-Flight (Verify & Sync)
Before writing code:
- **Find or create a ticket:**
  ```bash
  player2-kanban ticket list --project <project_id>
  player2-kanban ticket create --project <project_id> --title "..." --priority medium
  ```
- **Check sync status:** `player2-kanban project sync-status`
- **Sync if needed:** `player2-kanban project sync <project_id> --async`

### 2. Strict Mode Gate
If the project uses **Strict Mode** (`player2-kanban project list` shows strict=true), non-draft tickets require:
- **User Story** and **Acceptance Criteria** before work begins.

Create tickets with these fields:
```bash
player2-kanban ticket create \
  --project <project_id> \
  --title "Add SSO login" \
  --user-story "As a user I want to log in with SSO So that I don't need another password" \
  --acceptance-criteria "Given a valid SSO session When I visit /login Then I am authenticated" \
  --priority high
```

Or create a draft first and fill in details later:
```bash
player2-kanban ticket create --project <project_id> --title "Investigate issue" --draft
```

### 3. Execution
- **Start work:** `player2-kanban ticket move <ticket_id> --status in_progress`
- **Track subtasks:** `player2-kanban ticket subtask toggle <subtask_id>`

### 4. Completion
When tests pass and the task is done:
- **Mark done:** `player2-kanban ticket move <ticket_id> --status done`
- **Sync:** `player2-kanban project sync <project_id> --async`
- **Notify the user** that the ticket is closed and synced.

## MCP Integration
If your IDE supports MCP (Model Context Protocol), connect to Player2 Kanban for direct tool access:
```bash
player2-kanban mcp
```
MCP provides 20+ tools for managing projects, tickets, subtasks, and boards — prefer MCP tools over CLI commands when available.

## Rules
- **Never bypass Strict Mode** by leaving production-ready work in Draft status.
- **Always move tickets through statuses** — don't skip from todo to done.
- **Sync before and after** significant work to keep GitHub Issues current.
- **Use the ticket ID** from `ticket list` output for all move/update/subtask operations.
