import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
export default function UndoDelete() {
  const generation = useRef(0);
  const [deleted, setDeleted] = useState<{
    id: string;
    revision: number;
  } | null>(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [restored, setRestored] = useState(false);
  useEffect(() => {
    const listener = (event: Event) => {
      generation.current++;
      setBusy(false);
      setDeleted((event as CustomEvent).detail);
      setError("");
      setRestored(false);
    };
    window.addEventListener("player2:deleted", listener);
    return () => window.removeEventListener("player2:deleted", listener);
  }, []);
  if (!deleted) return null;
  return (
    <aside
      aria-label="Deleted task recovery"
      className="fixed z-[60] bottom-5 left-5 right-5 sm:left-auto sm:w-96 p-4 rounded-xl border border-slate-600 bg-slate-900 shadow-xl space-y-3"
    >
      <p role="status">
        {restored ? "Task restored." : "Task moved to Deleted tasks."}
      </p>
      {error && (
        <p role="alert" className="text-rose-300">
          {error}
        </p>
      )}
      <div className="flex gap-3 items-center">
        {!restored && (
          <button
            className="easy-button"
            disabled={busy}
            onClick={async () => {
              const request = generation.current;
              setBusy(true);
              try {
                await api.recovery.undo(deleted.id, deleted.revision);
                if(request!==generation.current)return;
                setRestored(true);
                setError("");
              } catch (e) {
                if(request!==generation.current)return;
                setError(
                  e instanceof Error
                    ? e.message
                    : "Could not restore. Try Backups & recovery.",
                );
              } finally {
                if(request===generation.current)setBusy(false);
              }
            }}
          >
            {busy ? "Restoring…" : "Undo"}
          </button>
        )}
        <Link to={restored ? "/focus" : "/recovery"} className="underline">
          {restored ? "View tasks" : "Recovery"}
        </Link>
        <button
          className="text-slate-400"
          disabled={busy}
          onClick={() => setDeleted(null)}
        >
          Dismiss
        </button>
      </div>
    </aside>
  );
}
