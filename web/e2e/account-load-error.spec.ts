import { test, expect } from "./fixtures";
import { forceApiStatus } from "./failure";

// Covers task 5.3: a failed /me must show the "account could not be loaded"
// state rather than the blank page `me: undefined` used to render forever
// on, and a genuine signed-out load (a real 401, not forced) must still
// reach the landing screen -- proving getCurrentUser's null-on-401 path is
// unaffected by the new error state.
test("a 500 from /me shows the account-load error state, with a working retry", async ({ page }) => {
  await forceApiStatus(page, "**/me", 500, { method: "GET", times: 1 });
  await page.goto("/");

  await expect(page.getByText("Your account could not be loaded", { exact: false })).toBeVisible({
    timeout: 5000,
  });

  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByRole("button", { name: "+ New page" })).toBeVisible({ timeout: 5000 });
});

test("a genuine signed-out load still reaches the landing screen", async ({ page, context }) => {
  await context.clearCookies();
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Hootden" })).toBeVisible();
});
