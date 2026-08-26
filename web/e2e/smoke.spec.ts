import { test, expect } from "@playwright/test";

test("landing page loads signed out", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Hootden" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
});
