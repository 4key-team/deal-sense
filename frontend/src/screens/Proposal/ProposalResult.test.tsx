import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { ProposalResult } from "./ProposalResult";
import * as api from "../../lib/api";

function docxFile(name = "template.docx") {
  return new File(["docx"], name, {
    type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  });
}

describe("ProposalResult upload phase (RU)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("renders upload screen by default", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Коммерческое предложение")).toBeInTheDocument();
    expect(screen.getByText("Шаблон КП (.docx)")).toBeInTheDocument();
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

    const btn = screen.getByRole("button", { name: /Сгенерировать КП/i });
    expect(btn).not.toBeDisabled();
  });
});

describe("ProposalResult generating phase", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("shows spinner during generation", async () => {
    let resolveGen!: () => void;
    vi.spyOn(api, "generateProposal").mockReturnValue(
      new Promise((resolve) => {
        resolveGen = () => resolve(new Blob(["docx"]));
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
    localStorage.setItem("ds:lang", "ru");
    vi.spyOn(api, "generateProposal").mockResolvedValue(new Blob(["docx"]));
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

  it("renders proposal result after generation", async () => {
    await goToResult();
    expect(screen.getByText("Коммерческое предложение")).toBeInTheDocument();
    expect(screen.getByText("Northwind Logistics")).toBeInTheDocument();
  });

  it("shows meta fields", async () => {
    await goToResult();
    expect(screen.getByText("2 450 000 ₽")).toBeInTheDocument();
    expect(screen.getByText("9 недель")).toBeInTheDocument();
  });

  it("renders sections with statuses", async () => {
    await goToResult();
    expect(screen.getByText("Резюме")).toBeInTheDocument();
    expect(screen.getAllByText("сгенерировано").length).toBeGreaterThan(0);
    expect(screen.getAllByText("заполнено").length).toBeGreaterThan(0);
    expect(screen.getByText("нужна проверка")).toBeInTheDocument();
  });

  it("shows context files", async () => {
    await goToResult();
    expect(screen.getByText("proposal-tpl.docx")).toBeInTheDocument();
    expect(screen.getByText("brief-klient.txt")).toBeInTheDocument();
  });

  it("shows changelog", async () => {
    await goToResult();
    expect(screen.getByText("14:31:04")).toBeInTheDocument();
  });

  it("shows stats footer", async () => {
    await goToResult();
    expect(screen.getByText(/Заняло/)).toBeInTheDocument();
    expect(screen.getByText(/Токенов/)).toBeInTheDocument();
  });

  it("shows source from uploaded template name", async () => {
    await goToResult();
    expect(screen.getByText(/template\.docx/)).toBeInTheDocument();
  });

  it("downloads docx on click", async () => {
    const downloadSpy = vi.spyOn(api, "downloadBlob").mockImplementation(() => {});
    const user = await goToResult();
    await user.click(screen.getByRole("button", { name: /Скачать .docx/i }));
    expect(downloadSpy).toHaveBeenCalledOnce();
    downloadSpy.mockRestore();
  });

  it("open preview is clickable", async () => {
    const user = await goToResult();
    const btn = screen.getByRole("button", { name: /Открыть в предпросмотре/i });
    await user.click(btn);
    expect(btn).toBeInTheDocument();
  });

  it("renders token counts", async () => {
    await goToResult();
    expect(screen.getByText("412 токенов")).toBeInTheDocument();
  });
});

describe("ProposalResult error phase", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("shows error on generation failure", async () => {
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

  it("handles non-Error rejection", async () => {
    vi.spyOn(api, "generateProposal").mockRejectedValue("oops");

    const user = userEvent.setup();
    renderWithProviders(<ProposalResult />);
    const inputs = screen.getAllByTestId("dropzone-input");
    await user.upload(inputs[0], docxFile());
    await user.click(screen.getByRole("button", { name: /Сгенерировать КП/i }));

    await waitFor(() => expect(screen.getByText("oops")).toBeInTheDocument());

    vi.restoreAllMocks();
  });

  it("retries from error state", async () => {
    vi.spyOn(api, "generateProposal")
      .mockRejectedValueOnce(new Error("fail"))
      .mockResolvedValueOnce(new Blob(["docx"]));

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
    localStorage.setItem("ds:lang", "en");
    vi.spyOn(api, "generateProposal").mockResolvedValue(new Blob(["docx"]));
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
    expect(screen.getByText("€24,500")).toBeInTheDocument();
    expect(screen.getByText("Executive summary")).toBeInTheDocument();
  });
});
