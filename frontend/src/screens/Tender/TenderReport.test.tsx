import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { TenderReport } from "./TenderReport";
import * as api from "../../lib/api";
import type { TenderResult } from "../../lib/api";

const MOCK_RESULT: TenderResult = {
  verdict: "go",
  risk: "low",
  score: 82,
  summary: "Хорошее соответствие",
  pros: [
    { title: "Стек совпадает", desc: "TypeScript, React, Go" },
    { title: "Бюджет адекватный", desc: "8.5 млн покрывает расходы" },
  ],
  cons: [
    { title: "Нужен ISO 27001", desc: "Сертификата нет" },
  ],
  requirements: [
    { label: "TypeScript", status: "met" },
    { label: "React", status: "met" },
    { label: "ISO 27001", status: "miss" },
    { label: "Опыт 1С", status: "partial" },
  ],
  effort: "~120 часов",
};

describe("TenderReport upload phase (RU)", () => {
  beforeEach(() => { localStorage.clear(); localStorage.setItem("ds:lang", "ru"); });

  it("renders upload screen by default", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Отчёт по тендеру")).toBeInTheDocument();
    expect(screen.getByText("Стоит ли участвовать")).toBeInTheDocument();
    expect(screen.getByText("Загрузите тендерную документацию")).toBeInTheDocument();
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
  beforeEach(() => { localStorage.clear(); localStorage.setItem("ds:lang", "ru"); });

  it("shows spinner during analysis", async () => {
    let resolveAnalyze!: () => void;
    vi.spyOn(api, "analyzeTender").mockReturnValue(
      new Promise((resolve) => {
        resolveAnalyze = () => resolve(MOCK_RESULT);
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
    localStorage.clear();
    localStorage.setItem("ds:lang", "ru");
    vi.spyOn(api, "analyzeTender").mockResolvedValue(MOCK_RESULT);
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
    expect(screen.getByText("TypeScript")).toBeInTheDocument();
    expect(screen.getAllByText(/подтверждено/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/не закрыто/).length).toBeGreaterThan(0);
  });

  it("shows uploaded file in sidebar", async () => {
    await goToResult();
    expect(screen.getByText("doc.pdf")).toBeInTheDocument();
  });

  it("shows effort from API", async () => {
    await goToResult();
    expect(screen.getByText("~120 часов")).toBeInTheDocument();
  });

  it("shows GO action buttons", async () => {
    await goToResult();
    expect(screen.getByRole("button", { name: /Готовить заявку/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Сохранить отчёт/ })).toBeInTheDocument();
  });

  it("shows pros and cons from API", async () => {
    await goToResult();
    expect(screen.getByText("Стек совпадает")).toBeInTheDocument();
    expect(screen.getByText("Нужен ISO 27001")).toBeInTheDocument();
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

describe("TenderReport NO-GO result", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("ds:lang", "ru");
    vi.spyOn(api, "analyzeTender").mockResolvedValue({
      ...MOCK_RESULT,
      verdict: "no-go",
      risk: "high",
      score: 35,
    });
  });

  afterEach(() => vi.restoreAllMocks());

  it("shows NO-GO verdict from API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TenderReport />);
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, new File(["pdf"], "doc.pdf", { type: "application/pdf" }));
    await user.click(screen.getByRole("button", { name: /Анализировать/i }));
    await waitFor(() => expect(screen.getByText("Пас")).toBeInTheDocument());
    expect(screen.getByText("35%", { exact: false })).toBeInTheDocument();
  });
});

describe("TenderReport error phase", () => {
  beforeEach(() => { localStorage.clear(); localStorage.setItem("ds:lang", "ru"); });

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
      .mockResolvedValueOnce(MOCK_RESULT);

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
    localStorage.clear();
    localStorage.setItem("ds:lang", "en");
    vi.spyOn(api, "analyzeTender").mockResolvedValue(MOCK_RESULT);
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
