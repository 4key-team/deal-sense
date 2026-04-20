import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { TenderReport } from "./TenderReport";
import * as api from "../../lib/api";

describe("TenderReport upload phase (RU)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("renders upload screen by default", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Отчёт по тендеру")).toBeInTheDocument();
    expect(screen.getByText("Стоит ли участвовать")).toBeInTheDocument();
    expect(screen.getByText("Загрузи тендерную документацию")).toBeInTheDocument();
  });

  it("disables analyze button when no files", () => {
    renderWithProviders(<TenderReport />);
    const btn = screen.getByRole("button", { name: /Анализировать/i });
    expect(btn).toBeDisabled();
  });

  it("enables analyze button after file upload", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);

    const input = screen.getByTestId("dropzone-input");
    const file = new File(["pdf"], "doc.pdf", { type: "application/pdf" });
    await user.upload(input, file);

    const btn = screen.getByRole("button", { name: /Анализировать/i });
    expect(btn).not.toBeDisabled();
  });

  it("does nothing if analyze clicked with no files", async () => {
    const analyzeSpy = vi.spyOn(api, "analyzeTender");
    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);

    // Force-click disabled button via fireEvent
    const btn = screen.getByRole("button", { name: /Анализировать/i });
    await user.click(btn);

    expect(analyzeSpy).not.toHaveBeenCalled();
    analyzeSpy.mockRestore();
  });
});

describe("TenderReport analyzing phase", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("shows spinner during analysis", async () => {
    let resolveAnalyze!: () => void;
    vi.spyOn(api, "analyzeTender").mockReturnValue(
      new Promise((resolve) => {
        resolveAnalyze = () => resolve({ verdict: "go", risk: "low", score: 82, summary: "ok" });
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);

    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));

    expect(screen.getByText("Анализирую документы…")).toBeInTheDocument();

    resolveAnalyze();
    vi.restoreAllMocks();
  });
});

describe("TenderReport result phase (RU, GO)", () => {
  beforeEach(() => {
    localStorage.setItem("ds:lang", "ru");
    vi.spyOn(api, "analyzeTender").mockResolvedValue({
      verdict: "go", risk: "low", score: 82, summary: "ok",
    });
  });

  afterEach(() => vi.restoreAllMocks());

  async function goToResult() {
    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));
    await waitFor(() => expect(screen.getByText("Идём")).toBeInTheDocument());
    return user;
  }

  it("renders verdict GO after analysis", async () => {
    await goToResult();
    expect(screen.getByText("82%", { exact: false })).toBeInTheDocument();
  });

  it("shows strengths and risks", async () => {
    await goToResult();
    expect(screen.getByText("Преимущества")).toBeInTheDocument();
    expect(screen.getByText("Риски")).toBeInTheDocument();
  });

  it("shows requirements grid", async () => {
    await goToResult();
    expect(screen.getByText("TypeScript, 3+ года")).toBeInTheDocument();
    expect(screen.getAllByText(/подтверждено/).length).toBeGreaterThan(0);
  });

  it("shows documents sidebar", async () => {
    await goToResult();
    expect(screen.getByText("tender-tz.pdf")).toBeInTheDocument();
  });

  it("shows effort card", async () => {
    await goToResult();
    expect(screen.getByText(/~6 часов/)).toBeInTheDocument();
  });

  it("shows GO action buttons", async () => {
    await goToResult();
    expect(screen.getByRole("button", { name: /Готовить заявку/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Сохранить отчёт/ })).toBeInTheDocument();
  });

  it("shows histogram and sparkline", async () => {
    await goToResult();
    expect(screen.getByText("63")).toBeInTheDocument();
    expect(screen.getByText("+37")).toBeInTheDocument();
  });

  it("exports report as JSON", async () => {
    const downloadSpy = vi.spyOn(api, "downloadBlob").mockImplementation(() => {});
    const user = await goToResult();
    await user.click(screen.getByRole("button", { name: /Сохранить отчёт/ }));
    expect(downloadSpy).toHaveBeenCalledOnce();
    downloadSpy.mockRestore();
  });

  it("navigates to /proposal on prep click", async () => {
    const user = await goToResult();
    await user.click(screen.getByRole("button", { name: /Готовить заявку/ }));
    expect(window.location.pathname).toBe("/proposal");
  });
});

describe("TenderReport NO-GO toggle", () => {
  beforeEach(() => {
    localStorage.setItem("ds:lang", "ru");
    vi.spyOn(api, "analyzeTender").mockResolvedValue({
      verdict: "go", risk: "low", score: 82, summary: "ok",
    });
  });

  afterEach(() => vi.restoreAllMocks());

  it("toggles to NO-GO and back", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));
    await waitFor(() => expect(screen.getByText("Идём")).toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "NO-GO" }));
    expect(screen.getByText("Пас")).toBeInTheDocument();
    expect(screen.getByText(/~14 часов/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Набросать отказ/ })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "GO" }));
    expect(screen.getByText("Идём")).toBeInTheDocument();
  });
});

describe("TenderReport error phase", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("shows error message on analysis failure", async () => {
    vi.spyOn(api, "analyzeTender").mockRejectedValue(new Error("Network error"));

    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));

    await waitFor(() => expect(screen.getByText("Не удалось проанализировать")).toBeInTheDocument());
    expect(screen.getByText("Network error")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Попробовать снова/ })).toBeInTheDocument();

    vi.restoreAllMocks();
  });

  it("handles non-Error rejection", async () => {
    vi.spyOn(api, "analyzeTender").mockRejectedValue("string error");

    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));

    await waitFor(() => expect(screen.getByText("string error")).toBeInTheDocument());

    vi.restoreAllMocks();
  });

  it("retries analysis from error state", async () => {
    const analyzeSpy = vi.spyOn(api, "analyzeTender")
      .mockRejectedValueOnce(new Error("fail"))
      .mockResolvedValueOnce({ verdict: "go", risk: "low", score: 82, summary: "ok" });

    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));

    await waitFor(() => expect(screen.getByText(/Попробовать снова/)).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /Попробовать снова/ }));

    await waitFor(() => expect(screen.getByText("Идём")).toBeInTheDocument());

    analyzeSpy.mockRestore();
  });
});

describe("TenderReport (EN)", () => {
  beforeEach(() => {
    localStorage.setItem("ds:lang", "en");
    vi.spyOn(api, "analyzeTender").mockResolvedValue({
      verdict: "go", risk: "low", score: 82, summary: "ok",
    });
  });

  afterEach(() => vi.restoreAllMocks());

  it("renders EN upload screen", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Tender report")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Analyze/i })).toBeDisabled();
  });

  it("renders EN result", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Analyze/i }));

    await waitFor(() => expect(screen.getByText("Go")).toBeInTheDocument());
    expect(screen.getByText("Strengths")).toBeInTheDocument();
    expect(screen.getByText("Risks")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Start preparing bid/ })).toBeInTheDocument();
  });
});
