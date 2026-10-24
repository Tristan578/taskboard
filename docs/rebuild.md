# Player2 agent workflow rebuild

Player2 helps people completing tasks with AI see current work, identify stalled
work, and verify completion. This is a local developer tool with a browser UI,
Go/SQLite storage, an npm installer and provider-neutral MCP/CLI interfaces.

## Decisions confirmed by Tristan

- Primary audience: people completing tasks with AI.
- Agents put work on the board and keep it updated as execution progresses.
- Done means merged or deployed. Passing tests means Review.
- macOS, Windows and Linux are mandatory. Retain the npm developer-tool installer.
- The failures to address are stale tasks, missed board discovery and GitHub drift.
- Sync recommendation: preserve both versions on concurrent edits; require explicit resolution.

## Implemented contract

Todo (draft or published), In Progress, Review, Blocked, Done are supported through
browser, CLI and MCP. Shared database methods validate status, title, priority and
strict-mode requirements. A new Done transition requires completion verification.
A merged PR or latest successful deployment in the linked GitHub repository is
checked through GitHub's API. The caller associates that evidence with the task;
Player2 does not prove that the PR implements the acceptance criteria.

Agents report their provider name, a stable session ID and a progress summary.
A competing live session is rejected. Activity older than 15 minutes is stale;
an active task with no activity is untracked. Heartbeats do not claim the agent is
still running or that code has changed. Polling refreshes the browser every five
seconds; a failed fetch keeps the last board and shows an error.

Each process must use the same database. MCP works when the UI is closed, but the
browser server must run with GITHUB_TOKEN for background synchronization. Linked
projects are reconciled at startup and once a minute. Changes to published ticket
fields queue durable jobs through database triggers, including CLI/MCP changes.
Failures are retained, retried with backoff, then require an explicit retry.

Sync compares acknowledged local and remote content hashes. Existing divergent
pairs without a baseline become conflicts. Title, description, workflow status,
priority, user story, acceptance criteria, technical details and testing details
are synchronized. Workflow fields use the existing hidden metadata format.
GitHub issue closure alone becomes Review because closure is not proof of a merge
or deployment. Labels, comments, attachments, subtasks, dependencies, team assignment
and ordering are not fully mirrored; do not describe this as complete GitHub parity.
GitHub offers no transaction spanning its API and SQLite; concurrent external writes
during a sync request remain a race window. Stable ticket metadata permits recovery
from a successful issue creation followed by a local interruption.

Deletion tombstones are retained after closure is acknowledged, preventing later
syncs from reimporting deleted tasks. Deleted issue numbers are excluded from import. Project leases serialize sync
processes; each run has a two-minute deadline and a five-minute lock lease.

The HTTP server binds to loopback. Host validation blocks DNS rebinding; foreign
browser origins are rejected, including the terminal WebSocket. Local processes are
trusted. Do not expose this server through a tunnel or network proxy. There is no
remote multi-user authentication service in this version.

## Agent connection

Run `player2-kanban agent-config install all` in each repository. Installation uses
managed blocks to preserve unrelated instructions. Configure the printed MCP command
in each client's settings, or use the CLI. The repository includes instructions for
Claude, Codex, Cursor, Gemini, Antigravity, Copilot, Windsurf and Devin. These are
protocol instructions, not tested native integrations with every vendor. Devin/cloud
agents need explicit access to the same workspace and configured knowledge; a local
rules file alone cannot connect a remote service. An agent that ignores all integrations
cannot be forced onto the board. Missing reports are visible, not silently fabricated.

## Migration and rollback

Migration 010 is additive. Existing tickets, issue links and project configuration
remain. Historical Done tickets retain their status and are labeled as legacy when
no completion evidence exists. New Done transitions are gated. Existing sync hashes
without the new paired baseline trigger conservative conflict handling.

Before upgrading real data, stop Player2 and back up the SQLite database and any WAL
and SHM files together, or use SQLite's backup API. Test this branch against a copy.
Rollback by stopping the new binary and restoring that backup with the prior binary;
do not run old and new binaries against the same database concurrently. Do not delete
migration records to attempt rollback.

## Canon and evidence register

Scope: rebuild decisions and implementation, reviewed against main d44b728.

| Source | Authority | Supported claims / status |
|---|---|---|
| Tristan's answers in this session | Product owner decision | Audience, agent visibility, platforms, installer, Done definition |
| main d44b728 README | Maintained documentation | Original promises; not proof of runtime behavior |
| internal/server/server.go and terminal implementations | Direct code evidence | Original wildcard origins/all-interface listener and terminal access |
| internal/db/store.go, internal/mcp/mcp.go, internal/cli/ticket.go | Direct code evidence | Original HTTP-only validation and missing shared queues |
| internal/github/sync.go and worker.go | Direct code evidence | Original timestamp winner, swallowed partial errors, unsafe tombstone purge |
| web/src/pages/Board.tsx | Direct code evidence | Original load errors represented as an empty board |
| CONTRIBUTING.md and CI | Maintained policy/config | Tests, 75% coverage floor, lint/build, PR-based integration |
| New regression tests | Executable evidence | See verification results in PR; no claim of production validation |

The old generated lifecycle's instruction to mark Done after tests is superseded by
the owner's merged/deployed requirement. Timestamp-based overwrite is superseded by
the chosen conflict policy. Existing status is preserved as historical data, not as
newly verified completion. No active PRs existed at initial discovery.

Installer downloads are pinned to the npm package version and verified against
SHA256SUMS from that release. This branch prepares 2.0.0-beta.1; it is not published.

## Remaining release checks

Review and merge the PR, allow native macOS/Windows/Linux CI to pass, validate a real
GitHub token against a disposable linked repository, visually test the UI, then create
a matching GitHub release and publish the npm version. Never publish an npm package
before its matching release assets exist. No release or npm publication is performed
by this development branch. Windows ARM64 is built; native ARM64 execution needs hardware.

## Development verification

- Full Go suite: passed; statement coverage 82.0% (75% required).
- Go race detector on internal packages: passed.
- go vet: passed. gosec v2.22.0: passed.
- Frontend: five component tests, ESLint and production build passed.
- Installer: three Node tests passed; npm package contents inspected.
- Linux daemon smoke test: start/readiness, HTTP health, current embedded assets,
  and stop all passed using an isolated database.
- Cross-compilation: Windows x64/ARM64 and macOS ARM64 passed. Native runs remain CI checks.
- Browser visual verification: unavailable because the provided browser could not
  reach the development server. Component tests do not replace visual verification.
- GitHub reconciliation tests use a fake GitHub HTTP service and real SQLite.
  Live API/credential behavior must still be validated in a disposable repository.
