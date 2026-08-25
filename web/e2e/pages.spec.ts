import { test, expect } from "./fixtures";

test("create, rename, and delete a page persist across reload", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "+ New page" }).click();
  await page.waitForURL(/\/pages\//);

  const titleRow = page.locator("aside").getByText("Untitled", { exact: true });
  await titleRow.dblclick();
  await page.locator("aside").getByRole("textbox").fill("My First Page");
  await page.keyboard.press("Enter");
  await expect(page.locator("aside").getByText("My First Page")).toBeVisible();

  await page.reload();
  await expect(page.locator("aside").getByText("My First Page")).toBeVisible();
  await expect(page.getByRole("heading", { name: "My First Page" })).toBeVisible();

  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "Delete page" }).click();
  await expect(page.getByText("Your den is empty.")).toBeVisible();

  await page.reload();
  await expect(page.getByText("Your den is empty.")).toBeVisible();
});
