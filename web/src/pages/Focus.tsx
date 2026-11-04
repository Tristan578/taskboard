import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowRight, Plus, Search } from "lucide-react";
import {
  api,
  type Ticket,
  type TicketUpdate,
  type Project,
  type Team,
} from "../api/client";
import {
  focusTasks,
  focusReason,
  displayDue,
  type FocusFilter,
} from "../lib/focus";
import TicketPanel from "../components/TicketPanel";
import CreateTicketModal from "../components/CreateTicketModal";
export default function Focus() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [selected, setSelected] = useState<Ticket | null>(null);
  const [create, setCreate] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<FocusFilter>("all");
  const [projectId, setProjectId] = useState("");
  const [shown, setShown] = useState(20);
  const generation = useRef(0);
  const load = useCallback(async () => {
    const current = ++generation.current;
    try {
      const [t, p, teams] = await Promise.all([
        api.tickets.list(),
        api.projects.list(),
        api.teams.list(),
      ]);
      if (current !== generation.current) return;
      setTickets(t || []);
      setProjects(p || []);
      setTeams(teams || []);
      setError("");
    } catch {
      if (current === generation.current)
        setError(
          "Could not refresh your tasks. Your last loaded list is still here.",
        );
    } finally {
      if (current === generation.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    let active = true;
    Promise.resolve().then(() => {
      if (active) void load();
    });
    const state = generation;
    return () => {
      active = false;
      state.current++;
    };
  }, [load]);
  useEffect(() => {
    if (selected || create) return;
    const timer = window.setInterval(() => {
      if (!document.hidden) void load();
    }, 15000);
    const refresh = () => void load();
    window.addEventListener("focus", refresh);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("focus", refresh);
    };
  }, [load, selected, create]);
  const visible = focusTasks(
    tickets.filter(
      (t) => projects.find((p) => p.id === t.projectId)?.status !== "archived",
    ),
    filter,
    query,
    projectId,
  );
  async function open(ticket: Ticket) {
    try {
      setSelected(await api.tickets.get(ticket.id));
    } catch {
      setError(
        "This task could not be opened. Refresh the list and try again.",
      );
    }
  }
  async function update(id: string, data: TicketUpdate) {
    const updated = await api.tickets.update(id, data);
    setSelected(updated);
    void load();
    return updated;
  }
  async function remove(id: string, revision?: number) {
    await api.tickets.delete(id, revision);
    void load();
  }
  return (
    <div className="easy-page">
      <p className="text-teal-300 font-medium">Next up</p>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-3xl sm:text-4xl font-semibold tracking-tight">
          One useful next step.
        </h1>
        <button
          className="easy-button"
          disabled={!projects.length || loading}
          onClick={() => setCreate(true)}
        >
          <Plus size={17} />
          Add a task
        </button>
      </div>
      <p className="mt-4 text-slate-400 leading-relaxed">
        Start with overdue work, something ready for review, or a task that
        needs help. Open a task to work on it with your AI.
      </p>
      <div className="mt-7 flex flex-wrap gap-3">
        <label className="flex-1 min-w-48 relative">
          <Search size={17} className="absolute left-3 top-6 text-slate-400" />
          <input
            aria-label="Search tasks"
            className="easy-input pl-10"
            placeholder="Find a task or a note…"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setShown(20);
            }}
          />
        </label>
        <select
          aria-label="Choose a board"
          className="easy-input w-auto"
          value={projectId}
          onChange={(e) => {
            setProjectId(e.target.value);
            setShown(20);
          }}
        >
          <option value="">All boards</option>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-wrap gap-2 mt-5" aria-label="Task filters">
        {(
          [
            ["all", "All open"],
            ["due", "Due & overdue"],
            ["review", "Ready for review"],
            ["blocked", "Needs help"],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            type="button"
            aria-pressed={filter === value}
            className={`rounded-full px-4 py-2 text-sm ${filter === value ? "bg-teal-300 text-slate-950" : "bg-slate-800 text-slate-300"}`}
            onClick={() => {
              setFilter(value);
              setShown(20);
            }}
          >
            {label}
          </button>
        ))}
      </div>
      {error && (
        <div role="alert" className="mt-5 text-rose-300">
          {error}{" "}
          <button className="underline" onClick={() => void load()}>
            Try again
          </button>
        </div>
      )}
      {loading ? (
        <p role="status" className="py-12">
          Finding your next steps…
        </p>
      ) : (
        <>
          <p className="mt-6 text-sm text-slate-400">
            {visible.length} open {visible.length === 1 ? "task" : "tasks"} ·
            Completed work and drafts stay on your board.
          </p>
          <div className="mt-3 divide-y divide-slate-800">
            {visible.slice(0, shown).map((t) => (
              <button
                key={t.id}
                className="w-full text-left flex gap-4 items-center py-5 px-2 hover:bg-slate-900 rounded-lg"
                onClick={() => void open(t)}
              >
                <div className="flex-1 min-w-0">
                  <span
                    className={`text-xs font-medium ${focusReason(t) === "Overdue" ? "text-amber-300" : "text-teal-300"}`}
                  >
                    {focusReason(t)}
                  </span>
                  <p className="mt-1 text-lg text-white break-words">
                    {t.title}
                  </p>
                  <p className="text-sm text-slate-400 mt-1">
                    {projects.find((p) => p.id === t.projectId)?.name ||
                      t.projectPrefix}
                    {t.dueDate ? ` · ${displayDue(t.dueDate)}` : ""}
                  </p>
                  {t.work?.summary && (
                    <p className="text-sm text-slate-300 mt-2 line-clamp-2">
                      {t.work.agent}: {t.work.summary}
                    </p>
                  )}
                </div>
                <ArrowRight size={18} className="shrink-0 text-slate-400" />
              </button>
            ))}
          </div>
          {!visible.length && !error && (
            <div className="py-12 space-y-3">
              <h2 className="text-xl">
                {query || filter !== "all"
                  ? "No matching tasks."
                  : "A little room to breathe."}
              </h2>
              <p className="text-slate-400">
                {query || filter !== "all"
                  ? "Try a different search or filter."
                  : "Choose one small thing you want to make progress on, or start from an example."}
              </p>
              <Link className="text-teal-300 underline" to="/welcome">
                Start a new board
              </Link>
            </div>
          )}
          {visible.length > shown && (
            <button
              className="easy-secondary mt-5"
              onClick={() => setShown(shown + 20)}
            >
              Show more tasks
            </button>
          )}
        </>
      )}
      <p className="mt-8 text-sm text-slate-400">
        <Link to="/board" className="underline">
          See the full board
        </Link>{" "}
        · Your list refreshes while you are here.
      </p>
      {selected && (
        <TicketPanel
          key={selected.id}
          ticket={selected}
          projects={projects}
          teams={teams}
          onClose={() => {
            setSelected(null);
            void load();
          }}
          onUpdate={update}
          onDelete={remove}
        />
      )}
      {create && (
        <CreateTicketModal
          projects={projects}
          teams={teams}
          defaultProjectId={projectId}
          onClose={() => setCreate(false)}
          onCreate={async (data) => {
            await api.tickets.create(data);
            setCreate(false);
            void load();
          }}
        />
      )}
    </div>
  );
}
