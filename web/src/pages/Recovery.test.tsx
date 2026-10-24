import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import Recovery from "./Recovery";
import { api } from "../api/client";
vi.mock("../api/client", () => ({
  api: {
    recovery: {
      status: vi.fn(),
      trash: vi.fn(),
      create: vi.fn(),
      preview: vi.fn(),
      previewSaved: vi.fn(),
      restore: vi.fn(),
      undo: vi.fn(),
      cancelPreview: vi.fn(),
    },
  },
}));
beforeEach(() => {
  vi.mocked(api.recovery.status).mockResolvedValue({
    directory: "/private",
    backups: [
      {
        id: "saved.db",
        reason: "automatic",
        createdAt: "2026-10-23T12:00:00Z",
        bytes: 1000,
      },
    ],
  });
  vi.mocked(api.recovery.trash).mockResolvedValue([]);
});
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});
it("requires reviewing a preview and preserves it after a failed restore so retries reuse the same ID", async () => {
  vi.mocked(api.recovery.previewSaved).mockResolvedValue({
    id: "preview-1",
    boards: ["Household"],
    tasks: 3,
  });
  vi.mocked(api.recovery.restore)
    .mockRejectedValueOnce(new Error("Safety backup failed"))
    .mockResolvedValueOnce({ projectIds: ["restored"] });
  render(
    <MemoryRouter>
      <Recovery />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole("button", { name: "Preview restore" }),
  );
  expect(await screen.findByText("1 boards · 3 tasks")).toBeTruthy();
  await userEvent.click(screen.getByRole("button", { name: "Restore boards" }));
  expect(api.recovery.restore).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("checkbox"));
  await userEvent.click(screen.getByRole("button", { name: "Restore boards" }));
  expect(await screen.findByRole("alert")).toHaveProperty(
    "textContent",
    "Safety backup failed",
  );
  expect(screen.getByRole("region", { name: "Restore preview" })).toBeTruthy();
  await userEvent.click(screen.getByRole("button", { name: "Restore boards" }));
  expect(vi.mocked(api.recovery.restore).mock.calls).toEqual([
    ["preview-1"],
    ["preview-1"],
  ]);
  expect(
    await screen.findByText(/Your recovered boards are ready/),
  ).toBeTruthy();
});
it("keeps failed task recovery visible and cancels unwanted restore previews", async () => {
  vi.mocked(api.recovery.trash).mockResolvedValue([
    {
      id: "task",
      title: "My notes",
      board: "Home",
      revision: 3,
      deletedAt: "2026-10-23T12:00:00Z",
      linked: false,
    },
  ]);
  vi.mocked(api.recovery.undo).mockRejectedValue(
    new Error("Try again after sync"),
  );
  vi.mocked(api.recovery.previewSaved).mockResolvedValue({
    id: "preview-2",
    boards: ["Home"],
    tasks: 1,
  });
  vi.mocked(api.recovery.cancelPreview).mockResolvedValue();
  render(
    <MemoryRouter>
      <Recovery />
    </MemoryRouter>,
  );
  await userEvent.click(
    await screen.findByRole("button", { name: "Restore task" }),
  );
  expect(await screen.findByRole("alert")).toHaveProperty(
    "textContent",
    "Try again after sync",
  );
  expect(screen.getByText("My notes")).toBeTruthy();
  await userEvent.click(
    screen.getByRole("button", { name: "Preview restore" }),
  );
  await userEvent.click(await screen.findByRole("button", { name: "Cancel" }));
  expect(api.recovery.cancelPreview).toHaveBeenCalledWith("preview-2");
  expect(screen.queryByRole("region", { name: "Restore preview" })).toBeNull();
});
