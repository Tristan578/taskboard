import TaskAIHelp from "./TaskAIHelp";
import WorkEvidence from "./WorkEvidence";
import { useEffect, useState } from "react";
import { X, Trash2, CheckCircle2, Circle, Pencil, Eye } from "lucide-react";
import Markdown from "./SafeMarkdown";
import {
  api,
  ApiError,
  type TicketUpdate,
  type Ticket,
  type Project,
  type Team,
  type Subtask,
} from "../api/client";

const STATUSES = ["todo", "in_progress", "review", "blocked", "done"];
const PRIORITIES = ["urgent", "high", "medium", "low"];

const STATUS_LABELS: Record<string, string> = {
  todo: "Todo",
  in_progress: "In Progress",
  review: "Review",
  blocked: "Blocked",
  done: "Done",
};

export default function TicketPanel({
  ticket,
  projects,
  teams,
  onClose,
  onUpdate,
  onDelete,
}: {
  ticket: Ticket;
  projects: Project[];
  teams: Team[];
  onClose: () => void;
  onUpdate: (id: string, data: TicketUpdate) => Promise<Ticket>;
  onDelete: (id: string, revision?: number) => Promise<void>;
}) {
  const [title, setTitle] = useState(ticket.title);
  const [description, setDescription] = useState(ticket.description || "");
  const [status, setStatus] = useState(ticket.status);
  const [priority, setPriority] = useState(ticket.priority);
  const [repeatRule, setRepeatRule] = useState(ticket.repeatRule || "");
  const [dueDate, setDueDate] = useState(ticket.dueDate?.slice(0, 10) || "");
  const [teamId, setTeamId] = useState(ticket.teamId || "");
  const [userStory, setUserStory] = useState(ticket.userStory || "");
  const [acceptanceCriteria, setAcceptanceCriteria] = useState(
    ticket.acceptanceCriteria || "",
  );
  const [technicalDetails, setTechnicalDetails] = useState(
    ticket.technicalDetails || "",
  );
  const [testingDetails, setTestingDetails] = useState(
    ticket.testingDetails || "",
  );
  const [subtasks, setSubtasks] = useState<Subtask[]>(ticket.subtasks || []);
  const [newSubtask, setNewSubtask] = useState("");
  const [busy, setBusy] = useState(false);
  const [baseRevision, setBaseRevision] = useState(ticket.revision);
  const [conflict, setConflict] = useState(false);
  const [latest, setLatest] = useState<Ticket | null>(null);
  const [saveError, setSaveError] = useState("");
  const [dirty, setDirty] = useState(false);
  const [descMode, setDescMode] = useState<"preview" | "write">(
    description ? "preview" : "write",
  );

  const markDirty = () => setDirty(true);

  const project = projects.find((p) => p.id === ticket.projectId);

  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  const closeSafely = () => {
    if (!busy && (!dirty || window.confirm("Discard your unsaved changes?")))
      onClose();
  };
  const handleSave = async (publish = false) => {
    setSaveError("");
    setBusy(true);
    try {
      const updated = await onUpdate(ticket.id, {
        title,
        description,
        status,
        priority,
        repeatRule,
        dueDate,
        teamId,
        userStory,
        acceptanceCriteria,
        technicalDetails,
        testingDetails,
        expectedRevision: baseRevision,
        ...(publish ? { isDraft: false } : {}),
      });
      setTitle(updated.title);
      setDescription(updated.description || "");
      setStatus(updated.status);
      setPriority(updated.priority);
      setRepeatRule(updated.repeatRule || "");
      setDueDate(updated.dueDate?.slice(0, 10) || "");
      setTeamId(updated.teamId || "");
      setUserStory(updated.userStory || "");
      setAcceptanceCriteria(updated.acceptanceCriteria || "");
      setTechnicalDetails(updated.technicalDetails || "");
      setTestingDetails(updated.testingDetails || "");
      setSubtasks(updated.subtasks || []);
      setBaseRevision(updated.revision);
      setDirty(false);
      setConflict(false);
      setLatest(null);
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : "Save failed");
      if (e instanceof ApiError && e.status === 409) setConflict(true);
    } finally {
      setBusy(false);
    }
  };
  const compareLatest = async () => {
    setBusy(true);
    try {
      setLatest(await api.tickets.get(ticket.id));
    } catch (e) {
      setSaveError(
        e instanceof Error ? e.message : "Could not load the latest task.",
      );
    } finally {
      setBusy(false);
    }
  };
  const handleDelete = async () => {
    if (
      !window.confirm(
        "Move this task to Deleted tasks? You can restore it from Backups & recovery.",
      )
    )
      return;
    setBusy(true);
    setSaveError("");
    try {
      await onDelete(ticket.id, baseRevision);
      onClose();
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : "Delete failed");
      if (e instanceof ApiError && e.status === 409) setConflict(true);
    } finally {
      setBusy(false);
    }
  };

  const subtaskAction = async (action: () => Promise<void>) => {
    setBusy(true);
    setSaveError("");
    try {
      await action();
    } catch (e) {
      setSaveError(
        e instanceof Error
          ? e.message
          : "Checklist update failed. Please try again.",
      );
    } finally {
      setBusy(false);
    }
  };
  const handleAddSubtask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSubtask.trim()) return;
    await subtaskAction(async () => {
      const sub = await api.tickets.addSubtask(ticket.id, newSubtask.trim());
      setSubtasks((prev) => [...prev, sub]);
      setNewSubtask("");
    });
  };
  const handleToggleSubtask = async (id: string) => {
    await subtaskAction(async () => {
      const updated = await api.subtasks.toggle(id);
      setSubtasks((prev) => prev.map((s) => (s.id === id ? updated : s)));
    });
  };
  const handleDeleteSubtask = async (id: string) => {
    await subtaskAction(async () => {
      await api.subtasks.delete(id);
      setSubtasks((prev) => prev.filter((s) => s.id !== id));
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/40" onClick={closeSafely} />
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Task details"
        onKeyDown={(e) => {
          if (e.key === "Escape") closeSafely();
        }}
        className="relative w-full max-w-xl bg-slate-900 border-l border-slate-700 overflow-y-auto"
      >
        <div className="sticky top-0 z-10 bg-slate-900 border-b border-slate-800 px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-xs font-mono text-slate-500">
              {ticket.projectPrefix}-{ticket.number}
            </span>
            {ticket.isDraft && (
              <span className="px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider bg-amber-500/10 text-amber-500 border border-amber-500/20 rounded">
                Draft
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              aria-label="Delete task"
              disabled={busy}
              onClick={() => void handleDelete()}
              className="text-slate-500 hover:text-red-400 transition-colors"
            >
              <Trash2 className="w-4 h-4" />
            </button>
            <button
              aria-label="Close task"
              disabled={busy}
              onClick={closeSafely}
              className="text-slate-500 hover:text-slate-300 transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        <fieldset disabled={busy} className="p-6 space-y-6 min-w-0">
          <p className="text-xs text-slate-500 break-all">
            Task ID: {ticket.id}
          </p>
          <TaskAIHelp
            ticket={{ ...ticket, title, description }}
            project={project}
          />
          {dirty && (
            <p role="status" className="text-sm text-amber-200">
              Unsaved changes. Save your notes before completing this task.
            </p>
          )}
          <fieldset disabled={dirty || busy}>
            <WorkEvidence
              manual={project?.completionPolicy === "manual"}
              ticket={ticket}
              onRefresh={onClose}
              onBusyChange={setBusy}
            />
          </fieldset>
          {saveError && (
            <p role="alert" className="text-red-400">
              {saveError}
            </p>
          )}
          {conflict && (
            <section className="border border-amber-700 rounded-lg p-4 space-y-3">
              <p className="text-sm">
                Your draft has not replaced the saved task. Compare both
                versions before continuing.
              </p>
              <button
                className="easy-secondary"
                disabled={busy}
                onClick={() => void compareLatest()}
              >
                View latest saved version
              </button>
              {latest && (
                <>
                  <dl
                    aria-label="Latest saved task"
                    className="text-sm space-y-3 max-h-72 overflow-auto"
                  >
                    {[
                      ["Title", latest.title],
                      ["Notes", latest.description],
                      ["Status", STATUS_LABELS[latest.status] || latest.status],
                      ["Priority", latest.priority],
                      ["Repeats", latest.repeatRule || "Does not repeat"],
                      ["Due date", latest.dueDate?.slice(0, 10)],
                      [
                        "Team",
                        teams.find((team) => team.id === latest.teamId)?.name,
                      ],
                      ["User story", latest.userStory],
                      ["Acceptance criteria", latest.acceptanceCriteria],
                      ["Technical details", latest.technicalDetails],
                      ["Testing details", latest.testingDetails],
                    ].map(([label, value]) => (
                      <div key={label}>
                        <dt className="text-slate-400">{label}</dt>
                        <dd className="whitespace-pre-wrap break-words">
                          {value || "None"}
                        </dd>
                      </div>
                    ))}
                  </dl>
                  <p className="text-sm text-slate-300">
                    Copy any changes you want to keep into your draft.
                    Continuing lets your next save replace the fields shown
                    above.
                  </p>
                  <button
                    className="easy-secondary"
                    disabled={busy}
                    onClick={() => {
                      setBaseRevision(latest.revision);
                      setConflict(false);
                      setSaveError("");
                      setLatest(null);
                    }}
                  >
                    I’ve compared them; continue with my draft
                  </button>
                </>
              )}
            </section>
          )}
          <input
            autoFocus
            aria-label="Task title"
            value={title}
            onChange={(e) => {
              setTitle(e.target.value);
              markDirty();
            }}
            className="w-full bg-transparent text-lg font-semibold text-white focus:outline-none"
          />

          <div>
            <div className="flex items-center gap-1 mb-1.5">
              <button
                type="button"
                onClick={() => setDescMode("write")}
                className={`inline-flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors ${
                  descMode === "write"
                    ? "bg-slate-700 text-white"
                    : "text-slate-500 hover:text-slate-300"
                }`}
              >
                <Pencil className="w-3 h-3" />
                Write
              </button>
              <button
                type="button"
                onClick={() => setDescMode("preview")}
                className={`inline-flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors ${
                  descMode === "preview"
                    ? "bg-slate-700 text-white"
                    : "text-slate-500 hover:text-slate-300"
                }`}
              >
                <Eye className="w-3 h-3" />
                Preview
              </button>
            </div>
            {descMode === "write" ? (
              <textarea
                aria-label="Task notes"
                value={description}
                onChange={(e) => {
                  setDescription(e.target.value);
                  markDirty();
                }}
                rows={8}
                placeholder="Add a description (supports markdown)…"
                className="w-full bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 resize-y font-mono min-h-[10.5rem]"
              />
            ) : description ? (
              <div className="prose-card bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2 min-h-[10.5rem] overflow-y-auto">
                <Markdown>{description}</Markdown>
              </div>
            ) : (
              <div
                onClick={() => setDescMode("write")}
                className="bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2 min-h-[10.5rem] text-sm text-slate-600 cursor-text"
              >
                Add a description…
              </div>
            )}
          </div>

          {(project?.strict ||
            userStory ||
            acceptanceCriteria ||
            technicalDetails ||
            testingDetails) && (
            <div className="space-y-4 border-t border-slate-800 pt-6">
              <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                Agentic Details{" "}
                {project?.strict && (
                  <span className="text-blue-500 ml-1">(Strict)</span>
                )}
              </h4>

              <div className="space-y-3">
                <div>
                  <label className="block text-xs font-medium text-slate-500 mb-1">
                    User Story
                  </label>
                  <textarea
                    value={userStory}
                    onChange={(e) => {
                      setUserStory(e.target.value);
                      markDirty();
                    }}
                    placeholder="As a... I want... So that..."
                    className={`w-full bg-slate-800/50 border rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500 ${
                      project?.strict && !userStory
                        ? "border-red-500/50"
                        : "border-slate-700"
                    }`}
                  />
                  {project?.strict && !userStory && (
                    <p className="text-[10px] text-red-400 mt-1">
                      Required in Strict Mode
                    </p>
                  )}
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-500 mb-1">
                    Acceptance Criteria (Gherkin)
                  </label>
                  <textarea
                    value={acceptanceCriteria}
                    onChange={(e) => {
                      setAcceptanceCriteria(e.target.value);
                      markDirty();
                    }}
                    placeholder="Given... When... Then..."
                    className={`w-full bg-slate-800/50 border rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono ${
                      project?.strict && !acceptanceCriteria
                        ? "border-red-500/50"
                        : "border-slate-700"
                    }`}
                  />
                  {project?.strict && !acceptanceCriteria && (
                    <p className="text-[10px] text-red-400 mt-1">
                      Required in Strict Mode
                    </p>
                  )}
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-500 mb-1">
                    Technical Implementation
                  </label>
                  <textarea
                    value={technicalDetails}
                    onChange={(e) => {
                      setTechnicalDetails(e.target.value);
                      markDirty();
                    }}
                    placeholder="Proposed architectural changes..."
                    className="w-full bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-500 mb-1">
                    Testing Strategy
                  </label>
                  <textarea
                    value={testingDetails}
                    onChange={(e) => {
                      setTestingDetails(e.target.value);
                      markDirty();
                    }}
                    placeholder="Unit tests, integration tests..."
                    className="w-full bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              </div>
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1.5">
                Status
              </label>
              <select
                value={status}
                onChange={(e) => {
                  setStatus(e.target.value);
                  markDirty();
                }}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                {STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABELS[s]}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1.5">
                Priority
              </label>
              <select
                value={priority}
                onChange={(e) => {
                  setPriority(e.target.value);
                  markDirty();
                }}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500 capitalize"
              >
                {PRIORITIES.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1.5">
                Due Date
              </label>
              <input
                type="date"
                aria-label="Due date"
                value={dueDate}
                onChange={(e) => {
                  setDueDate(e.target.value);
                  markDirty();
                }}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1.5">
                Team
              </label>
              <select
                value={teamId}
                onChange={(e) => {
                  setTeamId(e.target.value);
                  markDirty();
                }}
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                <option value="">None</option>
                {teams.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1.5">
                Project
              </label>
              <div className="text-sm text-slate-400 px-3 py-2">
                {projects.find((p) => p.id === ticket.projectId)?.name || "—"}
              </div>
            </div>
          </div>

          <label className="block text-sm text-slate-300">
            Repeats
            <select
              aria-label="Repeats"
              className="easy-input"
              value={repeatRule}
              onChange={(e) => {
                setRepeatRule(e.target.value);
                markDirty();
              }}
            >
              <option value="">Does not repeat</option>
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
            {repeatRule && (
              <span className="block mt-2 text-xs text-slate-400">
                Choose a due date. Completing this task creates one next
                occurrence with a fresh checklist. Missed dates are skipped;
                stopping repeats on the newest task ends the series.
              </span>
            )}
          </label>
          {dirty && (
            <button
              onClick={() => void handleSave()}
              disabled={busy || conflict}
              className="w-full px-4 py-2 text-sm font-medium bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors"
            >
              Save Changes
            </button>
          )}

          {ticket.isDraft && (
            <button
              disabled={busy || conflict}
              onClick={() => {
                if (
                  project?.strict &&
                  (!userStory.trim() || !acceptanceCriteria.trim())
                ) {
                  alert(
                    "Strict mode: User Story and Acceptance Criteria are required before publishing.",
                  );
                  return;
                }
                void handleSave(true);
              }}
              className="w-full px-4 py-2 text-sm font-medium bg-amber-600 hover:bg-amber-500 text-white rounded-lg transition-colors"
            >
              Publish Ticket (Sync to GitHub)
            </button>
          )}

          <div>
            <h4 className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-3">
              Subtasks
            </h4>
            <div className="space-y-1.5">
              {subtasks.map((sub) => (
                <div
                  key={sub.id}
                  className="flex items-center gap-2.5 group px-2 py-1.5 rounded-md hover:bg-slate-800/50"
                >
                  <button
                    onClick={() => handleToggleSubtask(sub.id)}
                    className="shrink-0"
                  >
                    {sub.completed ? (
                      <CheckCircle2 className="w-4 h-4 text-green-500" />
                    ) : (
                      <Circle className="w-4 h-4 text-slate-600" />
                    )}
                  </button>
                  <span
                    className={`flex-1 text-sm ${
                      sub.completed
                        ? "text-slate-600 line-through"
                        : "text-slate-300"
                    }`}
                  >
                    {sub.title}
                  </span>
                  <button
                    onClick={() => handleDeleteSubtask(sub.id)}
                    className="opacity-0 group-hover:opacity-100 text-slate-600 hover:text-red-400 transition-all"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
            </div>
            <form onSubmit={handleAddSubtask} className="mt-2 flex gap-2">
              <input
                value={newSubtask}
                onChange={(e) => setNewSubtask(e.target.value)}
                placeholder="Add subtask…"
                className="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-white placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <button
                type="submit"
                className="px-3 py-1.5 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg border border-slate-700 transition-colors"
              >
                Add
              </button>
            </form>
          </div>
        </fieldset>
      </div>
    </div>
  );
}
