import { Link, useSearchParams } from "react-router-dom";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  useDroppable,
  useDraggable,
  closestCorners,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
  type UniqueIdentifier,
} from "@dnd-kit/core";
import {
  Calendar,
  AlertTriangle,
  ArrowUp,
  ArrowRight,
  ArrowDown,
  CheckCircle2,
  FolderKanban,
  Users,
  Plus,
} from "lucide-react";
import { api, type Ticket, type Project, type Team, type BoardColumn, type SyncStatus } from "../api/client";
import TicketPanel from "../components/TicketPanel";
import CreateTicketModal from "../components/CreateTicketModal";

const STATUSES = ["todo", "in_progress", "review", "blocked", "done"];
const STATUS_LABELS: Record<string, string> = {
  todo: "To do",
  in_progress: "Working on it",
  review: "Review", blocked: "Needs help", done: "Done",
};
const STATUS_COLORS: Record<string, string> = {
  todo: "bg-slate-500",
  in_progress: "bg-blue-500",
  review: "bg-violet-500", blocked: "bg-amber-500", done: "bg-green-500",
};

const PRIORITY_CONFIG: Record<string, { color: string; icon: typeof ArrowUp }> = {
  urgent: { color: "text-red-500", icon: AlertTriangle },
  high: { color: "text-orange-500", icon: ArrowUp },
  medium: { color: "text-yellow-500", icon: ArrowRight },
  low: { color: "text-green-500", icon: ArrowDown },
};

function PriorityBadge({ priority }: { priority: string }) {
  const config = PRIORITY_CONFIG[priority];
  if (!config) return null;
  const Icon = config.icon;
  return (
    <span className={`inline-flex items-center gap-1 text-xs ${config.color}`}>
      <Icon className="w-3 h-3" />
      {priority}
    </span>
  );
}

function SubtaskProgress({ subtasks }: { subtasks: Ticket["subtasks"] }) {
  if (!subtasks || subtasks.length === 0) return null;
  const done = subtasks.filter((s) => s.completed).length;
  const pct = Math.round((done / subtasks.length) * 100);
  return (
    <div className="flex items-center gap-2 text-xs text-slate-500">
      <CheckCircle2 className="w-3 h-3" />
      <div className="flex-1 h-1 rounded-full bg-slate-700 overflow-hidden">
        <div
          className="h-full bg-blue-500 rounded-full transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span>
        {done}/{subtasks.length}
      </span>
    </div>
  );
}

