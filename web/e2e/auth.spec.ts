import { test, expect } from "./fixtures";
import { test as unauthedTest, expect as unauthedExpect } from "@playwright/test";

test("a seeded session lands on the authenticated empty Den", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("Your den is empty.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Sign in with Google" })).not.toBeVisible();
});

// Proves the positive test above is actually exercising RequireAuth, not
// just always rendering the Den regardless of cookie -- a garbage session
// cookie must be rejected the same way a missing one would be.
unauthedTest("an invalid session cookie is rejected, not treated as signed in", async ({ page, context }) => {
  await context.addCookies([
    {
      url: process.env.API_ORIGIN ?? "http://localhost:8080",
      name: "session",
      value: "not-a-real-session-token",
      httpOnly: true,
      secure: true,
      sameSite: "Lax",
    },
  ]);
  await page.goto("/");
  await unauthedExpect(page.getByRole("link", { name: "Sign in with Google" })).toBeVisible();
});
