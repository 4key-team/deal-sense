import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { ProposalResult } from "./ProposalResult";
import * as api from "../../lib/api";

describe("ProposalResult (RU)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("renders proposal title", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Коммерческое предложение")).toBeInTheDocument();
  });

  it("shows meta fields", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Northwind Logistics")).toBeInTheDocument();
    expect(screen.getByText("2 450 000 ₽")).toBeInTheDocument();
    expect(screen.getByText("9 недель")).toBeInTheDocument();
  });

  it("renders all 8 sections with statuses", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Резюме")).toBeInTheDocument();
    expect(screen.getByText("Условия работы")).toBeInTheDocument();
    // ai status
    expect(screen.getAllByText("сгенерировано").length).toBeGreaterThan(0);
    // filled status
    expect(screen.getAllByText("заполнено").length).toBeGreaterThan(0);
    // review status
    expect(screen.getByText("нужна проверка")).toBeInTheDocument();
  });

  it("shows context files with roles", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("proposal-tpl.docx")).toBeInTheDocument();
    expect(screen.getByText("brief-klient.txt")).toBeInTheDocument();
    expect(screen.getAllByText(/шаблон/).length).toBeGreaterThan(0);
  });

  it("shows changelog", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("14:31:04")).toBeInTheDocument();
    expect(screen.getByText(/прочитан шаблон/)).toBeInTheDocument();
  });

  it("shows stats footer", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText(/Заняло/)).toBeInTheDocument();
    expect(screen.getByText(/Токенов/)).toBeInTheDocument();
    expect(screen.getByText(/18 стр\./)).toBeInTheDocument();
  });

  it("renders download button", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByRole("button", { name: /Скачать .docx/i })).toBeInTheDocument();
  });

  it("renders token counts on ai sections", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("412 токенов")).toBeInTheDocument();
    expect(screen.getByText("1120 токенов")).toBeInTheDocument();
  });
});

describe("ProposalResult (EN)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "en"));

  it("renders EN title and meta", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Commercial proposal")).toBeInTheDocument();
    expect(screen.getByText("€24,500")).toBeInTheDocument();
    expect(screen.getByText("9 weeks")).toBeInTheDocument();
  });

  it("renders EN section statuses", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Executive summary")).toBeInTheDocument();
    expect(screen.getAllByText("generated").length).toBeGreaterThan(0);
    expect(screen.getAllByText("filled").length).toBeGreaterThan(0);
    expect(screen.getByText("needs review")).toBeInTheDocument();
  });

  it("renders EN context and stats", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("client-brief.txt")).toBeInTheDocument();
    expect(screen.getAllByText(/template/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/pages/).length).toBeGreaterThan(0);
  });

  it("renders EN token counts", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("412 tokens")).toBeInTheDocument();
  });
});

describe("ProposalResult actions", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("calls generateProposal and downloads on click", async () => {
    const mockBlob = new Blob(["docx"]);
    const generateSpy = vi.spyOn(api, "generateProposal").mockResolvedValue(mockBlob);
    const downloadSpy = vi.spyOn(api, "downloadBlob").mockImplementation(() => {});

    renderWithProviders(<ProposalResult />);

    const downloadBtn = screen.getByRole("button", { name: /Скачать .docx/i });
    await userEvent.click(downloadBtn);

    expect(generateSpy).toHaveBeenCalledOnce();
    expect(downloadSpy).toHaveBeenCalledWith(mockBlob, "proposal.docx");

    generateSpy.mockRestore();
    downloadSpy.mockRestore();
  });

  it("handles download error gracefully", async () => {
    const generateSpy = vi.spyOn(api, "generateProposal").mockRejectedValue(new Error("fail"));

    renderWithProviders(<ProposalResult />);

    const downloadBtn = screen.getByRole("button", { name: /Скачать .docx/i });
    await userEvent.click(downloadBtn);

    // Should not crash, button should be enabled again
    expect(downloadBtn).not.toBeDisabled();

    generateSpy.mockRestore();
  });

  it("open preview button is clickable", async () => {
    renderWithProviders(<ProposalResult />);
    const btn = screen.getByRole("button", { name: /Открыть в предпросмотре/i });
    await userEvent.click(btn);
    expect(btn).toBeInTheDocument();
  });
});
