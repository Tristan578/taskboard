import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, type ConnectionPilot, type ConnectionSetup } from "../api/client";
import CopyText from "../components/CopyText";
import SafeMarkdown from "../components/SafeMarkdown";

const storageKey = "player2-connection-pilot";
function savedAttempt() {
  try {
    return localStorage.getItem(storageKey) || "";
  } catch {
    return "";
  }
}

export default function Connect() {
  const [setup, setSetup] = useState<ConnectionSetup | null>(null);
  const [provider, setProvider] = useState("codex");
  const [allowed, setAllowed] = useState(false);
  const [goal, setGoal] = useState("");
  const [attempt, setAttempt] = useState(savedAttempt);
  const [pilot, setPilot] = useState<ConnectionPilot | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [pollError, setPollError] = useState("");
  const [storageWarning, setStorageWarning] = useState("");
  useEffect(() => {
    let active = true;
    api.connection
      .setup()
      .then((value) => {
        if (active) setSetup(value);
      })
      .catch(() => {
        if (active)
          setError(
            "Could not load connection settings. Reopen Player2 or reload this page.",
          );
      });
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    if (!attempt || busy) return;
    let active = true;
    let pending = false;
    async function refresh() {
      if (pending) return;
      pending = true;
      try {
        const value = await api.connection.pilot(attempt);
        if (active) {
          setPilot(value);
          setPollError("");
        }
      } catch (e) {
        if (active)
          setPollError(
            e instanceof Error
              ? e.message
              : "Cannot check progress. Your last result is still shown.",
          );
      } finally {
        pending = false;
      }
    }
    void refresh();
    const timer = setInterval(() => void refresh(), 3000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [attempt, busy]);
  function remember(id: string) {
    try {
      if (id) localStorage.setItem(storageKey, id);
      else localStorage.removeItem(storageKey);
    } catch {
      setStorageWarning(
        "This browser cannot remember this check. Keep this page open; the task itself is saved in My board.",
      );
    }
  }
  async function create() {
    setBusy(true);
    setError("");
    const id = attempt || `connect-${crypto.randomUUID()}`;
    remember(id);
    // Keep the same id after a lost response so Retry cannot duplicate boards.
    setAttempt(id);
    try {
      setPilot(await api.connection.create(id, goal));
      setPollError("");
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "Could not save your first task. Your goal is still here; try again.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function finish() {
    if (!pilot) return;
    setBusy(true);
    setError("");
    try {
      await api.tickets.move(
        pilot.task.id,
        "done",
        undefined,
        pilot.task.revision,
      );
      setPilot(await api.connection.pilot(pilot.id));
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "The task changed. Read the latest result before trying again.",
      );
    } finally {
      setBusy(false);
    }
  }
  function startAgain() {
    remember("");
    setAttempt("");
    setPilot(null);
    setPollError("");
    setError("");
    setGoal("");
    setAllowed(false);
  }
  const hasResult =
    pilot?.state === "needs_review" || pilot?.state === "completed";
  return (
    <div className="easy-page space-y-7">
      <div>
        <p className="text-teal-300 font-medium">Connect my AI</p>
        <h1 className="text-3xl font-semibold mt-3">
          See your AI’s work arrive here.
        </h1>
        <p className="mt-4 text-slate-300">
          Connect a local agent, try one useful task, and review its result.
          Your existing subscription and the agent’s own permissions still
          apply.
        </p>
      </div>
      {error && (
        <p role="alert" className="text-rose-300">
          {error}
        </p>
      )}
      {pollError && (
        <p role="alert" className="text-amber-300">
          Progress could not be refreshed: {pollError}
        </p>
      )}
      {storageWarning && <p role="status">{storageWarning}</p>}
      {!pilot ? (
        <>
          <section className="space-y-4" aria-labelledby="choose-agent">
            <h2 id="choose-agent" className="text-xl font-semibold">
              1. Choose the app on this computer
            </h2>
            <label className="easy-label">
              My AI app
              <select
                className="easy-input"
                value={provider}
                onChange={(e) => {
                  setProvider(e.target.value);
                  setAllowed(false);
                }}
              >
                <option value="codex">Codex CLI</option>
                <option value="claude">Claude Code</option>
                <option value="other">
                  Another local app with MCP support
                </option>
                <option value="chat">
                  I only use an AI chat website or phone app
                </option>
              </select>
            </label>
            {provider === "chat" ? (
              <div className="space-y-3">
                <p>
                  You can use Player2 with your usual chat through copy/paste.
                  Automatic updates need an agent that can use tools on this
                  computer; signing into a chat subscription alone does not
                  connect it.
                </p>
                <Link className="easy-button" to="/welcome">
                  Make a plan with my usual chat
                </Link>
                <Link className="block underline" to="/help">
                  See how copy/paste works
                </Link>
              </div>
            ) : (
              <>
                <p className="text-slate-300">
                  The agent must run on this same computer. This setup does not
                  connect a cloud agent, phone, or a separate WSL installation.
                </p>
                <p className="text-slate-300">
                  This connection lets your agent read and change{" "}
                  <strong>all boards in this Player2 database</strong>. Only
                  connect an agent you trust. Task content may be sent to its AI
                  provider under that provider’s settings.
                  Changes to linked software boards may synchronize to GitHub.
                </p>
                <label className="flex gap-3 items-start">
                  <input
                    type="checkbox"
                    checked={allowed}
                    onChange={(e) => setAllowed(e.target.checked)}
                  />
                  I understand the access and want to connect this local agent.
                </label>
                {allowed && setup && (
                  <div className="rounded-xl border border-slate-700 p-5 space-y-4">
                    <h3 className="font-semibold">Add Player2 to your agent</h3>
                    {provider === "other" ? (
                      <>
                        <p>
                          Add the entry below to your app’s existing MCP
                          settings. Keep its other connections. Your app must
                          support local stdio servers.
                        </p>
                        <CopyText
                          text={JSON.stringify(setup.config, null, 2)}
                          label="Copy connection settings"
                          copiedMessage="Copied. Add this entry to your app’s MCP settings."
                        />
                      </>
                    ) : (
                      <>
                        <p>
                          With{" "}
                          {provider === "codex" ? "Codex CLI" : "Claude Code"}{" "}
                          installed and signed in, open {setup.shell} and run
                          this command once. It registers this installation
                          under <code>{setup.name}</code>. If that name already
                          exists, inspect it first; do not replace a different
                          connection.
                        </p>
                        <p className="text-sm text-slate-400">
                          {setup.platform === "windows"
                            ? "Open Start, search for PowerShell, and open it. You do not need an administrator window."
                            : setup.platform === "darwin"
                              ? "Press Command + Space, type Terminal, and open it."
                              : "Open your applications menu and search for Terminal."}{" "}
                          The agent saves this connection for future sessions.
                          Keep its other connections.
                        </p>
                        <CopyText
                          text={
                            provider === "codex" ? setup.codex : setup.claude
                          }
                          label="Copy setup command"
                          copiedMessage={`Copied. Paste into ${setup.shell} and run it once.`}
                        />
                        <a
                          className="underline text-teal-300"
                          href={
                            provider === "codex"
                              ? "https://learn.chatgpt.com/docs/extend/mcp"
                              : "https://code.claude.com/docs/en/mcp"
                          }
                          target="_blank"
                          rel="noreferrer"
                        >
                          Official setup instructions
                        </a>
                      </>
                    )}
                    <p>
                      Reopen your agent session, check that Player2’s tools are
                      enabled, and return here. Copying settings does not
                      confirm a connection.
                    </p>
                  </div>
                )}
              </>
            )}
          </section>
          {provider !== "chat" && allowed && setup && (
            <form
              className="space-y-4 border-t border-slate-800 pt-6"
              onSubmit={(e) => {
                e.preventDefault();
                void create();
              }}
            >
              <h2 className="text-xl font-semibold">2. Try one useful task</h2>
              <p>
                We’ll create an everyday board and ask your agent for a short
                plan. Nothing runs until you send the instructions to your
                agent. Avoid private details for this first check.
              </p>
              <label className="easy-label">
                What would you like help planning?
                <textarea
                  className="easy-input"
                  rows={3}
                  required
                  maxLength={4000}
                  value={goal}
                  onChange={(e) => setGoal(e.target.value)}
                  placeholder="Prepare for a job interview, plan my week, organize a small project…"
                />
              </label>
              <button className="easy-button" disabled={busy || !goal.trim()}>
                {busy ? "Saving…" : "Create my first task"}
              </button>
            </form>
          )}
        </>
      ) : (
        <section className="space-y-5" aria-labelledby="connection-progress">
          <h2 id="connection-progress" className="text-xl font-semibold">
            3. Ask your agent, then watch this task
          </h2>
          <p>
            Copy these instructions into a new session in the agent you
            configured. Keep this page open or return later. The check watches
            actual reports and saved task content.
          </p>
          <CopyText
            text={
              (setup
                ? `Use the Player2 connection named ${setup.name}.\n\n`
                : "") + pilot.prompt
            }
            label="Copy first-task instructions"
          />
          <ol className="space-y-3" aria-label="Connection progress">
            <li>
              Task saved on this board: <strong>{pilot.task.title}</strong>
            </li>
            <li>
              {pilot.reported
                ? `Report received from ${pilot.task.work?.agent || "your agent"}.`
                : "Waiting for your agent’s first report. Nothing has connected yet."}
            </li>
            <li>
              {hasResult
                ? "A result was saved to the task."
                : "Waiting for a saved result and a move to Review."}
            </li>
            <li>
              {pilot.state === "completed"
                ? "Task marked Done. You can revisit the result on your board."
                : "Your review comes next."}
            </li>
          </ol>
          {pilot.reported && (
            <div role="status" className="rounded-xl bg-slate-900 p-4">
              <p>{pilot.task.work?.summary}</p>
              <p className="text-sm text-slate-400">
                Last report:{" "}
                {pilot.task.work?.heartbeatAt
                  ? new Date(pilot.task.work.heartbeatAt).toLocaleString()
                  : "unknown"}
                . This is an agent report, not proof that the process is still
                running.
              </p>
              {pilot.task.work?.stale && (
                <p className="text-amber-300">
                  No recent report. Ask your agent whether it is blocked before
                  retrying.
                </p>
              )}
            </div>
          )}
          {hasResult && (
            <section
              className="rounded-xl border border-teal-800 p-5 space-y-4"
              aria-label="Your agent’s result"
            >
              <h3 className="text-lg font-semibold">
                Read the result before calling it done
              </h3>
              <div className="prose prose-invert max-w-none">
                <SafeMarkdown>{pilot.task.description}</SafeMarkdown>
              </div>
              {pilot.state !== "completed" && (
                <button
                  className="easy-button"
                  disabled={busy}
                  onClick={() => void finish()}
                >
                  I reviewed the result — mark Done
                </button>
              )}
              <p className="text-sm text-slate-400">
                This is an everyday planning task. Software boards still require
                verified merge or deployment evidence.
              </p>
            </section>
          )}
          <Link
            className="easy-secondary"
            to={`/board?project=${encodeURIComponent(pilot.projectId)}`}
          >
            Open this board
          </Link>
          <details className="space-y-3">
            <summary className="cursor-pointer">
              No report yet, or the agent cannot find the task?
            </summary>
            <ul className="list-disc pl-5 space-y-2">
              <li>
                Open the agent’s tool or MCP settings and check that this
                installation is enabled.
              </li>
              <li>Start a new agent session after adding the connection.</li>
              <li>
                Approve the agent’s tool request if you are comfortable with its
                scope.
              </li>
              <li>
                If the task is missing, the agent is using another database.
                Return to setup and use this installation’s settings. Do not
                create a replacement task.
              </li>
              <li>
                A reply in chat alone does not update Player2. Ask the agent to
                use the listed tools.
              </li>
            </ul>
            <p>
              Agent names and reports are self-reported. This check proves that
              tools reached this task; it does not authenticate a provider or
              assess the quality of its answer.
            </p>
          </details>
        </section>
      )}
      {attempt && (
        <div className="border-t border-slate-800 pt-5 space-y-2">
          <button
            className="underline text-slate-300"
            disabled={busy}
            onClick={startAgain}
          >
            Start a new connection check
          </button>
          <p className="text-sm text-slate-400">
            Your existing board and task stay saved. A new check creates a
            separate board.
          </p>
        </div>
      )}
    </div>
  );
}
