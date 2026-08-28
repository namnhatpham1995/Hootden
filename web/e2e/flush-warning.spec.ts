import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";
import { forceApiStatus } from "./failure";

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

  // A: forced to fail permanently, so the unmount flush itself fails.
  await page.locator("aside").getByText("Flush Warn Test A", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toBeVisible();
  // times: 1 -- only the unmount flush's single PATCH should fail; the later
  // flush for page C must go through unforced to prove the "succeeds" half.
  await forceApiStatus(page, "**/pages/*", 500, { method: "PATCH", times: 1 });
  await page.locator(".editor-doc").click();
  await page.keyboard.type("this flush will fail");
  await page.locator("aside").getByText("Flush Warn Test B", { exact: true }).click();

  await expect.poll(() => alerts.length).toBeGreaterThan(0);
  expect(alerts[0]).toContain("could not be saved");

  // C: a normal, unforced flush -- must not warn.
  await page.locator("aside").getByText("Flush Warn Test C", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toBeVisible();
  await page.locator(".editor-doc").click();
  await page.keyboard.type("this flush will succeed");
  const alertsBeforeSwitch = alerts.length;
  await page.locator("aside").getByText("Flush Warn Test B", { exact: true }).click();

  // Give the successful flush a moment to land, then confirm it landed
  // silently -- no new dialog since B.
  await page.locator("aside").getByText("Flush Warn Test C", { exact: true }).click();
  await expect(page.locator(".editor-doc")).toContainText("this flush will succeed");
  expect(alerts.length).toBe(alertsBeforeSwitch);
});
