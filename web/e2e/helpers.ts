import { expect, type Page } from "@playwright/test";

// Creates a root page from the tree and renames it in one go, so specs can
// build a small tree by title without caring about the generated id.
export async function createRootPage(page: Page, title: string): Promise<void> {
  await page.goto("/");
  await page.getByRole("button", { name: "+ New page" }).click();
  await page.waitForURL(/\/pages\//);
  const titleSpan = page.locator("aside").getByText("Untitled", { exact: true });
  await titleSpan.dblclick();
  await page.locator("aside").getByRole("textbox").fill(title);
  await page.keyboard.press("Enter");
  // handleRename is async (PATCH, then a refetch) before the tree re-renders
  // with the new title -- wait for that to land before returning, so a
  // following createRootPage call doesn't race ahead and find this page
  // still showing "Untitled".
  await expect(page.locator("aside").getByText(title, { exact: true })).toBeVisible();
}

async function rowBox(page: Page, title: string) {
  const row = page.locator("aside").getByText(title, { exact: true }).locator("xpath=..");
  const box = await row.boundingBox();
  if (!box) throw new Error(`row not found: ${title}`);
  return box;
}

// Drives PageTree's drag-and-drop the way it's actually implemented --
// onMouseDown/onMouseEnter/onMouseUp, not HTML5 drag events -- so
// Playwright's dragTo() (built for the HTML5 DnD API) won't fire the right
// handlers. relY picks the drop placement PageTree computes from cursor
// position within the target row: <0.25 "before", >0.75 "after", else
// "inside".
export async function dragRow(page: Page, fromTitle: string, toTitle: string, relY: number): Promise<void> {
  const fromRow = page.locator("aside").getByText(fromTitle, { exact: true }).locator("xpath=..");
  const from = await rowBox(page, fromTitle);
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
  await page.mouse.down();
  // Wait for React to actually commit the mousedown's setState (draggedId)
  // before moving -- a fixed delay races under parallel-worker CPU
  // contention instead of guaranteeing the state landed.
  await expect(fromRow).toHaveCSS("opacity", "0.4");

  const to = await rowBox(page, toTitle);
  // A single jump, not an interpolated multi-step move: PageTree computes
  // drop placement once, from e.clientY at the mouseenter that fires on
  // crossing into the row's bounds, and never re-evaluates it from later
  // mousemove within the same row. Interpolated steps would cross the row's
  // near edge first and freeze the placement there instead of at relY.
  await page.mouse.move(to.x + to.width / 2, to.y + to.height * relY);
  // Wait for the drop-target marker PageTree actually renders for this
  // placement before releasing -- "before"/"after" set a border-*-color,
  // "inside" sets an outline; matching the inline style string sidesteps
  // needing the resolved CSS custom property value.
  const toRow = page.locator("aside").getByText(toTitle, { exact: true }).locator("xpath=..");
  const marker = relY < 0.25 || relY > 0.75 ? "var(--accent)" : "var(--secondary)";
  await expect(toRow).toHaveAttribute("style", new RegExp(marker.replace(/[()]/g, "\\$&")));

  await page.mouse.up();
}
