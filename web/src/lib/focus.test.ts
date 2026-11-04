import { expect, it } from "vitest";
import { focusTasks, focusReason, localDay } from "./focus";
import type { Ticket } from "../api/client";
const task = (id: string, extra: Partial<Ticket> = {}) =>
  ({
    id,
    title: id,
    projectId: "p",
    status: "todo",
    priority: "medium",
    description: "",
    isDraft: false,
    ...extra,
  }) as Ticket;
it("surfaces overdue work and review before new work while excluding finished and draft tasks", () => {
  const list = [
    task("future", { dueDate: "2026-10-24T00:00:00Z" }),
    task("review", { status: "review" }),
    task("overdue", { dueDate: "2026-10-22T00:00:00Z" }),
    task("done", { status: "done" }),
    task("draft", { isDraft: true }),
  ];
  expect(
    focusTasks(list, "all", "", "", "2026-10-23").map((t) => t.id),
  ).toEqual(["overdue", "review", "future"]);
  expect(list[0].id).toBe("future");
});
it("uses calendar due dates and can find stale agents, task notes and a specific board", () => {
  const due = task("due", {
    dueDate: "2026-10-23T00:00:00Z",
    description: "Plan meals",
  });
  const stale = task("stale", {
    status: "in_progress",
    work: { agent: "Codex", sessionId: "s", summary: "Working", stale: true },
  });
  expect(localDay(new Date(2026, 9, 23, 23, 59))).toBe("2026-10-23");
  expect(focusReason(due, "2026-10-23")).toBe("Due today");
  expect(focusTasks([due], "due", "MEALS", "p", "2026-10-23")).toHaveLength(1);
  expect(focusTasks([stale], "blocked", "", "", "2026-10-23")).toHaveLength(1);
  expect(focusReason(stale, "2026-10-23")).toBe("Agent update overdue");
  expect(focusTasks([due], "all", "", "other", "2026-10-23")).toHaveLength(0);
});
