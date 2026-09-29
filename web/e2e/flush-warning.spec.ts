import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";
import { forceApiStatus } from "./failure";

// Opens a page and waits for its doc to load -- typing before the load's
// setContent lands would be overwritten, leaving nothing dirty to flush.
async function openPage(page: Page, title: string) {
  const loaded = page.waitForResponse(
    (res) => res.request().method() === "GET" && /\/pages\/[^/?]+$/.test(res.url()),
  );
  await page.locator("aside").getByText(title, { exact: true }).click();
  await loaded;
  await expect(page.locator(".editor-doc")).toBeVisible();
}

// Covers task 4.2: the unmount flush is a single attempt with no component
// left to retry from, so a failure has nowhere left to go but a warning --
// and a flush that succeeds must stay silent.
test("a flush that fails on unmount warns, and a flush that succeeds does not", async ({ page }) => {
  await createRootPage(page, "Flush Warn Test A");
  await createRootPage(page, "Flush Warn Test B");
  await createRootPage(page, "Flush Warn Test C");

  const alerts: string[] = [];
  page.on("dialog", (dialog) => {
    alerts.push(dialog.message());
    dialog.accept().catch(() => {});
  });

  // A: every PATCH fails until the alert fires, so the unmount flush fails
  // even if the 800ms autosave debounce got a (retried) attempt in first --
  // a `times: 1` budget could be spent by that autosave instead.
  await openPage(page, "Flush Warn Test A");
  await forceApiStatus(page, "**/pages/*", 500, { method: "PATCH" });
  await page.locator(".editor-doc").click();
  await page.keyboard.type("this flush will fail");
  await page.locator("aside").getByText("Flush Warn Test B", { exact: true }).click();

  await expect.poll(() => alerts.length).toBeGreaterThan(0);
  expect(alerts[0]).toContain("could not be saved");
  // Lift the forcing so C's flush goes through unforced to prove the
  // "succeeds" half.
  await page.unroute("**/pages/*");

  // C: a normal, unforced flush -- must not warn.
  await openPage(page, "Flush Warn Test C");
  // Whichever save carries the full text -- the 800ms autosave if it fires
  // before the switch, else the unmount flush -- must land before C is
  // reopened, or the reload can race ahead of it and read the pre-edit doc.
  const saved = page.waitForResponse(
    (res) =>
      res.request().method() === "PATCH" &&
      (res.request().postData() ?? "").includes("this flush will succeed") &&
      res.ok(),
  );
  await page.locator(".editor-doc").click();
  await page.keyboard.type("this flush will succeed");
  const alertsBeforeSwitch = alerts.length;
  await page.locator("aside").getByText("Flush Warn Test B", { exact: true }).click();
  await saved;

  // Confirm it landed silently -- no new dialog since the switch.
  await page.locator("aside").getByText("Flush Warn Test C", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toContainText("this flush will succeed");
  expect(alerts.length).toBe(alertsBeforeSwitch);
});
