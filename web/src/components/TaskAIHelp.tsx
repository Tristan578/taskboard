import type { Ticket, Project } from "../api/client";
import CopyText from "./CopyText";
export default function TaskAIHelp({
  ticket,
  project,
}: {
  ticket: Ticket;
  project?: Project;
}) {
  const text = `Help me work on this task.\nGoal: ${project?.description || project?.name || "Complete the task below"}\nTask: ${ticket.title}\nNotes: ${ticket.description || "No additional notes yet."}\n\nAsk for missing information instead of guessing. Explain the next small step in plain language, then help me produce a useful result. Do not claim you have changed my board. I will paste the result into my task notes and review it myself. ${project?.completionPolicy === "manual" ? "I decide when the result is complete." : "For software, tests alone are not completion: the change needs to be merged or deployed."}`;
  return (
    <details className="border border-teal-900 rounded-xl p-4">
      <summary className="cursor-pointer font-medium text-teal-200">
        Ask my AI for help
      </summary>
      <div className="mt-4 space-y-3">
        <p className="text-sm text-slate-300">
          Copy this task into your usual AI chat. Review the text first. Bring
          useful answers back to Notes, then move the task to Review. This does
          not connect your chat automatically.
        </p>
        <CopyText text={text} />
      </div>
    </details>
  );
}
