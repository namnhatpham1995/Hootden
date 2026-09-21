import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";
import { forceApiStatus } from "./failure";

test("a persistently-400 save stops retrying, reports itself, and keeps the typed text", async ({ page }) => {
  await createRootPage(page, "Rejected Save Test");
  await forceApiStatus(page, "**/pages/*", 400, { method: "PATCH" });

  const patchRequests: string[] = [];
  page.on("request", (req) => {
    if (req.method() === "PATCH" && req.url().includes("/pages/")) {
      patchRequests.push(req.url());
    }
  });

  await page.locator(".editor-doc").click();
  await page.keyboard.type("this will never save");

  await expect(page.getByText("Could not be saved", { exact: false })).toBeVisible({ timeout: 5000 });
  await expect(page.locator(".editor-doc")).toContainText("this will never save");

  const countAfterRejection = patchRequests.length;
  // The shortest retry backoff is 1s -- wait well past it to prove no
  // second attempt was scheduled, not just that the message appeared once.
  await page.waitForTimeout(2000);
  expect(patchRequests.length).toBe(countAfterRejection);
});
