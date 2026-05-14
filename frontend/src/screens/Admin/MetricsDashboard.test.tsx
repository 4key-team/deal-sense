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
    vi.useFakeTimers();
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

    // Drain the mount-effect Promise so React applies state.
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledTimes(1);
    });

    expect(screen.getByText(/Requests/i)).toBeInTheDocument();
    expect(screen.getByText(/LLM/i)).toBeInTheDocument();
    expect(screen.getByText(/Endpoint risk/i)).toBeInTheDocument();
    expect(screen.getByText(/Security declines/i)).toBeInTheDocument();

    // A scrape value from the fixture must appear somewhere in the DOM —
    // confirms the parser hooked up to the rendered counters.
    expect(screen.getByText(/allowlist/i)).toBeInTheDocument();
  });

  it("re-fetches every interval ms (default 5s, picker switches to 1s)", async () => {
    const fetchSpy = mockFetchOk(SAMPLE);
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<MetricsDashboard />);

    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });
    expect(fetchSpy).toHaveBeenCalledTimes(1);

    // Advance 5s — the default interval — and confirm a second fetch.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(fetchSpy).toHaveBeenCalledTimes(2);

    // Switch to 1s.
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const picker = await screen.findByLabelText(/Refresh/i);
    await user.selectOptions(picker, "1000");

    // After switching, the prior 5s timer is replaced; 3 ticks of 1s → 3 more fetches.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000);
    });
    expect(fetchSpy).toHaveBeenCalledTimes(5);
  });

  it("renders an error state when fetch fails", async () => {
    const fetchSpy = vi.fn().mockRejectedValue(new Error("network down"));
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<MetricsDashboard />);

    await waitFor(() => {
      expect(screen.getByText(/network down|Не удалось/i)).toBeInTheDocument();
    });
  });

  it("sends X-API-Key header when localStorage has ds:apiKey", async () => {
    localStorage.setItem("ds:apiKey", JSON.stringify("secret-123"));
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
