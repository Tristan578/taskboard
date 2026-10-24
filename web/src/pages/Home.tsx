import { useEffect, useState } from "react";
import { Navigate } from "react-router-dom";
import { api } from "../api/client";
export default function Home() {
  const [target, setTarget] = useState("");
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    api.projects
      .list()
      .then((p) => {
        if (active) setTarget(p?.length ? "/focus" : "/welcome");
      })
      .catch(() => {
        if (active)
          setError(
            "We could not open your boards. Check that Player2 is running, then try again.",
          );
      });
    return () => {
      active = false;
    };
  }, []);
  if (target) return <Navigate to={target} replace />;
  return (
    <div className="easy-page">
      {error ? (
        <div role="alert">
          <p>{error}</p>
          <button
            className="easy-button mt-4"
            onClick={() => window.location.reload()}
          >
            Try again
          </button>
        </div>
      ) : (
        <p role="status">Opening your boards…</p>
      )}
    </div>
  );
}
