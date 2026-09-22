import { randomUUID } from "node:crypto";
import { test, expect, type Page } from "@playwright/test";
import { countUsersWithEmail } from "./db";
import { forceApiStatus } from "./failure";

// Password auth needs no external consent screen, so unlike the Google
// flow (seeded directly into Postgres, see fixtures.ts/db.ts) these drive
// the real UI end to end.
//
// All of them therefore also share one caller as far as the server's
// login/register limiter (auth/limiter.go) is concerned: Chromium refuses to
// let a page set X-Forwarded-For itself (the same browser protection that
// makes the header trustworthy from a real reverse proxy in the first
// place), and this stack has no proxy in front of it locally to assign one,
// so every request in this file resolves to the same client address. That's
// fine -- this file's total real register/login calls stay well under the
// configured limit -- but it means this suite can't exercise the
// address-keyed refusal itself; that's covered directly against the limiter
// in auth/limiter_test.go instead.

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

test("registering with mismatched password entries shows an error, creates no account, and keeps what was typed", async ({
  page,
}) => {
  const email = `e2e-${randomUUID()}@example.test`;
  await page.goto("/");
  await page.getByRole("button", { name: "New here? Create an account" }).click();
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password", { exact: true }).fill("correct-horse-battery");
  await page.getByLabel("Confirm password").fill("a-different-password");
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page.getByText("Passwords don't match.")).toBeVisible();
  await expect(page.getByLabel("Email")).toHaveValue(email);
  await expect(page.getByLabel("Password", { exact: true })).toHaveValue("correct-horse-battery");
  await expect(page.getByLabel("Confirm password")).toHaveValue("a-different-password");
  await expect.poll(() => countUsersWithEmail(email)).toBe(0);
});

test("toggling reveal exposes both password fields in registration and the sign-in field, without changing what is submitted", async ({
  page,
}) => {
  const email = `e2e-${randomUUID()}@example.test`;
  const password = "correct-horse-battery";

  await page.goto("/");
  await page.getByRole("button", { name: "New here? Create an account" }).click();
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password").fill(password);

  await page.getByRole("button", { name: "Show password" }).click();
  await expect(page.getByLabel("Password", { exact: true })).toHaveAttribute("type", "text");
  await expect(page.getByLabel("Confirm password")).toHaveAttribute("type", "text");
  await expect(page.getByLabel("Password", { exact: true })).toHaveValue(password);
  await expect(page.getByLabel("Confirm password")).toHaveValue(password);

  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByText("Your den is empty.")).toBeVisible();
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Show password" }).click();
  await expect(page.getByLabel("Password")).toHaveAttribute("type", "text");
  await expect(page.getByLabel("Password")).toHaveValue(password);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByText("Your den is empty.")).toBeVisible();
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

// The real limiter is process-wide on the server and shared across this
// whole suite's requests, so this forces the response instead of firing
// enough real attempts to trip it -- see auth/limiter_test.go for coverage
// of the limiter itself. This only checks that the frontend tells the two
// refusals apart.
test("a 429 from sign-in shows its own message, not a failed-credential one", async ({ page }) => {
  const email = `e2e-${randomUUID()}@example.test`;
  await registerThroughUI(page, email, "correct-horse-battery");
  await expect(page.getByText("Your den is empty.")).toBeVisible();
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

  await forceApiStatus(page, "**/auth/login", 429, { method: "POST", times: 1 });

  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("correct-horse-battery");
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByText("Too many attempts. Please wait a bit and try again.")).toBeVisible();
  await expect(page.getByText("invalid email or password")).not.toBeVisible();
});
