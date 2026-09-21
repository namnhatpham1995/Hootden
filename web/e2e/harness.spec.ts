import { test, expect } from "./fixtures";
import { createRootPage } from "./helpers";
import { forceApiStatus } from "./failure";
import { revokeSession } from "./db";

const API_ORIGIN = process.env.API_ORIGIN ?? "http://localhost:8080";

// Proves the failure-injection helper actually breaks a request rather than
// silently doing nothing -- every later test that forces a status depends
// on this working.
test("forceApiStatus makes a save fail and shows the failed state", async ({ page }) => {
  await createRootPage(page, "Harness Test");
  await forceApiStatus(page, "**/pages/*", 503, { method: "PATCH" });

  await page.locator(".editor-doc").click();
  await page.keyboard.type("this save will fail");

  await expect(page.getByText("Failed to save", { exact: false })).toBeVisible({ timeout: 5000 });
});

// Proves revokeSession produces a real 401 from the API, not one simulated
// by route interception -- the distinction the design calls out as the
// reason this is driven through a real revocation instead of forceApiStatus.
test("revokeSession makes the next API call receive a genuine 401", async ({ seededSession }) => {
  await revokeSession(seededSession.token);

  const res = await fetch(`${API_ORIGIN}/me`, {
    headers: { Cookie: `session=${seededSession.token}` },
  });
  expect(res.status).toBe(401);
});
