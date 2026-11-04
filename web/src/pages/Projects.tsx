import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, type Project } from "../api/client";
function EditBoard({
  project,
  onClose,
  onSaved,
}: {
  project: Project;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(project.name);
  const [description, setDescription] = useState(project.description || "");
  const [repo, setRepo] = useState(project.githubRepo || "");
  const [strict, setStrict] = useState(project.strict);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function save() {
    setBusy(true);
    setError("");
    try {
      await api.projects.update(project.id, {
        name: name.trim(),
        description,
        githubRepo: repo,
        strict,
      });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not save. Try again.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="fixed inset-0 z-50 bg-black/70 p-4 flex items-center justify-center">
      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="edit-board-title"
        className="max-w-xl w-full bg-slate-900 rounded-2xl p-6 max-h-[90dvh] overflow-auto space-y-5"
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
      >
        <h2 id="edit-board-title" className="text-xl font-semibold">
          Edit board
        </h2>
        <label className="easy-label">
          Name
          <input
            autoFocus
            required
            maxLength={120}
            className="easy-input"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <label className="easy-label">
          Goal and notes
          <textarea
            rows={4}
            className="easy-input"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </label>
        <p className="text-sm text-slate-400">
          {project.completionPolicy === "manual"
            ? "Everyday board · you review and mark tasks done."
            : "Software board · Done requires a verified merge or deployment."}{" "}
          This choice stays with the board.
        </p>
        <details>
          <summary className="cursor-pointer text-slate-300">Advanced</summary>
          <div className="space-y-4 pt-4">
            <p className="text-sm text-slate-400">
              Task prefix: {project.prefix}
            </p>
            {project.completionPolicy !== "manual" && (
              <>
                <label className="easy-label">
                  GitHub repository · optional
                  <input
                    className="easy-input"
                    value={repo}
                    onChange={(e) => setRepo(e.target.value)}
                    placeholder="https://github.com/owner/repository"
                  />
                </label>
                <Link className="text-teal-300 underline text-sm" to="/help">
                  Connection instructions
                </Link>
                <label className="flex gap-3 text-sm">
                  <input
                    type="checkbox"
                    checked={strict}
                    onChange={(e) => setStrict(e.target.checked)}
                  />
                  Require a user story and acceptance criteria before work
                  starts
                </label>
              </>
            )}
          </div>
        </details>
        {error && (
          <p role="alert" className="text-rose-300">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-3">
          <button
            type="button"
            className="easy-secondary"
            disabled={busy}
            onClick={onClose}
          >
            Cancel
          </button>
          <button className="easy-button" disabled={busy || !name.trim()}>
            {busy ? "Saving…" : "Save changes"}
          </button>
        </div>
      </form>
    </div>
  );
}
export default function Projects() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [editing, setEditing] = useState<Project | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  async function load() {
    try {
      setProjects((await api.projects.list()) || []);
      setError("");
    } catch {
      setError("Could not load your boards. Please try again.");
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void api.projects
      .list()
      .then((p) => setProjects(p || []))
      .catch(() => setError("Could not load your boards. Please try again."))
      .finally(() => setLoading(false));
  }, []);
  async function remove(p: Project) {
    if (
      !window.confirm(
        `Delete “${p.name}” and all its tasks? Download a backup first if you may need them.`,
      )
    )
      return;
    try {
      await api.projects.delete(p.id);
      await load();
    } catch {
      setError("Could not delete this board. Please try again.");
    }
  }
  return (
    <div className="easy-page">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold">Your boards</h1>
        <Link to="/welcome" className="easy-button">
          New board
        </Link>
      </div>
      <p className="text-slate-400 mt-4">
        A place for each goal, project, or part of your life.
      </p>
      {error && (
        <div role="alert" className="mt-6 text-rose-300">
          {error}{" "}
          <button className="underline" onClick={() => void load()}>
            Try again
          </button>
        </div>
      )}
      {loading ? (
        <p className="mt-8">Loading…</p>
      ) : (
        <div className="mt-8 divide-y divide-slate-800">
          {!projects.length && !error && (
            <p>No boards yet. Start with an example or your own idea.</p>
          )}
          {projects.map((p) => (
            <div key={p.id} className="py-5 space-y-3">
              <Link
                to={`/board?project=${encodeURIComponent(p.id)}`}
                className="text-xl font-medium text-teal-200"
              >
                {p.icon} {p.name}
              </Link>
              <p className="text-slate-400 line-clamp-2">{p.description}</p>
              <div className="flex gap-4 text-sm">
                <button className="underline" onClick={() => setEditing(p)}>
                  Edit
                </button>
                <button
                  className="text-slate-400 underline"
                  onClick={() => void remove(p)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
      {editing && (
        <EditBoard
          project={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
    </div>
  );
}
