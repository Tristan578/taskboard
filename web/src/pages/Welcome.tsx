import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowRight, Sparkles, Check } from "lucide-react";
import { api } from "../api/client";
import { starters, parseTaskLines, planPrompt } from "../lib/plans";
import CopyText from "../components/CopyText";

export default function Welcome() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [name, setName] = useState("");
  const [goal, setGoal] = useState("");
  const [policy, setPolicy] = useState<"manual" | "verified">("manual");
  const [taskText, setTaskText] = useState("");
  const [selected, setSelected] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [requestId] = useState(() => crypto.randomUUID());
  async function create() {
    setBusy(true);
    setError("");
    try {
      const project = await api.welcome.create({
        requestId,
        name: name.trim(),
        goal,
        completionPolicy: policy,
        tasks: parseTaskLines(taskText),
      });
      navigate(`/board?project=${encodeURIComponent(project.id)}`);
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : "Your board could not be saved. Please try again.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="easy-page">
      <p className="text-teal-300 text-sm font-semibold flex items-center gap-2">
        <Sparkles size={18} /> A little structure. A useful next step.
      </p>
      <p className="mt-7 text-sm text-slate-400">
        Step {step} of 2 · {step === 1 ? "Choose a goal" : "Make it yours"}
      </p>
      <h1 className="text-3xl sm:text-4xl font-semibold tracking-tight mt-3">
        {step === 1
          ? "What would you like help with?"
          : "Your first plan, ready to shape."}
      </h1>
      <p className="mt-4 text-slate-300 max-w-2xl leading-relaxed">
        {step === 1
          ? "Start with something that would make your work or life easier. No account, code, or AI connection is needed to create a board."
          : "These are suggested starting points. Keep what helps, change the rest, or paste a task list from your AI chat."}
      </p>
      <form
        className="mt-8 space-y-6"
        onSubmit={(e) => {
          e.preventDefault();
          if (step === 1) {
            setStep(2);
            setError("");
          } else {
            void create();
          }
        }}
      >
        {step === 1 ? (
          <>
            <div className="grid sm:grid-cols-2 gap-2">
              {starters.map((s) => (
                <button
                  type="button"
                  key={s.id}
                  aria-pressed={selected === s.id}
                  className={`text-left rounded-xl p-4 border transition-colors ${selected === s.id ? "border-teal-400 bg-teal-400/10" : "border-slate-700 hover:border-slate-400"}`}
                  onClick={() => {
                    setSelected(s.id);
                    setName(s.name);
                    setGoal(s.goal);
                    setPolicy(s.policy);
                    setTaskText(s.tasks.join("\n"));
                  }}
                >
                  {selected === s.id && (
                    <Check className="inline mr-2 text-teal-300" size={16} />
                  )}{" "}
                  {s.name}
                </button>
              ))}
            </div>
            <label className="easy-label">
              Name your board
              <input
                className="easy-input"
                required
                maxLength={120}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My next chapter, Home base, First app…"
              />
            </label>
            <label className="easy-label">
              What would a useful result look like?
              <textarea
                className="easy-input"
                rows={3}
                maxLength={4000}
                value={goal}
                onChange={(e) => setGoal(e.target.value)}
                placeholder="I want a simple way to…"
              />
            </label>
            <fieldset className="space-y-3">
              <legend className="easy-label mb-2">
                How will you know the work is done?
              </legend>
              <label className="flex gap-3 items-start">
                <input
                  className="mt-1"
                  type="radio"
                  name="completion"
                  checked={policy === "manual"}
                  onChange={() => setPolicy("manual")}
                />
                <span>
                  I review the result{" "}
                  <span className="block text-sm text-slate-400">
                    You review the result and mark it done.
                  </span>
                </span>
              </label>
              <label className="flex gap-3 items-start">
                <input
                  className="mt-1"
                  type="radio"
                  name="completion"
                  checked={policy === "verified"}
                  onChange={() => setPolicy("verified")}
                />
                <span>
                  I’m shipping software with GitHub{" "}
                  <span className="block text-sm text-slate-400">
                    Done requires a verified GitHub merge or deployment. You can
                    connect GitHub later.
                  </span>
                </span>
              </label>
            </fieldset>
            <button className="easy-button" disabled={!name.trim()}>
              Shape my plan <ArrowRight size={17} />
            </button>
          </>
        ) : (
          <>
            <label className="easy-label">
              Tasks to add · one per line
              <textarea
                className="easy-input"
                rows={8}
                value={taskText}
                onChange={(e) => setTaskText(e.target.value)}
                placeholder="Write a small first step…"
              />
            </label>
            <p className="text-sm text-slate-400">
              {parseTaskLines(taskText).length} tasks · You can start with an
              empty board. Nothing runs automatically.
            </p>
            <details className="border-y border-slate-800 py-5">
              <summary className="cursor-pointer font-medium">
                Want help from the AI chat you already use?
              </summary>
              <div className="pt-4 space-y-4">
                <p className="text-sm text-slate-300">
                  Copy these instructions, paste them into your AI chat, and
                  bring its task list back above. Review the text before sharing
                  personal details. Your chat does not automatically update this
                  board.
                </p>
                <CopyText text={planPrompt(goal || name)} />
                <p className="text-sm text-slate-400">
                  Open your usual ChatGPT, Claude, Gemini, or other AI app. No
                  API key is needed for copy/paste.
                </p>
              </div>
            </details>
            {error && (
              <p role="alert" className="text-rose-300">
                {error}
              </p>
            )}
            <div className="flex gap-3">
              <button
                type="button"
                className="easy-secondary"
                disabled={busy}
                onClick={() => setStep(1)}
              >
                Back
              </button>
              <button
                className="easy-button"
                disabled={busy || parseTaskLines(taskText).length > 50}
              >
                {busy ? "Saving your board…" : "Create my board"}{" "}
                <ArrowRight size={17} />
              </button>
            </div>
          </>
        )}
      </form>
      <p className="mt-9 text-sm text-slate-400">
        Saved on this computer. No automatic cloud backup or phone sync.{" "}
        <Link to="/help" className="underline">
          How Player2 works
        </Link>
      </p>
    </div>
  );
}
