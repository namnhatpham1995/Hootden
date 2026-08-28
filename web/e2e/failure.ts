import type { Page } from "@playwright/test";

// Forces the first `times` requests matching urlPattern (optionally filtered
// to one HTTP method) to fail with `status` instead of reaching the real
// API, then lets any further matching requests through unmodified. Omit
// `times` to fail every matching request for the rest of the test -- this is
// how a persistently-failing save is simulated.
export async function forceApiStatus(
  page: Page,
  urlPattern: string | RegExp,
  status: number,
  opts: { times?: number; method?: string } = {},
): Promise<void> {
  let matched = 0;
  await page.route(urlPattern, async (route) => {
    if (opts.method && route.request().method() !== opts.method) {
      await route.continue();
      return;
    }
    matched++;
    if (opts.times !== undefined && matched > opts.times) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status,
      contentType: "application/json",
      body: JSON.stringify({ error: `forced ${status} for testing` }),
    });
  });
}
