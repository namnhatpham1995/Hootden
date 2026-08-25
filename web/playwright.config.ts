import { defineConfig, devices } from "@playwright/test";

// Runs against the full-stack `docker compose --profile full` stack (see
// docker-compose.yml) -- not `next dev` -- so tests exercise the same
// server-rendered/client-fetched wiring self-hosting relies on.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  reporter: process.env.CI ? "html" : "list",
  use: {
    baseURL: process.env.BASE_URL ?? "http://localhost:3000",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
