import { test, expect } from "@playwright/test";

test.describe("DealSense smoke tests", () => {
  test("tender page loads without console errors", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (err) => errors.push(err.message));

    await page.goto("/tender");
    await expect(page).toHaveTitle("DealSense");
    await expect(page.getByText("Идём")).toBeVisible();
    expect(errors).toHaveLength(0);
  });

  test("proposal page loads without console errors", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (err) => errors.push(err.message));

    await page.goto("/proposal");
    await expect(page.getByText("Коммерческое предложение")).toBeVisible();
    expect(errors).toHaveLength(0);
  });

  test("language switches to EN", async ({ page }) => {
    await page.goto("/tender");
    await page.getByRole("button", { name: "en" }).click();
    await expect(page.getByText("Go")).toBeVisible();
    await expect(page.getByText("Partner portal")).toBeVisible();
  });

  test("theme toggles to dark", async ({ page }) => {
    await page.goto("/tender");
    await page.getByRole("button", { name: /dark theme/i }).click();
    const html = page.locator("html");
    await expect(html).toHaveAttribute("data-theme", "dark");
  });

  test("settings drawer opens from StatusPill", async ({ page }) => {
    await page.goto("/tender");
    await page.getByRole("button", { name: /Anthropic/i }).click();
    await expect(page.getByText("LLM-провайдер")).toBeVisible();
  });

  test("tabs navigate between tender and proposal", async ({ page }) => {
    await page.goto("/tender");
    await page.getByRole("tab", { name: /Генератор КП/i }).click();
    await expect(page).toHaveURL(/\/proposal/);
    await expect(page.getByText("Коммерческое предложение")).toBeVisible();

    await page.getByRole("tab", { name: /Анализ тендеров/i }).click();
    await expect(page).toHaveURL(/\/tender/);
    await expect(page.getByText("Идём")).toBeVisible();
  });

  test("verdict toggle switches to NO-GO", async ({ page }) => {
    await page.goto("/tender");
    await page.getByRole("button", { name: "NO-GO" }).click();
    await expect(page.getByText("Пас")).toBeVisible();
  });
});
