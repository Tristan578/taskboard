# Get started with Player2

Player2 keeps your tasks and AI work in one visible place. You can use it without
coding, a GitHub account, or an API key. It does not include an AI service or build
and host an app on its own.

This branch is a preview. Desktop installers are built as pull-request artifacts;
they are not yet a public, signed release. The package on npm does not yet include
these changes.

## Install and open

When the matching release is published, choose one installer from the release page:

| Computer | Installer | After installation |
| --- | --- | --- |
| Windows PC | `Player2-Setup-windows-amd64.exe` | Open Player2 from Start or the desktop shortcut |
| Windows on ARM | `Player2-Setup-windows-arm64.exe` | Open Player2 from Start |
| Mac with Apple silicon | `Player2-mac-arm64.pkg` | Open Player2 in Applications |
| Intel Mac | `Player2-mac-amd64.pkg` | Open Player2 in Applications |
| Ubuntu/Debian, Intel/AMD | `Player2-linux-amd64.deb` | Open the file in your software installer, then launch Player2 |
| Ubuntu/Debian, ARM64 | `Player2-linux-arm64.deb` | Open the file in your software installer, then launch Player2 |
| Other Linux distributions | Matching `linux-*.tar.gz` | Extract and run `Install.sh` to add a local application shortcut; requires `xdg-open` |

These desktop installers do not require Node.js, npm, Go, or a terminal for normal
Windows/Mac/Debian-family installation. The alternative Linux script is a terminal
path. Linux requires a graphical desktop and browser for the guided experience.
Windows installs for the current user. Mac and Debian installers may request
administrator permission. Removing the program preserves your local boards.

Preview installers are unsigned. Before broad distribution, the release owner must
sign Windows binaries/installers and sign/notarize the Mac app/package. Do not
teach people to disable OS security to install a preview.

Developers can keep using npm after matching assets are published:

```sh
npm install -g player2-kanban
player2-kanban open
```

Running `player2-kanban` without arguments also opens the browser and starts the
local service if needed. Opening it again reuses a healthy running service.

## Your first five minutes

1. Open Player2. On an empty installation it takes you to the guided start.
2. Choose an example or describe your own goal. A board is just a place for that goal.
3. Choose how work is completed. You can review everyday results yourself; GitHub
   software boards keep the verified merge/deployment rule.
4. Edit the suggested tasks. They are examples, not an AI-generated plan. You can
   replace them with a task list from your usual AI chat, one task per line.
5. Create your board. Open one task and use **Ask my AI for help**. Review and copy
   the instructions into your chat. Bring the result back to **Notes**, save it,
   and move it to **Review** or mark it done after checking it.

The first-app example starts with human-reviewed planning tasks. Its final task
asks you to publish the app, record the link, and test it. If you need automatic
GitHub verification, choose the software completion option when creating the board.
Completion policies cannot be silently switched on an existing board.

Copy/paste uses your existing chat service. Nothing is automatically sent to an AI
provider and your chat does not automatically update Player2. Agent automation is
an optional separate connection under **Help & connections → Advanced**.

## Keep your work safe

After setup, **Next up** helps you choose a task across your boards. Filter for
due work, reviews, or tasks that need help; search includes notes. Open a task to
edit it or ask your AI for help. The list refreshes while you use it.

If an agent changes a task while you are editing, Player2 keeps your draft open
and asks you to compare it with the latest saved version. Copy anything you want
to keep before continuing. Closing an unsaved editor or reloading the page warns
you first. Drafts in an open editor are not a backup; save your work before quitting.
External images in notes appear as links so opening a task does not contact the
image host automatically.

Boards live on this computer, not in browser cache. **Backups & recovery** saves a
backup once a day while the Player2 browser service runs and keeps the latest 14
copies. **Back up now** creates an immediate copy. Failures are shown on that page;
check them after running out of disk space. Download a backup to a trusted external
location for protection against computer loss or disk failure. Backups contain
private task data and are not encrypted or sent to a cloud service.

To recover, choose **Preview restore** beside a saved copy, or select a downloaded
Player2 `.db` backup up to 32 MB. Review its boards and task count, then confirm.
Player2 saves a safety backup and creates separate boards named “(restored)”. Your
current work stays in place. Repeating the same request after a lost response
returns the same restored boards. Cancel frees the preview; unused previews expire
after an hour. Older supported Player2 backups are upgraded in an isolated copy.

Recovered boards contain nondeleted tasks, notes, teams, labels, checklists and
available dependencies. In-progress tasks return to Todo, software Done returns to
Review, and GitHub links and active agent sessions are not reactivated. Review the
copies before reconnecting a software board. Unexpected schemas, damaged files and
broken references are rejected before any restored board is created.

To undo a task deletion, choose **Undo** in the notification or **Restore task** in
**Backups & recovery → Deleted tasks**. Notes and checklists are retained. If GitHub
is syncing, try again once it finishes; review any resulting sync conflict. The
page shows the most recent 200 deleted tasks. Deleting an entire board is not task
Undo; recover a prior backup if you need that board again.

## Repeat useful work

Set a task's due date and choose **Daily**, **Weekly** or **Monthly** under **Repeats**.
Completing it creates one next occurrence in Todo, copying the task text, team,
priority, labels and a fresh checklist. The old task remains in Done. Software
boards still require verified completion. Agent assignments, evidence and dependency
links are not copied into the next occurrence.

Dates follow the scheduled calendar cadence. Late completion skips missed dates
and produces one future occurrence. Monthly work keeps its original day where
possible: January 31 becomes February's last day, then March 31. Reopening and
completing an old occurrence does not create another duplicate. To stop a series,
choose **Does not repeat** on the newest occurrence. Repeats are completion-driven;
they do not send notifications or create tasks while the current one is unfinished.

Developers can also use `ticket create --due YYYY-MM-DD --repeat weekly` or the
`repeatRule` field through HTTP/MCP. The browser retains the simplest setup path.

## Connect a coding agent

The Help screen provides downloadable MCP settings containing the full executable
and database paths for this installation. Merge the `player2` entry into the agent's
existing MCP configuration. Installing instruction files alone is not a connection.
For project instructions, run `player2-kanban agent-config install all` in that
project directory. Existing instructions are preserved.

For an installed local CLI agent, use `player2-kanban run --ticket TASK_ID --agent
Codex -- codex`. Replace the task ID and agent command. This tracks the process
while it runs. Cloud agents require a connection to the same workspace; there is
no universal one-click connection to all providers in this release.

GitHub remains optional. Add a repository to a software board under **Boards →
Edit → Advanced**, then start Player2 with `GITHUB_TOKEN` set through your normal
credential manager. Restart the service after changing its environment. Do not
paste tokens into task notes, shared prompts, or downloaded connection settings.
