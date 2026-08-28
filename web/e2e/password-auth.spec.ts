import { randomUUID } from "node:crypto";
import { test, expect, type Page } from "@playwright/test";
import { countUsersWithEmail } from "./db";

// Password auth needs no external consent screen, so unlike the Google
// flow (seeded directly into Postgres, see fixtures.ts/db.ts) these drive
// the real UI end to end.

async function registerThroughUI(page: Page, email: string, password: string) {
  await page.goto("/");
  await page.getByRole("button", { name: "New here? Create an account" }).click();
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password").fill(password);
  await page.getByRole("button", { name: "Create account" }).click();
}

test("registering through the UI creates an account and lands in an empty Den", async ({ page }) => {
  const email = `e2e-${randomUUID()}@example.test`;
  await registerThroughUI(page, email, "correct-horse-battery");

  await expect(page.getByText("Your den is empty.")).toBeVisible();
  await expect.poll(() => countUsersWithEmail(email)).toBe(1);
});

test("registering with an already-used email shows an error and creates no account", async ({ page }) => {
  const email = `e2e-${randomUUID()}@example.test`;
  await registerThroughUI(page, email, "correct-horse-battery");
  await expect(page.getByText("Your den is empty.")).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

  await registerThroughUI(page, email, "a-different-password");

  await expect(page.getByText("email already in use")).toBeVisible();
  await expect(page.getByRole("button", { name: "Create account" })).toBeVisible();
  await expect.poll(() => countUsersWithEmail(email)).toBe(1);
});

test("signing out and back in with the same email/password reaches the same Den with prior content intact", async ({
  page,
}) => {
  const email = `e2e-${randomUUID()}@example.test`;
  const password = "correct-horse-battery";
  await registerThroughUI(page, email, password);
  await expect(page.getByText("Your den is empty.")).toBeVisible();

  await page.getByRole("button", { name: "+ New page" }).click();
  await page.waitForURL(/\/pages\//);
  const titleRow = page.locator("aside").getByText("Untitled", { exact: true });
  await titleRow.dblclick();
  await page.locator("aside").getByRole("textbox").fill("My Persisted Page");
  await page.keyboard.press("Enter");
  await expect(page.locator("aside").getByText("My Persisted Page")).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.locator("aside").getByText("My Persisted Page")).toBeVisible();
});
