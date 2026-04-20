import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "../../providers/I18nProvider";
import { SettingsDrawer } from "./SettingsDrawer";
import type { LLMSettings } from "./SettingsDrawer";
import { getItem } from "../../lib/storage";
import * as api from "../../lib/api";

function renderDrawer(onClose = vi.fn()) {
  return render(
    <I18nProvider>
      <SettingsDrawer open onClose={onClose} />
    </I18nProvider>,
  );
}

describe("SettingsDrawer", () => {
  beforeEach(() => localStorage.clear());

  it("renders nothing when closed", () => {
    const { container } = render(
      <I18nProvider>
        <SettingsDrawer open={false} onClose={vi.fn()} />
      </I18nProvider>,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders dialog when open", () => {
    renderDrawer();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("saves settings to localStorage on Save click", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderDrawer(onClose);

    const keyInput = screen.getByDisplayValue("");
    await user.type(keyInput, "sk-test-123");

    const saveBtn = screen.getByRole("button", { name: /сохранить|save/i });
    await user.click(saveBtn);

    const saved = getItem<LLMSettings>("llm-settings", {
      providerId: "",
      apiKey: "",
      url: "",
      model: "",
    });
    expect(saved.apiKey).toBe("sk-test-123");
    expect(saved.providerId).toBe("anthropic");
    expect(onClose).toHaveBeenCalled();
  });

  it("loads saved settings on mount", () => {
    localStorage.setItem(
      "ds:llm-settings",
      JSON.stringify({
        providerId: "openai",
        apiKey: "sk-saved",
        url: "https://api.openai.com/v1",
        model: "gpt-4o",
      }),
    );

    renderDrawer();

    const keyInput = screen.getByDisplayValue("sk-saved");
    expect(keyInput).toBeInTheDocument();
  });

  it("closes on backdrop click", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderDrawer(onClose);

    const backdrop = document.querySelector("[aria-hidden='true']")!;
    await user.click(backdrop);
    expect(onClose).toHaveBeenCalled();
  });

  it("closes on X button click", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderDrawer(onClose);

    const closeBtn = screen.getByRole("button", { name: "Close" });
    await user.click(closeBtn);
    expect(onClose).toHaveBeenCalled();
  });

  it("closes on Cancel click", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderDrawer(onClose);

    const cancelBtn = screen.getByRole("button", { name: /отмена|cancel/i });
    await user.click(cancelBtn);
    expect(onClose).toHaveBeenCalled();
  });

  it("toggles API key visibility", async () => {
    const user = userEvent.setup();
    renderDrawer();

    const keyInput = document.querySelector("input[type='password']")!;
    expect(keyInput).toBeInTheDocument();

    const toggleBtn = screen.getByRole("button", { name: /показать|show/i });
    await user.click(toggleBtn);

    expect(keyInput.getAttribute("type")).toBe("text");
  });

  it("switches provider and updates url/model", async () => {
    const user = userEvent.setup();
    renderDrawer();

    const providerTrigger = screen.getByText("Anthropic");
    await user.click(providerTrigger);

    const openaiOption = screen.getByText("OpenAI");
    await user.click(openaiOption);

    const urlInput = screen.getByDisplayValue("https://api.openai.com/v1");
    expect(urlInput).toBeInTheDocument();

    expect(screen.getByDisplayValue("gpt-4o")).toBeInTheDocument();
  });

  it("switches model via dropdown", async () => {
    const user = userEvent.setup();
    renderDrawer();

    expect(screen.getByDisplayValue("claude-haiku-4-5")).toBeInTheDocument();

    // Open model dropdown
    await user.click(screen.getByRole("button", { name: "Pick model" }));

    const sonnetOption = screen.getByText("claude-sonnet-4-5");
    await user.click(sonnetOption);

    expect(screen.getByDisplayValue("claude-sonnet-4-5")).toBeInTheDocument();
  });

  it("handles unknown provider gracefully", async () => {
    const user = userEvent.setup();
    renderDrawer();

    const providerTrigger = screen.getByText("Anthropic");
    await user.click(providerTrigger);

    const dropdownBackdrop = document.querySelectorAll("[class*='backdrop']");
    await user.click(dropdownBackdrop[dropdownBackdrop.length - 1]);

    expect(screen.getByText("Anthropic")).toBeInTheDocument();
  });

  it("allows editing URL manually", async () => {
    const user = userEvent.setup();
    renderDrawer();

    const urlInput = screen.getByDisplayValue("https://api.anthropic.com/v1");
    await user.clear(urlInput);
    await user.type(urlInput, "http://localhost:8080");

    await user.click(screen.getByRole("button", { name: /сохранить|save/i }));

    const saved = getItem<LLMSettings>("llm-settings", {
      providerId: "",
      apiKey: "",
      url: "",
      model: "",
    });
    expect(saved.url).toBe("http://localhost:8080");
  });

  it("allows typing custom model name", async () => {
    const user = userEvent.setup();
    renderDrawer();

    const modelInput = screen.getByDisplayValue("claude-haiku-4-5");
    await user.clear(modelInput);
    await user.type(modelInput, "my-custom-model");

    await user.click(screen.getByRole("button", { name: /сохранить|save/i }));

    const saved = getItem<LLMSettings>("llm-settings", {
      providerId: "",
      apiKey: "",
      url: "",
      model: "",
    });
    expect(saved.model).toBe("my-custom-model");
  });

  it("saves changed provider to localStorage", async () => {
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByText("Anthropic"));
    await user.click(screen.getByText("OpenAI"));

    await user.click(screen.getByRole("button", { name: /сохранить|save/i }));

    const saved = getItem<LLMSettings>("llm-settings", {
      providerId: "",
      apiKey: "",
      url: "",
      model: "",
    });
    expect(saved.providerId).toBe("openai");
    expect(saved.url).toBe("https://api.openai.com/v1");
    expect(saved.model).toBe("gpt-4o");
  });

  it("shows ok state when connection succeeds", async () => {
    vi.spyOn(api, "checkConnection").mockResolvedValue({ ok: true, provider: "anthropic" });
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: /проверить|test/i }));

    await waitFor(() => expect(screen.getByText(/работает|working/i)).toBeInTheDocument());
    vi.restoreAllMocks();
  });

  it("shows fail state when connection fails", async () => {
    vi.spyOn(api, "checkConnection").mockResolvedValue({ ok: false, provider: "anthropic", error: "bad key" });
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: /проверить|test/i }));

    await waitFor(() => expect(screen.getByText(/не получилось|failed/i)).toBeInTheDocument());
    vi.restoreAllMocks();
  });

  it("shows fail state when connection throws", async () => {
    vi.spyOn(api, "checkConnection").mockRejectedValue(new Error("network"));
    const user = userEvent.setup();
    renderDrawer();

    await user.click(screen.getByRole("button", { name: /проверить|test/i }));

    await waitFor(() => expect(screen.getByText(/не получилось|failed/i)).toBeInTheDocument());
    vi.restoreAllMocks();
  });
});
