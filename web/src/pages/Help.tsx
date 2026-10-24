import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, type SyncStatus } from "../api/client";
import CopyText from "../components/CopyText";
export default function Help() {
  const [connection,setConnection] = useState("");
  const [sync, setSync] = useState<SyncStatus | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    api.connections().then(c=>setConnection(JSON.stringify(c,null,2))).catch(()=>setError("Could not prepare connection settings. Try reopening Player2."));
    api.sync
      .status()
      .then(setSync)
      .catch(() =>
        setError("Could not check the connection. Try reopening Player2."),
      );
  }, []);
  return (
    <div className="easy-page space-y-8">
      <div>
        <p className="text-teal-300 font-medium">Make yourself at home</p>
        <h1 className="text-3xl font-semibold mt-3">
          Your work, with a little help.
        </h1>
        <p className="mt-4 text-slate-300">
          Player2 keeps a plan visible while you and your AI work through it. It
          does not include an AI subscription or build an app by itself.
        </p>
      </div>
      <ol className="space-y-5 list-decimal pl-6 text-slate-300">
        <li>
          <strong className="text-white">Start with a goal.</strong> Create a
          board from an example or your own idea.
        </li>
        <li>
          <strong className="text-white">Choose one small task.</strong> Open it
          and use “Ask my AI for help” to copy its context into the chat you
          already use.
        </li>
        <li>
          <strong className="text-white">Bring the result back.</strong> Save
          useful answers in the task’s notes. Move it to Review when you are
          ready to check it.
        </li>
        <li>
          <strong className="text-white">Review before calling it done.</strong>{" "}
          Everyday boards let you mark it done yourself. Software boards verify
          a GitHub merge or successful deployment.
        </li>
      </ol>
      <Link to="/welcome" className="easy-button">
        Start a new board
      </Link>
      <Link to="/connect" className="easy-secondary">Connect an agent and try a first task</Link>
      <section className="border-t border-slate-800 pt-6 space-y-3">
        <h2 className="text-xl font-semibold">Use the AI you already have</h2>
        <p className="text-slate-300">
          Copy/paste works with your usual AI chat. No API key, developer
          account, or automatic connection is required. Only text you choose to
          copy is shared with your AI service.
        </p>
        <p className="text-slate-300">
          Automatic updates need a compatible agent connected through Player2’s
          tools. A chat subscription alone does not connect it to your computer.
          The board shows recent agent reports only when they actually arrive.
        </p>
      </section>
      <section className="border-t border-slate-800 pt-6 space-y-3">
        <h2 className="text-xl font-semibold">Where your work lives</h2>
        <p className="text-slate-300">
          Your boards are saved on this computer. Closing the browser does not
          delete them; open Player2 again to return. They do not automatically
          appear on your phone or another computer.
        </p>
        <p className="text-slate-300">
          Automatic backups are saved daily while Player2 runs. Use Backups & recovery to restore a backup or recover deleted tasks. Download a backup somewhere you trust. It includes your task content,
          so keep it private.
        </p>
        <Link to="/recovery" className="easy-secondary">Backups & recovery</Link>
        <a
          className="easy-secondary"
          href="/api/backup"
          download="player2-backup.db"
        >
          Download backup
        </a>
      </section>
      <details className="border-t border-slate-800 pt-6">
        <summary className="cursor-pointer text-lg font-semibold">
          Connect coding agents and GitHub · advanced
        </summary>
        <div className="mt-5 space-y-6">
          <p className="text-slate-300">
            Use this path when your agent can run commands on the same computer.
            Cloud agents do not automatically have access to this local board.
          </p>
          <div>
            <h3 className="font-semibold mb-3">1. Add project instructions</h3>
            <p className="text-sm text-slate-400 mb-3">
              Run this from your project folder. It preserves existing
              instructions and adds Player2’s workflow.
            </p>
            <CopyText
              text="player2-kanban agent-config install all"
              label="Copy setup command"
            />
          </div>
          <div>
            <h3 className="font-semibold mb-3">2. Connect an agent</h3>
            {connection && <div className="mb-4 space-y-3"><p className="text-sm text-slate-400">These settings point to this installation and database. Add the player2 entry to your agent’s existing MCP settings; keep its other connections.</p><CopyText text={connection} label="Copy agent settings"/><a href="/api/connections" download="player2-mcp.json" className="underline text-teal-300 text-sm">Download connection settings</a></div>}
            <p className="text-sm text-slate-400 mb-3">
              For an agent supporting local MCP tools, add a stdio server with
              command “player2-kanban” and argument “mcp”. Use its full
              installed path if the command cannot be found. Or run a local
              agent with tracking:
            </p>
            <CopyText
              text="player2-kanban run --ticket YOUR_TASK_ID --agent Codex -- codex"
              label="Copy tracking example"
            />
            <p className="mt-2 text-sm text-slate-400">
              Replace YOUR_TASK_ID with the ID inside a task and codex with your
              agent’s command. A process heartbeat means it is running, not that
              it has made progress.
            </p>
          </div>
          <div>
            <h3 className="font-semibold mb-3">
              3. Connect GitHub when you need it
            </h3>
            <p className="text-slate-300">
              Edit a software board under Boards and set its repository under
              Advanced. Start Player2 with GITHUB_TOKEN in its environment using
              your credential manager. Restart Player2 after changing that
              environment. Issue sync needs repository read/write access.
            </p>
            <p role="status" className="mt-3 text-sm text-teal-300">
              {sync
                ? sync.githubConfigured
                  ? "GitHub credentials detected. Permissions are checked when you sync."
                  : "GitHub is optional and is not configured. Local boards work without it."
                : "Checking local service…"}
            </p>
            {error && <p role="alert">{error}</p>}
          </div>
          <div>
            <h3 className="font-semibold mb-3">Troubleshooting and restore</h3>
            <CopyText
              text="player2-kanban status"
              label="Copy status command"
            />
            <p className="mt-3 text-sm text-slate-400">
              If port 3010 is occupied, run player2-kanban open --port 3011. To
              restore, stop Player2, keep a copy of the current database, and
              replace it with your backup at the path shown by status. Restore
              only backups you trust.
            </p>
          </div>
        </div>
      </details>
    </div>
  );
}
