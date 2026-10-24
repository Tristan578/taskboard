# Player2 Kanban: Observable Agent Work

This managed block supersedes older Player2 workflow instructions.

Before implementation, call `player2_status` through MCP or `player2-kanban status`.
List projects and tickets, select the matching task, or create it. If the browser
is offline, run `player2-kanban start`. Use the same `--db` for every process.
If Player2 is unavailable, tell the user explicitly; never imply work is tracked.

1. Find/create a ticket. Strict projects require a user story and acceptance
   criteria for published work. Ideas may remain drafts in Todo.
2. Claim it with `report_activity` (id, agent, sessionId, summary), or:
   `player2-kanban activity <id> --agent Codex --session <unique-run-id> --summary "Starting investigation"`
3. Move it using `move_ticket` or `player2-kanban ticket move <id> --status in_progress`.
4. Send activity with the SAME sessionId at least every five minutes and on each
   meaningful milestone. After 15 minutes the board flags missing activity.
   Never take over another active session; stale sessions may be reclaimed.
5. Move to `blocked` with a progress summary describing what is needed, or
   `review` when implementation/tests are ready for review. Tests passing is not Done.
6. After merge or deployment, call `complete_ticket` with id/evidence, or
   `player2-kanban ticket complete <id> --evidence https://github.com/OWNER/REPO/pull/NUMBER`.
   Player2 verifies the merge. GitHub deployment API URLs are also supported;
   the latest deployment status must be success. Use evidence for THIS task.
7. Check sync health. Surface conflicts or failed sync to the user. Never say
   GitHub is current when sync is pending or failed.

MCP stdio setup (configure in your agent client's MCP settings):
```json
{"mcpServers":{"player2":{"command":"player2-kanban","args":["mcp"]}}}
```
Commands and MCP share the local database, including when the browser is closed.
The browser process performs background GitHub reconciliation and needs
GITHUB_TOKEN in its environment. Remote/cloud agents need an explicitly
configured connection to the same workspace; installing a rule alone does not
connect them. Devin instructions must be added to the agent's configured knowledge.

For local CLI agents, the developer can launch through Player2:
`player2-kanban run --ticket <id> --agent Codex -- codex <arguments>`.
This claims the task and sends process heartbeats automatically. Successful exit
moves In Progress to Review; failed/interrupted exit moves it to Blocked. The
process heartbeat is not evidence of implementation progress. Continue reporting
meaningful work summaries through the normal tools when possible.
