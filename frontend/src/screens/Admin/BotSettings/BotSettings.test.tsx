import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../../test/render";
import { BotSettings } from "./BotSettings";

function mockFetchGet(body: unknown, status = 200) {
  return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    const method = init?.method ?? "GET";
    // LLM info endpoints are unrelated to the BotConfig fixture — answer
    // them with empty stubs so the page renders without errors.
    if (typeof url === "string" && url.includes("/api/llm/")) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () =>
          url.includes("/providers")
            ? Promise.resolve({ providers: [] })
            : Promise.resolve({ models: [] }),
      } as unknown as Response);
    }
    if (method === "GET") {
      return Promise.resolve({
        ok: status >= 200 && status < 300,
        status,
        json: () => Promise.resolve(body),
      } as unknown as Response);
    }
    // PUT — return same body as success
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(body),
    } as unknown as Response);
  });
}

function mockFetchSequence(
  responses: Array<{
    ok: boolean;
    status: number;
    body: unknown;
  }>
) {
  // GETs to /api/llm/* always return empty stubs; the queue applies only to
  // /api/admin/bot-config calls (GET, then PUT, in order).
  const queue = [...responses];
  return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    if (typeof url === "string" && url.includes("/api/llm/")) {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () =>
          url.includes("/providers")
            ? Promise.resolve({ providers: [] })
            : Promise.resolve({ models: [] }),
      } as unknown as Response);
    }
    const next = queue.shift();
    if (!next) {
      return Promise.reject(new Error("unexpected fetch — queue empty"));
    }
    void init;
    return Promise.resolve({
      ok: next.ok,
      status: next.status,
      json: () => Promise.resolve(next.body),
    } as unknown as Response);
  });
}

describe("BotSettings", () => {
  beforeEach(() => {
    localStorage.clear();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads current configuration on mount and seeds the form", async () => {
    const fetchSpy = mockFetchGet({
      configured: true,
      masked_token: "1234567:****abcd",
      allowlist_user_ids: [42, 100],
      log_level: "warn",
    });
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<BotSettings />);

    // Allowlist textarea seeded from existing config (wait for the
    // post-fetch useEffect to flush).
    const allowlist = (await screen.findByLabelText(
      /Allowlist/i
    )) as HTMLTextAreaElement;
    await waitFor(() => {
      expect(allowlist.value).toContain("42");
      expect(allowlist.value).toContain("100");
    });

    // Log level select seeded.
    const logLevel = (await screen.findByLabelText(/Log level/i)) as HTMLSelectElement;
    await waitFor(() => {
      expect(logLevel.value).toBe("warn");
    });

    // Mask of current token surfaced as read-only hint.
    expect(screen.getByText(/1234567:\*\*\*\*abcd/)).toBeInTheDocument();

    // Token input stays empty so "keep current" is the default action.
    const token = (await screen.findByLabelText(
      /Telegram bot token/i
    )) as HTMLInputElement;
    expect(token.value).toBe("");
  });

  it("hides the bot token by default and toggles to visible on demand", async () => {
    const fetchSpy = mockFetchGet({ configured: false });
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();

    renderWithProviders(<BotSettings />);
    const token = (await screen.findByLabelText(
      /Telegram bot token/i
    )) as HTMLInputElement;
    expect(token.type).toBe("password");

    await user.click(screen.getByRole("button", { name: /show token/i }));
    expect(token.type).toBe("text");

    await user.click(screen.getByRole("button", { name: /hide token/i }));
    expect(token.type).toBe("password");
  });

  it("submits a PUT with the parsed allowlist and shows the success banner", async () => {
    const seed = {
      configured: true,
      masked_token: "old:****",
      allowlist_user_ids: [],
      log_level: "info",
    };
    const fetchSpy = mockFetchSequence([
      { ok: true, status: 200, body: seed },
      {
        ok: true,
        status: 200,
        body: { ...seed, allowlist_user_ids: [42, 100] },
      },
    ]);
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();

    renderWithProviders(<BotSettings />);
    await screen.findByLabelText(/Telegram bot token/i);

    const allowlist = screen.getByLabelText(/Allowlist/i) as HTMLTextAreaElement;
    await user.clear(allowlist);
    await user.type(allowlist, "42, 100");

    await user.click(
      screen.getByRole("button", { name: /Save configuration/i })
    );

    await screen.findByRole("status");
    expect(screen.getByRole("status").textContent).toMatch(/Saved/i);

    // Inspect the PUT call.
    const putCall = fetchSpy.mock.calls.find(
      (c) => (c[1] as RequestInit)?.method === "PUT"
    );
    expect(putCall).toBeDefined();
    const body = JSON.parse((putCall![1] as RequestInit).body as string) as {
      allowlist_user_ids: number[];
      log_level: string;
    };
    expect(body.allowlist_user_ids).toEqual([42, 100]);
    expect(body.log_level).toBe("info");
  });

  it("highlights the offending field when the backend returns 400 with `field`", async () => {
    const fetchSpy = mockFetchSequence([
      { ok: true, status: 200, body: { configured: false } },
      {
        ok: false,
        status: 400,
        body: {
          error: "bot token must be in the form <digits>:<secret> (Telegram format)",
          field: "token",
        },
      },
    ]);
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();

    renderWithProviders(<BotSettings />);
    await screen.findByLabelText(/Telegram bot token/i);

    const token = screen.getByLabelText(/Telegram bot token/i) as HTMLInputElement;
    await user.type(token, "garbage");
    await user.click(
      screen.getByRole("button", { name: /Save configuration/i })
    );

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toMatch(/Telegram format/);
  });

  it("rejects a non-numeric allowlist entry client-side before hitting the backend", async () => {
    const fetchSpy = mockFetchGet({ configured: false });
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();

    renderWithProviders(<BotSettings />);
    await screen.findByLabelText(/Telegram bot token/i);

    const allowlist = screen.getByLabelText(/Allowlist/i) as HTMLTextAreaElement;
    await user.clear(allowlist);
    await user.type(allowlist, "42, banana, 100");

    await user.click(
      screen.getByRole("button", { name: /Save configuration/i })
    );

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toMatch(/banana/);

    // No PUT should have been issued.
    const putCalls = fetchSpy.mock.calls.filter(
      (c) => (c[1] as RequestInit | undefined)?.method === "PUT"
    );
    expect(putCalls).toHaveLength(0);
  });

  it("shows a generic error banner when the GET fails with non-2xx", async () => {
    const fetchSpy = mockFetchSequence([
      { ok: false, status: 500, body: {} },
    ]);
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<BotSettings />);

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((a) => /HTTP 500/.test(a.textContent ?? ""))).toBe(true);
    });
  });
});
