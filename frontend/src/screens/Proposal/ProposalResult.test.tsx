import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { ProposalResult } from "./ProposalResult";
import * as api from "../../lib/api";
import type { ProposalResult as ProposalResultType } from "../../lib/api";

function docxFile(name = "template.docx") {
  return new File(["docx"], name, {
    type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  });
}

const MOCK_RESULT: ProposalResultType = {
  template: "template.docx",
  summary: "КП сгенерировано на основе контекста",
  sections: [
    { title: "Резюме проекта", status: "ai", tokens: 250 },
    { title: "Описание решения", status: "ai", tokens: 400 },
    { title: "Стоимость", status: "filled", tokens: 0 },
    { title: "Команда", status: "review", tokens: 120 },
  ],
  meta: { client: "Северные технологии", project: "HR-портал", price: "8.5 млн", timeline: "5 месяцев" },
  log: [
    { time: "14:31:04", msg: "прочитан шаблон · 13 плейсхолдеров" },
    { time: "14:31:06", msg: "индексирован контекст · 1 файл" },
    { time: "14:31:18", msg: "сгенерированы секции · 4 из 4" },
  ],
  docx: "ZG9jeA==",
};

describe("ProposalResult upload phase (RU)", () => {
  beforeEach(() => { localStorage.clear(); localStorage.setItem("ds:lang", "ru"); });

  it("renders upload screen by default", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Коммерческое предложение")).toBeInTheDocument();
    expect(screen.getByText("Шаблон КП (опционально)")).toBeInTheDocument();
    expect(screen.getByText("Контекст (бриф, кейсы, прайс)")).toBeInTheDocument();
  });

  it("disables generate button when no template", () => {
    renderWithProviders(<ProposalResult />);
    const btn = screen.getByRole("button", { name: /Сгенерировать КП/i });
    expect(btn).toBeDisabled();
  });

  it("enables generate button after template upload", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);

    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());

    expect(screen.getByRole("button", { name: /Сгенерировать КП/i })).not.toBeDisabled();
  });
});

describe("ProposalResult generating phase", () => {
  beforeEach(() => { localStorage.clear(); localStorage.setItem("ds:lang", "ru"); });

  it("shows spinner during generation", async () => {
    let resolveGen!: () => void;
    vi.spyOn(api, "generateProposal").mockReturnValue(
      new Promise((resolve) => {
        resolveGen = () => resolve(MOCK_RESULT);
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);
    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());
    await user.click(screen.getByRole("button", { name: /Сгенерировать КП/i }));

    expect(screen.getByText("Генерирую предложение…")).toBeInTheDocument();

    resolveGen();
    vi.restoreAllMocks();
  });
});

describe("ProposalResult result phase (RU)", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("ds:lang", "ru");
    vi.spyOn(api, "generateProposal").mockResolvedValue(MOCK_RESULT);
  });

  afterEach(() => vi.restoreAllMocks());

  async function goToResult() {
    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);
    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());
    await user.click(screen.getByRole("button", { name: /Сгенерировать КП/i }));
    await waitFor(() => expect(screen.getByText("Готово")).toBeInTheDocument());
    return user;
  }

  it("renders result after generation", async () => {
    await goToResult();
    expect(screen.getByText("Коммерческое предложение")).toBeInTheDocument();
    expect(screen.getAllByText("template.docx").length).toBeGreaterThan(0);
  });

  it("shows sections from API", async () => {
    await goToResult();
    expect(screen.getByText("Резюме проекта")).toBeInTheDocument();
    expect(screen.getByText("Описание решения")).toBeInTheDocument();
    expect(screen.getByText("Стоимость")).toBeInTheDocument();
    expect(screen.getByText("Команда")).toBeInTheDocument();
  });

  it("shows section statuses", async () => {
    await goToResult();
    expect(screen.getAllByText("сгенерировано").length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText("заполнено").length).toBeGreaterThan(0);
    expect(screen.getAllByText("нужна проверка").length).toBeGreaterThan(0);
  });

  it("shows total token count in hero", async () => {
    await goToResult();
    expect(screen.getByText(/770/)).toBeInTheDocument();
  });

  it("shows summary in hero subtitle", async () => {
    await goToResult();
    expect(screen.getAllByText(/HR-портал/).length).toBeGreaterThan(0);
  });

  it("shows uploaded files", async () => {
    await goToResult();
    expect(screen.getAllByText("template.docx").length).toBeGreaterThan(0);
  });

  it("shows download button in hero", async () => {
    await goToResult();
    expect(screen.getByRole("button", { name: /Скачать .docx/i })).toBeInTheDocument();
  });

  it("shows meta grid", async () => {
    await goToResult();
    expect(screen.getByText("Северные технологии")).toBeInTheDocument();
    expect(screen.getByText("HR-портал")).toBeInTheDocument();
    expect(screen.getByText("8.5 млн")).toBeInTheDocument();
  });

  it("shows changelog", async () => {
    await goToResult();
    expect(screen.getByText(/прочитан шаблон/)).toBeInTheDocument();
  });

  it("shows stats sidebar", async () => {
    await goToResult();
    // ai: 2, filled: 1, review: 1
    expect(screen.getAllByText("2").length).toBeGreaterThan(0);
  });
});

describe("ProposalResult error phase", () => {
  beforeEach(() => { localStorage.clear(); localStorage.setItem("ds:lang", "ru"); });

  it("shows error on failure", async () => {
    vi.spyOn(api, "generateProposal").mockRejectedValue(new Error("Server error"));

    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);
    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());
    await user.click(screen.getByRole("button", { name: /Сгенерировать КП/i }));

    await waitFor(() => expect(screen.getByText("Не удалось сгенерировать")).toBeInTheDocument());
    expect(screen.getByText("Server error")).toBeInTheDocument();

    vi.restoreAllMocks();
  });

  it("retries from error state", async () => {
    vi.spyOn(api, "generateProposal")
      .mockRejectedValueOnce(new Error("fail"))
      .mockResolvedValueOnce(MOCK_RESULT);

    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);
    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());
    await user.click(screen.getByRole("button", { name: /Сгенерировать КП/i }));

    await waitFor(() => expect(screen.getByText(/Попробовать снова/)).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /Попробовать снова/ }));

    await waitFor(() => expect(screen.getByText("Готово")).toBeInTheDocument());

    vi.restoreAllMocks();
  });
});

describe("ProposalResult (EN)", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("ds:lang", "en");
    vi.spyOn(api, "generateProposal").mockResolvedValue(MOCK_RESULT);
  });

  afterEach(() => vi.restoreAllMocks());

  it("renders EN upload screen", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Commercial proposal")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Generate proposal/i })).toBeDisabled();
  });

  it("renders EN result", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);
    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());
    await user.click(screen.getByRole("button", { name: /Generate proposal/i }));

    await waitFor(() => expect(screen.getByText("Ready")).toBeInTheDocument());
    expect(screen.getByText("Резюме проекта")).toBeInTheDocument();
  });
});
