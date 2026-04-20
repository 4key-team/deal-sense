import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  analyzeTender,
  generateProposal,
  checkConnection,
  listProviders,
  downloadBlob,
} from "./api";

const mockFetch = vi.fn();

beforeEach(() => {
  mockFetch.mockReset();
  vi.stubGlobal("fetch", mockFetch);
  localStorage.clear();
});

afterEach(() => {
  vi.restoreAllMocks();
});

function jsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
  });
}

describe("analyzeTender", () => {
  it("sends multipart with files and company_profile", async () => {
    mockFetch.mockReturnValueOnce(
      jsonResponse({ verdict: "go", risk: "low", score: 82, summary: "Good" }),
    );

    const file = new File(["pdf"], "doc.pdf", { type: "application/pdf" });
    const result = await analyzeTender([file], "Our Company");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toBe("http://localhost:8080/api/tender/analyze");
    expect(opts.method).toBe("POST");

    const form = opts.body as FormData;
    expect(form.get("company_profile")).toBe("Our Company");
    expect(form.getAll("files")).toHaveLength(1);

    expect(result.verdict).toBe("go");
    expect(result.score).toBe(82);
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockReturnValueOnce(
      Promise.resolve({ ok: false, status: 500 }),
    );

    await expect(
      analyzeTender([], "X"),
    ).rejects.toThrow("Tender analyze failed: 500");
  });

  it("sends LLM settings from localStorage", async () => {
    localStorage.setItem(
      "ds:llm-settings",
      JSON.stringify({
        providerId: "groq",
        apiKey: "sk-test-key",
        url: "https://api.groq.com/openai/v1",
        model: "llama-3.3-70b",
      }),
    );

    mockFetch.mockReturnValueOnce(
      jsonResponse({ verdict: "go", risk: "low", score: 50, summary: "" }),
    );

    await analyzeTender([], "Co");

    const headers = mockFetch.mock.calls[0][1].headers;
    expect(headers["X-LLM-Key"]).toBe("sk-test-key");
    expect(headers["X-LLM-Provider"]).toBe("groq");
    expect(headers["X-LLM-URL"]).toBe("https://api.groq.com/openai/v1");
    expect(headers["X-LLM-Model"]).toBe("llama-3.3-70b");
  });
});

describe("generateProposal", () => {
  it("sends template with context and returns JSON", async () => {
    mockFetch.mockReturnValueOnce(
      jsonResponse({ template: "tpl.docx", summary: "ok", sections: [], docx: true }),
    );

    const tpl = new File(["docx"], "tpl.docx");
    const ctx = [new File(["pdf"], "brief.pdf", { type: "application/pdf" })];
    const result = await generateProposal(tpl, ctx, "ru", { client: "Acme" });

    const form = mockFetch.mock.calls[0][1].body as FormData;
    expect(form.get("template")).toBeTruthy();
    expect(form.get("lang")).toBe("ru");
    expect(form.getAll("context")).toHaveLength(1);
    expect(form.get("params")).toBe('{"client":"Acme"}');

    expect(result.template).toBe("tpl.docx");
  });

  it("sends without params when omitted", async () => {
    mockFetch.mockReturnValueOnce(
      jsonResponse({ template: "t.docx", summary: "", sections: [], docx: false }),
    );

    const tpl = new File(["docx"], "tpl.docx");
    await generateProposal(tpl, []);

    const form = mockFetch.mock.calls[0][1].body as FormData;
    expect(form.get("params")).toBeNull();
  });

  it("throws on error", async () => {
    mockFetch.mockReturnValueOnce(
      Promise.resolve({ ok: false, status: 422 }),
    );

    await expect(
      generateProposal(new File(["x"], "t.docx"), []),
    ).rejects.toThrow("Proposal generate failed: 422");
  });
});

describe("checkConnection", () => {
  it("returns ok result", async () => {
    mockFetch.mockReturnValueOnce(
      jsonResponse({ ok: true, provider: "openai" }),
    );

    const result = await checkConnection();

    expect(result.ok).toBe(true);
    expect(result.provider).toBe("openai");

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toBe("http://localhost:8080/api/llm/check");
    expect(opts.method).toBe("POST");
  });

  it("returns error result without throwing", async () => {
    mockFetch.mockReturnValueOnce(
      jsonResponse({ ok: false, provider: "openai", error: "invalid key" }, 503),
    );

    const result = await checkConnection();
    expect(result.ok).toBe(false);
    expect(result.error).toBe("invalid key");
  });
});

describe("listProviders", () => {
  it("returns providers list", async () => {
    mockFetch.mockReturnValueOnce(
      jsonResponse({
        providers: [{ id: "openai", name: "OpenAI", models: ["gpt-4o"] }],
      }),
    );

    const result = await listProviders();

    expect(result.providers).toHaveLength(1);
    expect(result.providers[0].id).toBe("openai");

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toBe("http://localhost:8080/api/llm/providers");
    expect(opts.method).toBe("GET");
  });

  it("throws on error", async () => {
    mockFetch.mockReturnValueOnce(
      Promise.resolve({ ok: false, status: 500 }),
    );

    await expect(listProviders()).rejects.toThrow("List providers failed: 500");
  });
});

describe("downloadBlob", () => {
  it("creates and clicks a download link", () => {
    const createObjectURL = vi.fn(() => "blob:test");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL });

    const clickSpy = vi.fn();
    const createElement = vi.spyOn(document, "createElement").mockReturnValueOnce({
      set href(_: string) { /* noop */ },
      set download(_: string) { /* noop */ },
      click: clickSpy,
    } as unknown as HTMLAnchorElement);

    const blob = new Blob(["test"]);
    downloadBlob(blob, "output.docx");

    expect(createObjectURL).toHaveBeenCalledWith(blob);
    expect(clickSpy).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test");

    createElement.mockRestore();
  });
});
