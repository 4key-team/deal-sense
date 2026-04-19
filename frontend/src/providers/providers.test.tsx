import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider } from "./ThemeProvider";
import { useTheme } from "./useTheme";
import { I18nProvider } from "./I18nProvider";
import { useI18n } from "./useI18n";

function ThemeConsumer() {
  const { theme, toggleTheme } = useTheme();
  return <button onClick={toggleTheme}>{theme}</button>;
}

function I18nConsumer() {
  const { lang, setLang, t } = useI18n();
  return (
    <div>
      <span data-testid="lang">{lang}</span>
      <span data-testid="tab">{t.tabs.kp}</span>
      <button onClick={() => setLang("en")}>to-en</button>
    </div>
  );
}

describe("ThemeProvider", () => {
  beforeEach(() => localStorage.clear());

  it("defaults to light", () => {
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    expect(screen.getByRole("button").textContent).toBe("light");
  });

  it("toggles light to dark", async () => {
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    await userEvent.click(screen.getByRole("button"));
    expect(screen.getByRole("button").textContent).toBe("dark");
  });

  it("toggles dark to light", async () => {
    localStorage.setItem("ds:theme", "dark");
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    expect(screen.getByRole("button").textContent).toBe("dark");
    await userEvent.click(screen.getByRole("button"));
    expect(screen.getByRole("button").textContent).toBe("light");
  });

  it("restores light from localStorage", () => {
    localStorage.setItem("ds:theme", "light");
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    expect(screen.getByRole("button").textContent).toBe("light");
  });

  it("persists to localStorage", async () => {
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    await userEvent.click(screen.getByRole("button"));
    expect(localStorage.getItem("ds:theme")).toBe("dark");
  });

  it("restores from localStorage", () => {
    localStorage.setItem("ds:theme", "dark");
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    expect(screen.getByRole("button").textContent).toBe("dark");
  });

  it("falls back to prefers-color-scheme dark", () => {
    const original = window.matchMedia;
    window.matchMedia = (query: string) => ({
      matches: query === "(prefers-color-scheme: dark)",
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    });
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    expect(screen.getByRole("button").textContent).toBe("dark");
    window.matchMedia = original;
  });

  it("falls back to light when matchMedia is unavailable", () => {
    const original = window.matchMedia;
    // @ts-expect-error testing edge case
    window.matchMedia = undefined;
    render(<ThemeProvider><ThemeConsumer /></ThemeProvider>);
    expect(screen.getByRole("button").textContent).toBe("light");
    window.matchMedia = original;
  });
});

describe("I18nProvider", () => {
  beforeEach(() => localStorage.clear());

  it("defaults to ru", () => {
    render(<I18nProvider><I18nConsumer /></I18nProvider>);
    expect(screen.getByTestId("lang").textContent).toBe("ru");
    expect(screen.getByTestId("tab").textContent).toBe("Генератор КП");
  });

  it("switches to en", async () => {
    render(<I18nProvider><I18nConsumer /></I18nProvider>);
    await userEvent.click(screen.getByText("to-en"));
    expect(screen.getByTestId("lang").textContent).toBe("en");
    expect(screen.getByTestId("tab").textContent).toBe("Proposal");
  });

  it("persists lang to localStorage", async () => {
    render(<I18nProvider><I18nConsumer /></I18nProvider>);
    await userEvent.click(screen.getByText("to-en"));
    expect(localStorage.getItem("ds:lang")).toBe("en");
  });

  it("restores from localStorage", () => {
    localStorage.setItem("ds:lang", "en");
    render(<I18nProvider><I18nConsumer /></I18nProvider>);
    expect(screen.getByTestId("lang").textContent).toBe("en");
  });
});

describe("hooks outside provider", () => {
  it("useTheme throws without provider", () => {
    expect(() => render(<ThemeConsumer />)).toThrow("useTheme must be used within ThemeProvider");
  });

  it("useI18n throws without provider", () => {
    expect(() => render(<I18nConsumer />)).toThrow("useI18n must be used within I18nProvider");
  });
});
