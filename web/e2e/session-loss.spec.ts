import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";
import { revokeSession } from "./db";

// Covers tasks 3.2 and 3.3: a session dying mid-edit must stop the retry
// loop and say why without navigating away, and the old bug where every
// retry re-attempted apiFetch's redirect (and so kept re-arming the
// beforeunload leave-confirmation) must not recur.
test("a session revoked mid-edit stops retrying, reports itself without navigating, and keeps the typed text", async ({
  page,
  seededSession,
}) => {
  await createRootPage(page, "Session Loss Test");
  const pageUrl = page.url();
  await revokeSession(seededSession.token);

  const patchRequests: string[] = [];
  page.on("request", (req) => {
    if (req.method() === "PATCH" && req.url().includes("/pages/")) {
      patchRequests.push(req.url());
    }
  });

  let dialogSeen = false;
  page.on("dialog", (dialog) => {
    dialogSeen = true;
    dialog.dismiss().catch(() => {});
  });

  await page.locator(".editor-doc").click();
  await page.keyboard.type("this will not be saved");

  await expect(page.getByText("Signed out", { exact: false })).toBeVisible({ timeout: 5000 });
  await expect(page.locator(".editor-doc")).toContainText("this will not be saved");
  expect(page.url()).toBe(pageUrl);

  const countAfterSignOut = patchRequests.length;
  // The shortest retry backoff is 1s -- wait past two intervals to prove the
  // 401 stopped the retry loop rather than merely delaying the next attempt.
  await page.waitForTimeout(2500);
  expect(patchRequests.length).toBe(countAfterSignOut);
  expect(dialogSeen).toBe(false);
});
