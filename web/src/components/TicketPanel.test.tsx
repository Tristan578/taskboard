import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import TicketPanel from "./TicketPanel";
import { api, ApiError, type Ticket } from "../api/client";
vi.mock("./WorkEvidence", () => ({ default: () => null }));
vi.mock("../api/client", async (original) => {
  const actual = await original<typeof import("../api/client")>();
  return {
    ...actual,
    api: { ...actual.api, tickets: { ...actual.api.tickets, get: vi.fn() } },
  };
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
const ticket: Ticket = {
  id: "t1",
  revision: 7,
  projectId: "p1",
  title: "Original",
  description: "Saved notes",
  status: "todo",
  priority: "medium",
  dueDate: "2026-10-23T00:00:00Z",
  subtasks: [],
  number: 1,
  position: 1000,
  isDraft: false,
  createdAt: "2026-10-23T00:00:00Z",
  updatedAt: "2026-10-23T00:00:00Z",
  projectPrefix: "HOME",
  labels: [],
  blockedBy: [],
};
it("preserves a draft on conflict and requires comparison before using the newer revision", async () => {
  const update = vi
    .fn()
    .mockRejectedValueOnce(
      new ApiError("This task changed since you opened it.", 409),
    )
    .mockResolvedValueOnce({ ...ticket, title: "My draft", revision: 9 });
  vi.mocked(api.tickets.get).mockResolvedValue({
    ...ticket,
    title: "Agent edit",
    revision: 8,
  });
  render(
    <TicketPanel
      ticket={ticket}
      projects={[]}
      teams={[]}
      onClose={vi.fn()}
      onUpdate={update}
      onDelete={vi.fn()}
    />,
  );
  await userEvent.clear(screen.getByLabelText("Task title"));
  await userEvent.type(screen.getByLabelText("Task title"), "My draft");
  await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  expect(await screen.findByRole("alert")).toHaveProperty(
    "textContent",
    expect.stringContaining("changed"),
  );
  expect(screen.getByLabelText("Task title")).toHaveProperty(
    "value",
    "My draft",
  );
  expect(update.mock.calls[0][1].expectedRevision).toBe(7);
  await userEvent.click(
    screen.getByRole("button", { name: "View latest saved version" }),
  );
  expect(await screen.findByText("Agent edit")).toBeTruthy();
  await userEvent.click(
    screen.getByRole("button", { name: /I’ve compared them/ }),
  );
  await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  expect(update.mock.calls[1][1].expectedRevision).toBe(8);
  expect(
    screen.queryByText(
      "Unsaved changes. Save your notes before completing this task.",
    ),
  ).toBeNull();
});
it("warns before closing unsaved work and sends an explicit date clear", async () => {
  const close = vi.fn();
  const update = vi
    .fn()
    .mockResolvedValue({ ...ticket, revision: 8, dueDate: undefined });
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
  render(
    <TicketPanel
      ticket={ticket}
      projects={[]}
      teams={[]}
      onClose={close}
      onUpdate={update}
      onDelete={vi.fn()}
    />,
  );
  expect(screen.getByLabelText("Due date")).toHaveProperty(
    "value",
    "2026-10-23",
  );
  await userEvent.clear(screen.getByLabelText("Due date"));
  await userEvent.click(screen.getByRole("button", { name: "Close task" }));
  expect(confirm).toHaveBeenCalled();
  expect(close).not.toHaveBeenCalled();
  const event = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  expect(event.defaultPrevented).toBe(true);
  await userEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  expect(update.mock.calls[0][1].dueDate).toBe("");
});
it("keeps the panel open when deletion fails", async () => {
  vi.spyOn(window, "confirm").mockReturnValue(true);
  const close = vi.fn();
  const remove = vi.fn().mockRejectedValue(new Error("Could not delete"));
  render(
    <TicketPanel
      ticket={ticket}
      projects={[]}
      teams={[]}
      onClose={close}
      onUpdate={vi.fn()}
      onDelete={remove}
    />,
  );
  await userEvent.click(screen.getByRole("button", { name: "Delete task" }));
  expect(await screen.findByRole("alert")).toHaveProperty(
    "textContent",
    "Could not delete",
  );
  expect(close).not.toHaveBeenCalled();
  expect(remove).toHaveBeenCalledWith("t1", 7);
});
