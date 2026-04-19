import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { TenderReport } from "./TenderReport";
import * as api from "../../lib/api";

describe("TenderReport (RU, GO)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("renders verdict GO by default", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Идём")).toBeInTheDocument();
    expect(screen.getByText("82%", { exact: false })).toBeInTheDocument();
  });

  it("shows strengths and risks", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Преимущества")).toBeInTheDocument();
    expect(screen.getByText("Риски")).toBeInTheDocument();
  });

  it("shows requirements grid", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("TypeScript, 3+ года")).toBeInTheDocument();
    expect(screen.getByText("SOC 2 Type II")).toBeInTheDocument();
    expect(screen.getAllByText(/подтверждено/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/частично/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/не закрыто/).length).toBeGreaterThan(0);
  });

  it("shows documents sidebar", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("tender-tz.pdf")).toBeInTheDocument();
    expect(screen.getByText(/2\.1 MB/)).toBeInTheDocument();
  });

  it("shows effort card for GO", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText(/~6 часов/)).toBeInTheDocument();
  });

  it("shows GO action buttons", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByRole("button", { name: /Готовить заявку/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Сохранить отчёт/ })).toBeInTheDocument();
  });

  it("shows histogram and sparkline", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("63")).toBeInTheDocument();
    expect(screen.getByText("+37")).toBeInTheDocument();
  });
});

describe("TenderReport (RU, NO-GO)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("toggles to NO-GO and back", async () => {
    renderWithProviders(<TenderReport />);
    await userEvent.click(screen.getByRole("button", { name: "NO-GO" }));
    expect(screen.getByText("Пас")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "GO" }));
    expect(screen.getByText("Идём")).toBeInTheDocument();
  });

  it("shows NO-GO effort and actions", async () => {
    renderWithProviders(<TenderReport />);
    await userEvent.click(screen.getByRole("button", { name: "NO-GO" }));
    expect(screen.getByText(/~14 часов/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Набросать отказ/ })).toBeInTheDocument();
  });
});

describe("TenderReport (EN)", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "en"));

  it("renders EN GO verdict", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Go")).toBeInTheDocument();
    expect(screen.getByText("Strengths")).toBeInTheDocument();
    expect(screen.getByText("Risks")).toBeInTheDocument();
  });

  it("renders EN requirements", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("TypeScript, 3+ years")).toBeInTheDocument();
    expect(screen.getAllByText(/^met$/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^partial$/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/^missing$/).length).toBeGreaterThan(0);
  });

  it("renders EN effort and actions", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText(/~6 hours/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Start preparing bid/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Export report/ })).toBeInTheDocument();
  });

  it("renders EN NO-GO", async () => {
    renderWithProviders(<TenderReport />);
    await userEvent.click(screen.getByRole("button", { name: "NO-GO" }));
    expect(screen.getByText("Pass")).toBeInTheDocument();
    expect(screen.getByText(/~14 hours/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Draft polite decline/ })).toBeInTheDocument();
  });

  it("renders EN sidebar labels", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText(/Documents/i)).toBeInTheDocument();
    expect(screen.getByText(/Prep effort/i)).toBeInTheDocument();
  });
});

describe("TenderReport actions", () => {
  beforeEach(() => localStorage.setItem("ds:lang", "ru"));

  it("exports report as JSON on export click", async () => {
    const downloadSpy = vi.spyOn(api, "downloadBlob").mockImplementation(() => {});
    renderWithProviders(<TenderReport />);

    await userEvent.click(screen.getByRole("button", { name: /Сохранить отчёт/ }));

    expect(downloadSpy).toHaveBeenCalledOnce();
    const [blob, filename] = downloadSpy.mock.calls[0];
    expect(blob).toBeInstanceOf(Blob);
    expect(filename).toBe("tender-report.json");

    downloadSpy.mockRestore();
  });

  it("navigates to /proposal on prep click", async () => {
    renderWithProviders(<TenderReport />);

    await userEvent.click(screen.getByRole("button", { name: /Готовить заявку/ }));

    expect(window.location.pathname).toBe("/proposal");
  });
});
