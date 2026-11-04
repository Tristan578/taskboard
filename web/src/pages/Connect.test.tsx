import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import Connect from "./Connect";
import { api, type ConnectionPilot, type Ticket } from "../api/client";

vi.mock("../api/client", () => ({
  api: {
    connection: { setup: vi.fn(), create: vi.fn(), pilot: vi.fn() },
    tickets: { move: vi.fn() },
  },
}));
const id = "connect-00000000-0000-4000-8000-000000000001";
function pilot(state: ConnectionPilot["state"] = "waiting"): ConnectionPilot {
  return {
    id,
    projectId: "board",
    state,
    reported: state !== "waiting",
    prompt: "Use actual Player2 tools for task-1",
    task: {
      id: "task-1",
      title: "Plan one useful next step",
      description: "Three useful next steps",
      revision: 7,
      work:
        state === "waiting"
          ? undefined
          : {
              agent: "Test agent",
              sessionId: id,
              summary: "Plan saved",
              stale: false,
              heartbeatAt: "2026-10-24T12:00:00Z",
            },
    } as Ticket,
  };
}
function renderPage() {
  render(
    <MemoryRouter>
      <Connect />
    </MemoryRouter>,
  );
}
beforeEach(() => {
  localStorage.clear();
  vi.resetAllMocks();
  vi.mocked(api.connection.setup).mockResolvedValue({
    name: "player2-test",
    shell: "PowerShell",
    platform: "windows",
    codex: "codex mcp add player2-test -- 'app' mcp",
    claude: "claude mcp add player2-test -- 'app' mcp",
    config: { mcpServers: {} },
  });
  vi.mocked(api.connection.pilot).mockRejectedValue(
    new Error("No report available yet"),
  );
});
afterEach(cleanup);

it("requires an access acknowledgement and never calls copied settings a connection", async () => {
  vi.mocked(api.connection.create).mockResolvedValue(pilot());
  renderPage();
  expect(
    screen.queryByRole("button", { name: "Copy setup command" }),
  ).toBeNull();
  await userEvent.click(screen.getByRole("checkbox"));
  expect(
    await screen.findByRole("button", { name: "Copy setup command" }),
  ).toBeTruthy();
  await userEvent.type(
    screen.getByLabelText("What would you like help planning?"),
    "Plan my week",
  );
  await userEvent.click(
    screen.getByRole("button", { name: "Create my first task" }),
  );
  expect(
    await screen.findByText(/Waiting for your agent’s first report/),
  ).toBeTruthy();
  expect(
    screen.queryByRole("button", { name: /I reviewed the result/ }),
  ).toBeNull();
  expect(api.tickets.move).not.toHaveBeenCalled();
});

it("retains the goal and request ID when creation needs a retry", async () => {
  vi.mocked(api.connection.create)
    .mockRejectedValueOnce(new Error("Connection interrupted"))
    .mockResolvedValueOnce(pilot());
  renderPage();
  await userEvent.click(screen.getByRole("checkbox"));
  await userEvent.type(
    await screen.findByLabelText("What would you like help planning?"),
    "Prepare for an interview",
  );
  await userEvent.click(
    screen.getByRole("button", { name: "Create my first task" }),
  );
  expect(await screen.findByText("Connection interrupted")).toBeTruthy();
  expect(
    screen.getByLabelText("What would you like help planning?"),
  ).toHaveProperty("value", "Prepare for an interview");
  await userEvent.click(
    screen.getByRole("button", { name: "Create my first task" }),
  );
  await screen.findByText(/Waiting for your agent’s first report/);
  const calls = vi.mocked(api.connection.create).mock.calls;
  expect(calls[0]).toEqual(calls[1]);
  expect(localStorage.getItem("player2-connection-pilot")).toBe(calls[0][0]);
});

it("resumes an observed result and requires a revision-checked review action", async () => {
  localStorage.setItem("player2-connection-pilot", id);
  vi.mocked(api.connection.pilot).mockResolvedValue(pilot("needs_review"));
  vi.mocked(api.tickets.move).mockResolvedValue(pilot("completed").task);
  renderPage();
  expect(await screen.findByText("Three useful next steps")).toBeTruthy();
  expect(api.tickets.move).not.toHaveBeenCalled();
  vi.mocked(api.connection.pilot).mockResolvedValue(pilot("completed"));
  await userEvent.click(
    screen.getByRole("button", { name: /I reviewed the result/ }),
  );
  await waitFor(() =>
    expect(api.tickets.move).toHaveBeenCalledWith(
      "task-1",
      "done",
      undefined,
      7,
    ),
  );
  expect(await screen.findByText(/Task marked Done/)).toBeTruthy();
});

it("gives chat-only users a usable manual path without claiming automatic updates", async () => {
  renderPage();
  await userEvent.selectOptions(screen.getByLabelText("My AI app"), "chat");
  expect(
    screen.getByRole("link", { name: "Make a plan with my usual chat" }),
  ).toBeTruthy();
  expect(screen.queryByRole("checkbox")).toBeNull();
  expect(
    screen.queryByRole("button", { name: "Copy setup command" }),
  ).toBeNull();
  expect(api.connection.create).not.toHaveBeenCalled();
});
