import type { Ticket } from "../api/client";
export type FocusFilter = "all" | "due" | "review" | "blocked";
export function localDay(date = new Date()) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
export function dueDay(ticket: Ticket) {
  return ticket.dueDate?.slice(0, 10) || "";
}
export function focusReason(ticket: Ticket, today = localDay()) {
  const due = dueDay(ticket);
  if (due && due < today) return "Overdue";
  if (ticket.status === "review") return "Ready for your review";
  if (ticket.status === "blocked") return "Needs help";
  if (
    ticket.work?.stale &&
    ticket.work.agent &&
    ticket.status === "in_progress"
  )
    return "Agent update overdue";
  if (due === today) return "Due today";
  if (ticket.status === "in_progress") return "Continue where you left off";
  return "A next step";
}
export function focusTasks(
  tickets: Ticket[],
  filter: FocusFilter,
  query: string,
  projectId: string,
  today = localDay(),
) {
  const priority: Record<string, number> = {
    urgent: 0,
    high: 1,
    medium: 2,
    low: 3,
  };
  const rank = (t: Ticket) => {
    const d = dueDay(t);
    return d && d < today
      ? 0
      : t.status === "review"
        ? 1
        : t.status === "blocked"
          ? 2
          : t.status === "in_progress" && t.work?.agent && t.work.stale
            ? 3
            : d === today
              ? 4
              : t.status === "in_progress"
                ? 5
                : 6;
  };
  return tickets
    .filter(
      (t) =>
        t.status !== "done" &&
        !t.isDraft &&
        (!projectId || t.projectId === projectId),
    )
    .filter(
      (t) =>
        filter === "all" ||
        (filter === "due"
          ? !!dueDay(t) && dueDay(t) <= today
          : t.status === filter ||
            (filter === "blocked" &&
              t.status === "in_progress" &&
              !!t.work?.agent &&
              t.work.stale)),
    )
    .filter((t) =>
      `${t.title} ${t.description || ""} ${t.projectPrefix}-${t.number}`
        .toLocaleLowerCase()
        .includes(query.trim().toLocaleLowerCase()),
    )
    .sort(
      (a, b) =>
        rank(a) - rank(b) ||
        (priority[a.priority] ?? 2) - (priority[b.priority] ?? 2) ||
        (dueDay(a) || "9999").localeCompare(dueDay(b) || "9999") ||
        a.id.localeCompare(b.id),
    );
}
export function displayDue(value: string) {
  const [y, m, d] = value.slice(0, 10).split("-").map(Number);
  return new Date(y, m - 1, d).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}
