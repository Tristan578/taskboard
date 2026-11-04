# Guided agent connection

Scope, October 24, 2026: turn local agent setup into an observed first task.
The owner authorized guided connection and a first-user pilot. `/connect` is
available as **Connect my AI** in the main navigation.

## Sources and decisions

| Source | Authority | Consequence |
| --- | --- | --- |
| Tristan's direction in this session | Product intent | Connect an agent and see useful work arrive with little assistance. |
| `connections.go`, `connection_pilot.go`, `connection.go` | Implementation | Settings use the installed executable and current database; the pilot watches saved task state. |
| [Official Codex MCP documentation](https://learn.chatgpt.com/docs/extend/mcp) | Provider syntax, checked October 24, 2026 | The CLI can register a local stdio server; ordinary web chat does not read local configuration. |
| [Official Claude Code MCP documentation](https://code.claude.com/docs/en/mcp) | Provider syntax, checked October 24, 2026 | The CLI can register stdio servers; user scope persists across projects. |
| `docs/rebuild.md` and agent instructions | Maintained contract | Protocol instructions do not establish tested native support for every vendor. |
| `docs/first-user-pilot.md` | Proposed research | Participant outcomes remain unobserved until sessions occur. |

The older Help page exposed generic JSON without guided verification. It remains
available. The new flow uses a database-specific name to avoid replacing an unrelated
`player2` entry. It does not edit client files or execute setup commands. Users review
access, then run the command in the indicated shell or add the MCP configuration.

## Observable contract

- Codex CLI and Claude Code get platform-quoted commands. Other local MCP clients get
  a configuration entry. Chat-only users get an explicit manual copy/paste path.
- Access covers all boards in this database. Task content may reach the AI provider;
  linked software changes may synchronize to GitHub. There is no per-board sandbox.
- A stable request ID creates one everyday board and one planning task atomically.
  Retries return the original task using existing workspace receipts; no migration
  is required. Starting another check preserves prior boards.
- The prompt identifies the connection, task and session. A missing task must stop
  the attempt rather than create replacements in another board.
- Only a report for that task with the matching session counts. A saved result also
  requires changed, nonempty notes and Review or Done status. Copying settings,
  replying in chat, or moving unchanged notes to Review cannot pass.
- The completion button uses the current revision. Done alone does not prove human
  review: trusted local clients can also edit an everyday task. Software completion
  rules are unchanged.
- Failed refreshes retain the last result and show an error. Old reports are labeled
  stale. The browser remembers the attempt ID; SQLite retains the task even if
  browser storage is unavailable. Reports do not authenticate a vendor or prove
  that an agent process is still running.

## Evidence and unknowns

Database, HTTP and component tests cover receipt reuse, unrelated sessions, missing
results, invalid requests, origin rejection, resume and completion. Native installer
checks drive the actual stdio MCP executable through initialization, task read,
activity, movement and result updates, then verify the HTTP view.

These are protocol tests, not live Claude/Codex sessions or participant observations.
Provider CLIs and signed-in accounts are unavailable in this execution environment.
Commands follow official documentation; actual client behavior, first-use comprehension
and interactive accessibility remain pilot checks. This flow does not connect cloud
agents, phones, or separate WSL installations to a local board automatically.
