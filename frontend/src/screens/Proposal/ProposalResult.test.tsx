import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/render";
import { ProposalResult } from "./ProposalResult";

describe("ProposalResult", () => {
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

  it("renders all 8 sections", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("Резюме")).toBeInTheDocument();
    expect(screen.getByText("Условия работы")).toBeInTheDocument();
    expect(screen.getByText("Секции шаблона · 8")).toBeInTheDocument();
  });

  it("shows context files", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByText("proposal-tpl.docx")).toBeInTheDocument();
    expect(screen.getByText("brief-klient.txt")).toBeInTheDocument();
  });

  it("renders download button", () => {
    renderWithProviders(<ProposalResult />);
    expect(screen.getByRole("button", { name: /Скачать .docx/i })).toBeInTheDocument();
  });
});
