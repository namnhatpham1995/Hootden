import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";
import { forceApiStatus } from "./failure";

// The flush fires from Editor's unmount cleanup and is never awaited by
// anything -- the component is already gone, there's nothing left to await
// it -- so it's a genuine race against whatever runs next. Waiting for the
// PATCH response here (rather than just clicking and hoping) is what makes
// these tests assert the flush landed instead of assuming it did.
function waitForPagePatch(page: Page) {
  return page.waitForResponse((res) => res.request().method() === "PATCH" && res.url().includes("/pages/"));
}

// Covers task 4.3: switching pages before the debounce (or the backoff
// timer) ever fires must not drop the edit -- Editor's unmount cleanup has
// to flush it, not just clear the pending timer.
test("typing and immediately switching pages persists the edit without waiting for Saved", async ({ page }) => {
  await createRootPage(page, "Flush Test A");
  await createRootPage(page, "Flush Test B");

  await page.locator("aside").getByText("Flush Test A", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toBeVisible();

  await page.locator(".editor-doc").click();
  await page.keyboard.type("flushed on unmount");
  // Switch away well within the 800ms debounce -- this proves the flush,
  // not a save that happened to fire first.
  await Promise.all([waitForPagePatch(page), page.locator("aside").getByText("Flush Test B", { exact: true }).click()]);

  await page.locator("aside").getByText("Flush Test A", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toContainText("flushed on unmount");
});

// Covers task 4.4: leaving mid-retry (after a save has already failed once
// and is waiting out its backoff) must still get the edit out via the
// unmount flush, exactly like the never-attempted case above.
test("leaving during a retry backoff still saves the edit via the unmount flush", async ({ page }) => {
  await createRootPage(page, "Flush Retry Test A");
  await createRootPage(page, "Flush Retry Test B");

  await page.locator("aside").getByText("Flush Retry Test A", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toBeVisible();

  // Fail exactly the first save so the editor lands in "failed -- retrying"
  // with a pending backoff timer, then switch away while still in it.
  await forceApiStatus(page, "**/pages/*", 503, { method: "PATCH", times: 1 });

  await page.locator(".editor-doc").click();
  await page.keyboard.type("saved despite a retry in flight");

  await expect(page.getByText("Failed to save", { exact: false })).toBeVisible({ timeout: 15000 });
  await Promise.all([
    waitForPagePatch(page),
    page.locator("aside").getByText("Flush Retry Test B", { exact: true }).click(),
  ]);

  await page.locator("aside").getByText("Flush Retry Test A", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toContainText("saved despite a retry in flight");
});
