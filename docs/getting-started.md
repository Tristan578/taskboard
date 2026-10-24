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

Boards live on this computer, not in the browser cache. Closing the browser leaves
them intact. There is no automatic cloud backup or phone sync. Use **Help &
connections → Download backup** to save a consistent database snapshot somewhere
you trust. A backup contains private task data.

To restore: stop Player2, retain a copy of the current database, and replace it with
a trusted backup at the path shown by `player2-kanban status`. If restoring after an
unclean shutdown, have a technical person handle SQLite WAL/SHM sidecars rather
than mixing a backup with stale sidecar files. Reopen Player2.

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
