import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import SafeMarkdown from "./SafeMarkdown";
afterEach(cleanup);
it("does not silently request remote images from task notes", () => {
  render(
    <SafeMarkdown>
      {
        "![Tracking image](https://example.com/pixel)\n[Unsafe](javascript:alert(1))"
      }
    </SafeMarkdown>,
  );
  expect(screen.queryByRole("img")).toBeNull();
  expect(screen.getByText(/View image/).closest("a")).toHaveProperty(
    "href",
    "https://example.com/pixel",
  );
  expect(
    screen.getByText("Unsafe").closest("a")?.getAttribute("href"),
  ).not.toContain("javascript:");
});