function TicketCard({
  ticket,
  projects,
  teams,
  isDragging,
  onClick,
}: {
  ticket: Ticket;
  projects: Project[];
  teams: Team[];
  isDragging?: boolean;
  onClick?: () => void;
}) {
  const project = projects.find((p) => p.id === ticket.projectId);
  const team = teams.find((t) => t.id === ticket.teamId);

  return (
    <div
      onClick={onClick}
      className={`rounded-lg border border-slate-700/50 bg-slate-900 p-3 space-y-2 transition-colors hover:border-slate-600 cursor-pointer ${
        isDragging ? "opacity-90 shadow-xl shadow-blue-500/10 rotate-2" : ""
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="text-[11px] font-mono text-slate-500">
            {ticket.projectPrefix}-{ticket.number}
          </span>
          {ticket.isDraft && (
            <span className="px-1 py-0.5 text-[9px] font-bold uppercase tracking-wider bg-amber-500/10 text-amber-500 border border-amber-500/20 rounded">
              Draft
            </span>
          )}
        </div>
        <PriorityBadge priority={ticket.priority} />
      </div>
      <p className="text-sm text-slate-200 leading-snug">{ticket.title}</p>
      {(project || team) && (
        <div className="flex flex-wrap items-center gap-1.5">
          {project && (
            <span
              className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium"
              style={{
                backgroundColor: (project.color || "#3b82f6") + "1a",
                color: project.color || "#3b82f6",
              }}
            >
              <FolderKanban className="w-3 h-3" />
              {project.name}
            </span>
          )}
          {team && (
            <span
              className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium"
              style={{
                backgroundColor: (team.color || "#8b5cf6") + "1a",
                color: team.color || "#8b5cf6",
              }}
            >
              <Users className="w-3 h-3" />
              {team.name}
            </span>
          )}
        </div>
      )}
      {ticket.dueDate && (
        <div className="flex items-center gap-1.5 text-xs text-slate-500">
          <Calendar className="w-3 h-3" />
          {new Date(ticket.dueDate).toLocaleDateString()}
        </div>
      )}
      {ticket.work?.agent && <p className={`text-xs ${ticket.work.stale && ticket.status==="in_progress" ? "text-amber-400" : "text-cyan-400"}`}>{ticket.work.agent} · {ticket.work.stale ? "No recent heartbeat" : "Reported recently"}</p>}
      {project?.completionPolicy!=="manual" && ticket.status==="in_progress" && !ticket.work?.agent && <p className="text-xs text-amber-400">Untracked activity</p>}
      {ticket.work?.summary && <p className="line-clamp-2 text-xs text-slate-400">{ticket.work.summary}</p>}
      {ticket.work?.conflict && <p className="text-xs text-amber-400">GitHub conflict · resolve required</p>}
      {ticket.status==="done" && <p className="text-xs text-slate-400">{project?.completionPolicy==="manual" ? "Marked done" : ticket.work?.completionUrl ? "Completion verified" : "Legacy · evidence not recorded"}</p>}
      <SubtaskProgress subtasks={ticket.subtasks} />
    </div>
  );
}

function DraggableTicket({
  ticket,
  projects,
  teams,
  onClick,
}: {
  ticket: Ticket;
  projects: Project[];
  teams: Team[];
  onClick: () => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: ticket.id,
    data: { ticket },
  });

  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className={`cursor-grab active:cursor-grabbing ${isDragging ? "opacity-30" : ""}`}
    >
      <TicketCard ticket={ticket} projects={projects} teams={teams} onClick={onClick} />
    </div>
  );
}

function Column({
  status,
  tickets,
  projects,
  teams,
  onTicketClick,
  onAddTicket,
}: {
  status: string;
  tickets: Ticket[];
  projects: Project[];
  teams: Team[];
  onTicketClick: (ticket: Ticket) => void;
  onAddTicket: (status: string) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: status });

  return (
    <div className="flex flex-col w-80 shrink-0">
      <div className="flex items-center gap-2 px-1 pb-3">
        <div className={`w-2 h-2 rounded-full ${STATUS_COLORS[status]}`} />
        <h3 className="text-sm font-medium text-slate-300">
          {STATUS_LABELS[status]}
        </h3>
        <span className="text-xs text-slate-600 ml-auto">{tickets.length}</span>
        <button
          disabled={status === "done"}
          aria-label={`Add ticket to ${STATUS_LABELS[status]}`}
          onClick={() => onAddTicket(status)}
          className="text-slate-600 hover:text-slate-300 transition-colors"
        >
          <Plus className="w-4 h-4" />
        </button>
      </div>
      <div
        ref={setNodeRef}
        className={`flex-1 space-y-2 rounded-lg p-2 transition-colors min-h-32 ${
          isOver ? "bg-blue-500/5 ring-1 ring-blue-500/20" : ""
        }`}
      >
        {tickets.map((ticket) => (
          <DraggableTicket
            key={ticket.id}
            ticket={ticket}
            projects={projects}
            teams={teams}
            onClick={() => onTicketClick(ticket)}
          />
        ))}
        {tickets.length === 0 && (
          <div className="flex items-center justify-center h-24 text-xs text-slate-700 border border-dashed border-slate-800 rounded-lg">
            Drop tickets here
          </div>
        )}
      </div>
    </div>
  );
}

export default function Board() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [searchParams] = useSearchParams();
  const [selectedProject, setSelectedProject] = useState<string>(searchParams.get("project") || "");
  const [columns, setColumns] = useState<BoardColumn[]>([]);
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [createForStatus, setCreateForStatus] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [sync, setSync] = useState<SyncStatus | null>(null);
  const generation = useRef(0);
  const allTickets = columns.flatMap(c => c.tickets);
  const stale = allTickets.filter(t => t.status === "in_progress" && (projects.find(p=>p.id===t.projectId)?.completionPolicy!=="manual" || t.work?.agent) && (!t.work?.agent || t.work.stale)).length;
  const conflicts = allTickets.filter(t => t.work?.conflict).length;

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor)
  );

  const loadBoard = useCallback(async () => {
    const request = ++generation.current;
    try {
      const [board, health] = await Promise.all([api.board.get(selectedProject || undefined), api.sync.status()]);
      if(request !== generation.current) return;
      setColumns(board.columns || []); setSync(health); setError("");
    } catch(e) {
      if(request === generation.current) setError(e instanceof Error ? e.message : "Board unavailable");
    } finally { if(request === generation.current) setLoading(false); }
  }, [selectedProject]);

  useEffect(() => {
    api.projects.list().then(setProjects).catch(e => setError(String(e)));
    api.teams.list().then(setTeams).catch(e => setError(String(e)));
  }, []);

  useEffect(() => {
    if(!activeTicket) void loadBoard();
    const timer = window.setInterval(() => { if(!activeTicket) void loadBoard(); }, 5000);
    return () => { window.clearInterval(timer); };
  }, [loadBoard, activeTicket]);

  const getColumnTickets = (status: string) =>
    columns.find((c) => c.status === status)?.tickets || [];

  const findTicketById = (id: UniqueIdentifier): Ticket | undefined => {
    for (const col of columns) {
      const found = col.tickets.find((t) => t.id === id);
      if (found) return found;
    }
    return undefined;
  };

  const findColumnByTicketId = (id: UniqueIdentifier): string | undefined => {
    for (const col of columns) {
      if (col.tickets.find((t) => t.id === id)) return col.status;
    }
    return undefined;
  };

  const handleDragStart = (event: DragStartEvent) => {
    generation.current++;
    const ticket = findTicketById(event.active.id);
    setActiveTicket(ticket ?? null);
  };

  const handleDragOver = (event: DragOverEvent) => {
    const { active, over } = event;
    if (!over) return;

    const activeStatus = findColumnByTicketId(active.id);
    const overStatus = STATUSES.includes(over.id as string)
      ? (over.id as string)
      : findColumnByTicketId(over.id);

    if (!activeStatus || !overStatus || activeStatus === overStatus) return;

    setColumns((prev) =>
      prev.map((col) => {
        if (col.status === activeStatus) {
          return { ...col, tickets: col.tickets.filter((t) => t.id !== active.id) };
        }
        if (col.status === overStatus) {
          const ticket = findTicketById(active.id);
          if (!ticket) return col;
          return { ...col, tickets: [...col.tickets, { ...ticket, status: overStatus }] };
        }
        return col;
      })
    );
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveTicket(null);

    if (!over) { void loadBoard(); return; }

    const targetStatus = STATUSES.includes(over.id as string)
      ? (over.id as string)
      : findColumnByTicketId(over.id);

    if (!targetStatus) return;

    try {
      await api.tickets.move(active.id as string, targetStatus);
    } catch(e) {
      await loadBoard(); setError(e instanceof Error ? e.message : "Move failed");
    }
  };

  const handleTicketClick = (ticket: Ticket) => {
    setSelectedTicket(ticket);
  };

  const handleUpdate = async (id: string, data: Partial<Ticket>) => {
    await api.tickets.update(id, data);
    loadBoard();
  };

  const handleDelete = async (id: string) => {
    await api.tickets.delete(id);
    loadBoard();
  };

  const handleCreate = async (data: Partial<Ticket>) => {
    await api.tickets.create(data);
    setCreateForStatus(null);
    loadBoard();
  };

  return (
    <div className="h-full flex flex-col">
      <header className="shrink-0 flex items-center justify-between px-6 h-14 border-b border-slate-800">
        <h1 className="text-lg font-semibold text-white">My board</h1>
        <select
          aria-label="Choose a board"
          value={selectedProject}
          onChange={(e) => setSelectedProject(e.target.value)}
          className="bg-slate-800 text-sm text-slate-300 rounded-md border border-slate-700 px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value="">All boards</option>
          {projects.map((p) => (
            <option key={p.id} value={p.id}>
              {p.icon} {p.name}
            </option>
          ))}
        </select>
      </header>

      <div className="border-b border-slate-800 bg-slate-950 px-6 py-4 space-y-2">
       {(allTickets.some(t=>t.work?.agent) || projects.some(p=>p.githubRepo)) && <div className="flex flex-wrap gap-4 text-xs text-slate-400">
        <span className="text-cyan-400">● Updates every 5 seconds</span>
        <span>{allTickets.length} tasks</span><span className={stale ? "text-amber-400" : ""}>{stale} stale / untracked</span>
        <span className={conflicts ? "text-amber-400" : ""}>{conflicts} conflicts</span>
        <span>{sync?.pendingJobs ?? "…"} sync pending · {sync?.failedJobs ?? "…"} failed</span>
        {sync?.githubConfigured === false && <span className="text-amber-400">GitHub not connected · optional</span>}
        {!!sync?.failedJobs && <button onClick={() => api.sync.retry().then(loadBoard).catch(e=>setError(String(e)))} className="underline">Retry failed sync</button>}
       </div>}
       <p className="text-sm text-slate-400">Open a task to work on it or ask your AI for help. <Link to="/help" className="text-teal-300 underline">How it works</Link></p>
       {!loading && !error && allTickets.length===0 && <div className="py-4 text-slate-300">A useful plan starts with one small step. <Link to="/welcome" className="text-teal-300 underline">Create a board from an example</Link> or add a task below.</div>}
       {sync?.lastError && <p className="text-xs text-amber-400">Last sync error: {sync.lastError}</p>}
       {error && <div role="alert" className="text-sm text-red-400">{error} <button onClick={()=>void loadBoard()} className="underline">Retry</button></div>}
      </div>
      <div className="flex-1 overflow-x-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-full text-slate-600">
            Loading board…
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCorners}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
            onDragCancel={() => {setActiveTicket(null); void loadBoard();}}
          >
            <div className="flex gap-6 h-full">
              {STATUSES.map((status) => (
                <Column
                  key={status}
                  status={status}
                  tickets={getColumnTickets(status)}
                  projects={projects}
                  teams={teams}
                  onTicketClick={handleTicketClick}
                  onAddTicket={setCreateForStatus}
                />
              ))}
            </div>
            <DragOverlay>
              {activeTicket ? (
                <div className="w-80">
                  <TicketCard ticket={activeTicket} projects={projects} teams={teams} isDragging />
                </div>
              ) : null}
            </DragOverlay>
           </DndContext>
        )}
      </div>

      {createForStatus && (
        <CreateTicketModal
          projects={projects}
          teams={teams}
          defaultStatus={createForStatus}
          defaultProjectId={selectedProject}
          onClose={() => setCreateForStatus(null)}
          onCreate={handleCreate}
        />
      )}

      {selectedTicket && (
        <TicketPanel
          ticket={selectedTicket}
          projects={projects}
          teams={teams}
          onClose={() => {
            setSelectedTicket(null);
            loadBoard();
          }}
          onUpdate={handleUpdate}
          onDelete={handleDelete}
        />
      )}
    </div>
  );
}
