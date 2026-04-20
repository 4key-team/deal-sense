import { describe, it, expect, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import { CompanyProfile } from "./CompanyProfile";
import { loadProfile } from "./profileData";

describe("CompanyProfile (RU)", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("ds:lang", "ru");
  });

  it("renders title and subtitle", () => {
    renderWithProviders(<CompanyProfile />);
    expect(screen.getByText("Профиль компании")).toBeInTheDocument();
    expect(screen.getByText(/Заполните один раз/)).toBeInTheDocument();
  });

  it("renders all form fields", () => {
    renderWithProviders(<CompanyProfile />);
    expect(screen.getByText("Название компании")).toBeInTheDocument();
    expect(screen.getByText("Размер команды")).toBeInTheDocument();
    expect(screen.getByText("Опыт работы")).toBeInTheDocument();
    expect(screen.getByText("Технологический стек")).toBeInTheDocument();
    expect(screen.getByText("Сертификации")).toBeInTheDocument();
    expect(screen.getByText("Специализация")).toBeInTheDocument();
    expect(screen.getByText("Ключевые клиенты и проекты")).toBeInTheDocument();
    expect(screen.getByText("Дополнительная информация")).toBeInTheDocument();
  });

  it("saves profile to localStorage", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const nameInput = screen.getAllByRole("textbox")[0];
    await user.type(nameInput, "Acme Corp");

    await user.click(screen.getByRole("button", { name: /Сохранить профиль/i }));

    const saved = loadProfile();
    expect(saved.name).toBe("Acme Corp");
  });

  it("loads saved profile on mount", () => {
    localStorage.setItem(
      "ds:company-profile",
      JSON.stringify({
        name: "Saved Corp",
        teamSize: "10",
        experience: "5",
        stack: ["React"],
        certs: [],
        specializations: [],
        clients: "",
        extra: "",
      }),
    );

    renderWithProviders(<CompanyProfile />);
    expect(screen.getByDisplayValue("Saved Corp")).toBeInTheDocument();
    expect(screen.getByDisplayValue("10")).toBeInTheDocument();
    expect(screen.getByDisplayValue("5")).toBeInTheDocument();
  });

  it("shows saved confirmation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    await user.click(screen.getByRole("button", { name: /Сохранить профиль/i }));

    expect(screen.getByText("Сохранено")).toBeInTheDocument();
  });

  it("adds and removes stack tags", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const tagInput = screen.getByPlaceholderText(/React, TypeScript/);
    await user.type(tagInput, "Go{Enter}");

    expect(screen.getByText("Go")).toBeInTheDocument();

    const removeBtn = screen.getByRole("button", { name: "Remove Go" });
    await user.click(removeBtn);

    expect(screen.queryByText("Go")).not.toBeInTheDocument();
  });

  it("adds tag on comma", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const tagInput = screen.getByPlaceholderText(/React, TypeScript/);
    await user.type(tagInput, "Python,");

    expect(screen.getByText("Python")).toBeInTheDocument();
  });

  it("removes last tag on Backspace in empty input", async () => {
    localStorage.setItem(
      "ds:company-profile",
      JSON.stringify({
        name: "", teamSize: "", experience: "",
        stack: ["React", "Go"],
        certs: [], specializations: [], clients: "", extra: "",
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("Go")).toBeInTheDocument();

    // Tag input is inside the tagsWrap div
    const tagInput = document.querySelector("[class*='tagInput']") as HTMLInputElement;
    await user.click(tagInput);
    await user.keyboard("{Backspace}");

    expect(screen.queryByText("Go")).not.toBeInTheDocument();
    expect(screen.getByText("React")).toBeInTheDocument();
  });

  it("does not add duplicate tags", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const tagInput = screen.getByPlaceholderText(/React, TypeScript/);
    await user.type(tagInput, "Go{Enter}");
    await user.type(tagInput, "Go{Enter}");

    expect(screen.getAllByText("Go")).toHaveLength(1);
  });

  it("toggles certification checkbox", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const checkboxes = screen.getAllByRole("checkbox");
    // First checkbox is ISO 27001
    await user.click(checkboxes[0]);
    await user.click(screen.getByRole("button", { name: /Сохранить профиль/i }));

    const saved = loadProfile();
    expect(saved.certs).toContain("cert_iso27001");

    // uncheck
    await user.click(checkboxes[0]);
    await user.click(screen.getByRole("button", { name: /Сохранить профиль/i }));

    const saved2 = loadProfile();
    expect(saved2.certs).not.toContain("cert_iso27001");
  });

  it("toggles specialization checkbox", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    // Specializations start after 5 cert checkboxes
    const checkboxes = screen.getAllByRole("checkbox");
    await user.click(checkboxes[5]); // first specialization = spec_web
    await user.click(screen.getByRole("button", { name: /Сохранить профиль/i }));

    const saved = loadProfile();
    expect(saved.specializations).toContain("spec_web");
  });

  it("saves clients and extra text", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const textareas = document.querySelectorAll("textarea");
    await user.type(textareas[0], "Сбербанк 2024");
    await user.type(textareas[1], "Agile команда");

    await user.click(screen.getByRole("button", { name: /Сохранить профиль/i }));

    const saved = loadProfile();
    expect(saved.clients).toBe("Сбербанк 2024");
    expect(saved.extra).toBe("Agile команда");
  });

  it("shows tooltips on ? click", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyProfile />);

    const tipButtons = screen.getAllByRole("button", { name: /^Info:/ });
    expect(tipButtons.length).toBeGreaterThan(0);

    await user.click(tipButtons[0]);

    // Tooltip text should appear
    await waitFor(() => {
      const tips = document.querySelectorAll("[class*='tipBox']");
      expect(tips.length).toBeGreaterThan(0);
    });
  });
});

describe("CompanyProfile (EN)", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("ds:lang", "en");
  });

  it("renders EN labels", () => {
    renderWithProviders(<CompanyProfile />);
    expect(screen.getByText("Company profile")).toBeInTheDocument();
    expect(screen.getByText("Tech stack")).toBeInTheDocument();
    expect(screen.getByText("Certifications")).toBeInTheDocument();
  });
});
