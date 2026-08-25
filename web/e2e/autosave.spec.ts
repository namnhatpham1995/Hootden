import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";

test("editor autosaves, and content survives a reload", async ({ page }) => {
  await createRootPage(page, "Autosave Test");

  const editor = page.locator(".editor-doc");
  await editor.click();
  await page.keyboard.type("Hello from Playwright");

  await expect(page.getByText("Saved", { exact: true })).toBeVisible({ timeout: 5000 });

  await page.reload();
  await expect(page.locator(".editor-doc")).toContainText("Hello from Playwright");
});
