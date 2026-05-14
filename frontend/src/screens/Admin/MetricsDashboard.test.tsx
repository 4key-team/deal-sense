import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { MetricsDashboard } from "./MetricsDashboard";

const SAMPLE = `# HELP dealsense_requests_total Total HTTP requests.
# TYPE dealsense_requests_total counter
dealsense_requests_total{path="/api/tender/analyze",status="200"} 5
dealsense_requests_total{path="/api/llm/check",status="500"} 1
# HELP dealsense_llm_calls_total LLM calls.
# TYPE dealsense_llm_calls_total counter
dealsense_llm_calls_total{provider="anthropic",status="ok"} 3
dealsense_llm_calls_total{provider="anthropic",status="error"} 1
# HELP dealsense_endpoint_risk Endpoint risk.
# TYPE dealsense_endpoint_risk gauge
dealsense_endpoint_risk{path="/api/tender/analyze",level="modify"} 1
dealsense_endpoint_risk{path="/healthz",level="safe_read"} 1
# HELP dealsense_security_decline_total Declines.
# TYPE dealsense_security_decline_total counter
dealsense_security_decline_total{kind="allowlist"} 4
dealsense_security_decline_total{kind="api_key"} 2
`;

function mockFetchOk(body: string) {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    text: () => Promise.resolve(body),
  } as unknown as Response);
}

describe("MetricsDashboard", () => {
  beforeEach(() => {
    localStorage.clear();
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("fetches /metrics on mount and renders the four sections", async () => {
    const fetchSpy = mockFetchOk(SAMPLE);
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<MetricsDashboard />);

    // findByText awaits the async fetch + state update for every heading.
    expect(await screen.findByText(/Requests by path/i)).toBeInTheDocument();
    expect(await screen.findByText(/LLM calls/i)).toBeInTheDocument();
    expect(await screen.findByText(/Endpoint risk/i)).toBeInTheDocument();
    expect(await screen.findByText(/Security declines/i)).toBeInTheDocument();

    // A scrape value from the fixture must appear somewhere in the DOM —
    // confirms the parser hooked up to the rendered counters.
    expect(await screen.findByText(/allowlist/i)).toBeInTheDocument();
    expect(fetchSpy).toHaveBeenCalled();
  });

  it("re-fetches when the interval picker switches to a shorter cadence", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const fetchSpy = mockFetchOk(SAMPLE);
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<MetricsDashboard />);

    // Wait for the mount fetch.
    await waitFor(() => expect(fetchSpy).toHaveBeenCalledTimes(1));

    // Switch from the default 5s to 1s.
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const picker = await screen.findByLabelText(/Refresh/i);
    await user.selectOptions(picker, "1000");

    // The hook restarts the interval; expect at least one more fetch after
    // a second of fake time.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1100);
    });
    expect(fetchSpy.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it("renders an error state when fetch fails", async () => {
    const fetchSpy = vi.fn().mockRejectedValue(new Error("network down"));
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<MetricsDashboard />);

    await waitFor(() => {
      expect(screen.getByText(/network down|Не удалось/i)).toBeInTheDocument();
    });
  });

  it("sends X-API-Key header when localStorage has the deal-sense key", async () => {
    // Reuses the existing ds:llm-settings shape so the dashboard piggybacks
    // on the same setting the rest of the app reads from (see lib/api.ts).
    localStorage.setItem(
      "ds:llm-settings",
      JSON.stringify({ dealSenseApiKey: "secret-123" }),
    );
    const fetchSpy = mockFetchOk(SAMPLE);
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<MetricsDashboard />);

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalled();
    });

    const init = fetchSpy.mock.calls[0][1] as RequestInit | undefined;
    const headers = init?.headers as Record<string, string> | undefined;
    expect(headers?.["X-API-Key"]).toBe("secret-123");
  });
});
