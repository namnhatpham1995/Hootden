import { test as base, expect } from "@playwright/test";
import { seedSession } from "./db";

const API_ORIGIN = process.env.API_ORIGIN ?? "http://localhost:8080";

type Fixtures = { seededSession: { userId: string; denId: string; token: string } };

// Seeds a fresh signed-in user before every test in files that import this
// `test` -- `auto: true` runs it even when a test never references the
// fixture by name, so plain `import { test } from "./fixtures"` is enough.
export const test = base.extend<Fixtures>({
  seededSession: [
    async ({ context }, use) => {
      const { userId, denId, token } = await seedSession();
      await context.addCookies([
        { url: API_ORIGIN, name: "session", value: token, httpOnly: true, secure: true, sameSite: "Lax" },
      ]);
      await use({ userId, denId, token });
    },
    { auto: true },
  ],
});

export { expect };
