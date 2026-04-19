import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { TenderReport } from "./TenderReport";

describe("TenderReport", () => {
  it("renders verdict GO by default", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Идём")).toBeInTheDocument();
    expect(screen.getByText("82%", { exact: false })).toBeInTheDocument();
  });

  it("shows strengths and risks sections", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("Преимущества")).toBeInTheDocument();
    expect(screen.getByText("Риски")).toBeInTheDocument();
  });

  it("shows requirements grid", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("TypeScript, 3+ года")).toBeInTheDocument();
    expect(screen.getByText("SOC 2 Type II")).toBeInTheDocument();
  });

  it("shows documents sidebar", () => {
    renderWithProviders(<TenderReport />);
    expect(screen.getByText("tender-tz.pdf")).toBeInTheDocument();
  });

  it("toggles to NO-GO verdict", async () => {
    renderWithProviders(<TenderReport />);
    await userEvent.click(screen.getByRole("button", { name: "NO-GO" }));
    expect(screen.getByText("Пас")).toBeInTheDocument();
  });
});
