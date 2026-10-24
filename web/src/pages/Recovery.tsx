import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  api,
  type RecoveryStatus,
  type RecoveryPreview,
  type DeletedTask,
} from "../api/client";

export default function Recovery() {
  const [status, setStatus] = useState<RecoveryStatus | null>(null);
  const [trash, setTrash] = useState<DeletedTask[]>([]);
  const [preview, setPreview] = useState<RecoveryPreview | null>(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [message, setMessage] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const load = useCallback(async () => {
    try {
      const [s, t] = await Promise.all([
        api.recovery.status(),
        api.recovery.trash(),
      ]);
      setStatus(s);
      setTrash(t);
    } catch {
      setError("Could not read recovery options. Reopen Player2 or try again.");
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  async function act(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      await action();
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "This action could not be completed. Please try again.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function showPreview(action: () => Promise<RecoveryPreview>) {
    const previous = preview;
    setPreview(null);
    setConfirmed(false);
    await act(async () => {
      if (previous) await api.recovery.cancelPreview(previous.id);
      setPreview(await action());
    });
  }
  return (
    <div className="easy-page space-y-8">
      <div>
        <p className="text-teal-300 font-medium">Backups & recovery</p>
        <h1 className="text-3xl font-semibold mt-3">
          A way back when you need it.
        </h1>
        <p className="mt-4 text-slate-300">
          Player2 saves an automatic backup once a day while running and keeps
          the latest 14 backups. These copies live on this computer. Download
          one to a trusted location for protection if the computer is lost or
          fails.
        </p>
      </div>
      {error && (
        <p role="alert" className="text-rose-300">
          {error}
        </p>
      )}
      {message && (
        <p role="status" className="text-teal-200">
          {message}{" "}
          <Link to="/focus" className="underline">
            View your tasks
          </Link>
        </p>
      )}
      {status?.lastError && (
        <p role="alert" className="text-amber-200">
          {status.lastError}
        </p>
      )}
      <fieldset disabled={busy} className="space-y-8 min-w-0">
        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Saved backups</h2>
          <div className="flex gap-3 flex-wrap">
            <button
              className="easy-button"
              onClick={() =>
                void act(async () => {
                  await api.recovery.create();
                  setMessage("Backup saved.");
                  await load();
                })
              }
            >
              Back up now
            </button>
            <a
              className="easy-secondary"
              href="/api/backup"
              download="player2-backup.db"
            >
              Download current backup
            </a>
          </div>
          {!status ? (
            <button className="underline" onClick={() => void load()}>
              Load backups
            </button>
          ) : !status.backups.length ? (
            <p className="text-slate-400">
              Your first automatic backup will appear here shortly. You can also
              save one now.
            </p>
          ) : (
            <ul className="divide-y divide-slate-800">
              {status.backups.map((b) => (
                <li
                  key={b.id}
                  className="py-4 flex flex-wrap items-center justify-between gap-3"
                >
                  <div>
                    <p>{new Date(b.createdAt).toLocaleString()}</p>
                    <p className="text-sm text-slate-400">
                      {b.reason === "before-restore"
                        ? "Safety copy before restore"
                        : b.reason === "automatic"
                          ? "Automatic backup"
                          : "Saved by you"}{" "}
                      · {Math.max(1, Math.round(b.bytes / 1024))} KB
                    </p>
                  </div>
                  <div className="flex gap-3">
                    <a
                      className="underline text-teal-300"
                      href={`/api/recovery/backups/${b.id}`}
                      download
                    >
                      Download
                    </a>
                    <button
                      className="easy-secondary"
                      onClick={() =>
                        void showPreview(() => api.recovery.previewSaved(b.id))
                      }
                    >
                      Preview restore
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
        <section className="space-y-4 border-t border-slate-800 pt-6">
          <h2 className="text-xl font-semibold">Restore a downloaded backup</h2>
          <p className="text-slate-300">
            Choose a Player2 backup up to 32 MB. You will review its boards and
            task count before restoring.
          </p>
          <input
            type="file"
            accept=".db"
            aria-label="Choose a Player2 backup"
            onChange={(e) => {
              const file = e.target.files?.[0];
              e.target.value = "";
              if (file) {
                if (file.size > 32 * 1024 * 1024) {
                  setConfirmed(false);
                  setError("Choose a backup smaller than 32 MB.");
                  return;
                }
                void showPreview(() => api.recovery.preview(file));
              }
            }}
          />
          {preview && (
            <section
              aria-label="Restore preview"
              className="rounded-xl border border-teal-700 bg-slate-900 p-5 space-y-4"
            >
              <h3 className="text-lg font-semibold">
                {preview.boards.length} boards · {preview.tasks} tasks
              </h3>
              <ul className="list-disc pl-5 max-h-48 overflow-auto">
                {preview.boards.map((name, index) => (
                  <li key={index}>{name}</li>
                ))}
              </ul>
              <p className="text-sm text-slate-300">
                Restoring creates separate boards named “(restored)” and saves a
                safety backup first. Your current boards stay in place. GitHub
                connections and agent sessions are inactive in the copies;
                software tasks marked done return to Review for verification.
              </p>
              <label className="flex gap-3 items-start">
                <input
                  type="checkbox"
                  checked={confirmed}
                  onChange={(e) => setConfirmed(e.target.checked)}
                />
                I’ve reviewed this backup and want to create the recovered
                boards.
              </label>
              <div className="flex gap-3">
                <button
                  className="easy-button"
                  disabled={!confirmed || !preview.boards.length}
                  onClick={() =>
                    void act(async () => {
                      await api.recovery.restore(preview.id);
                      setPreview(null);
                      setConfirmed(false);
                      setMessage("Your recovered boards are ready.");
                      await load();
                    })
                  }
                >
                  Restore boards
                </button>
                <button
                  className="easy-secondary"
                  onClick={() =>
                    void act(async () => {
                      await api.recovery.cancelPreview(preview.id);
                      setPreview(null);
                      setConfirmed(false);
                    })
                  }
                >
                  Cancel
                </button>
              </div>
            </section>
          )}
        </section>
        <section className="space-y-4 border-t border-slate-800 pt-6">
          <h2 className="text-xl font-semibold">Deleted tasks</h2>
          <p className="text-slate-400">
            Restore a task with its notes and checklist. For linked software
            tasks, review any GitHub conflict before syncing. The latest 200
            deleted tasks are shown.
          </p>
          {trash.length ? (
            <ul className="divide-y divide-slate-800">
              {trash.map((t) => (
                <li
                  key={t.id}
                  className="flex justify-between items-center gap-4 py-4"
                >
                  <div className="min-w-0">
                    <p className="break-words">{t.title}</p>
                    <p className="text-sm text-slate-400">
                      {t.board} · {new Date(t.deletedAt).toLocaleDateString()}
                    </p>
                  </div>
                  <button
                    className="easy-secondary"
                    onClick={() =>
                      void act(async () => {
                        await api.recovery.undo(t.id, t.revision);
                        setMessage(
                          "Task restored with its notes and checklist.",
                        );
                        await load();
                      })
                    }
                  >
                    Restore task
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p>No deleted tasks.</p>
          )}
        </section>
      </fieldset>
      {busy && <p role="status">Working on your recovery request…</p>}
    </div>
  );
}
