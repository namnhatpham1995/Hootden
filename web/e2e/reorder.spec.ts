import { test, expect } from "./fixtures";
import { createRootPage, dragRow } from "./helpers";

async function rowY(page: import("@playwright/test").Page, title: string): Promise<number> {
  const box = await page.locator("aside").getByText(title, { exact: true }).boundingBox();
  if (!box) throw new Error(`row not found: ${title}`);
  return box.y;
}

test("drag to reorder root pages, and the new order survives a reload", async ({ page }) => {
  await createRootPage(page, "Page A");
  await createRootPage(page, "Page B");
  await createRootPage(page, "Page C");

  // Drop C near the top edge of A -- PageTree treats that as "before".
  await dragRow(page, "Page C", "Page A", 0.1);

  // handleMove is async (PATCH, then a refetch) before the reordered tree
  // re-renders -- toPass retries until that lands, instead of measuring
  // positions at a single, possibly-too-early instant.
  async function assertOrder() {
    await expect(async () => {
      const [yC, yA, yB] = await Promise.all([rowY(page, "Page C"), rowY(page, "Page A"), rowY(page, "Page B")]);
      expect(yC).toBeLessThan(yA);
      expect(yA).toBeLessThan(yB);
    }).toPass({ timeout: 5000 });
  }

  await assertOrder();
  await page.reload();
  await assertOrder();
});

test("drag to reparent a page under a sibling, and it survives a reload", async ({ page }) => {
  await createRootPage(page, "Parent");
  await createRootPage(page, "Child");

  // Drop Child near the middle of Parent -- PageTree treats that as "inside".
  await dragRow(page, "Child", "Parent", 0.5);

  const childRow = page.locator("aside").getByText("Child", { exact: true }).locator("xpath=..");
  await expect(childRow).toHaveCSS("margin-left", "16px");

  await page.reload();
  await expect(childRow).toHaveCSS("margin-left", "16px");
});
