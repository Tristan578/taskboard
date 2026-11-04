# Player2 development workflow

Use the CLI or MCP to discover the board and find or create the task before implementation.
Use `player2-kanban status` and `player2-kanban ticket list`. If no linked board or GitHub token is available, report the gap; do not claim live synchronization.

- Claim work with `player2-kanban activity <id> --agent Codex --session <id> --summary "Starting"`.
- Start using `player2-kanban ticket move <id> --status in_progress`.
- Report meaningful progress and move ready work to Review.
- Software boards require verified merged/deployed evidence for Done. Tests alone are insufficient. Use `ticket complete <id> --evidence <url>` after merge/deployment.
- Everyday boards use explicit human completion. Never confuse that with verified software delivery.
- Strict boards require a user story and acceptance criteria for published tasks.
- Preserve existing agent instructions and user data. Surface failed or conflicting sync.
