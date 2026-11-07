import { defineConfig, devices } from "@playwright/test";
import { LEARNER_STATE } from "./lib/env";

const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:3000";
const isCI = !!process.env.CI;

/**
 * E2E runs against an already-running stack (see `make e2e-up`):
 *   docker compose -f docker-compose.yml -f docker-compose.e2e.yml up -d --build --wait
 * It does not start servers itself.
 */
export default defineConfig({
  testDir: "./tests",
  outputDir: "./test-results",
  // Journeys share a database; keep them deterministic.
  fullyParallel: false,
  workers: 1,
  forbidOnly: isCI,
  retries: isCI ? 1 : 0,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: isCI
    ? [["github"], ["list"], ["html", { open: "never", outputFolder: "playwright-report" }]]
    : [["list"], ["html", { open: "never", outputFolder: "playwright-report" }]],
  use: {
    baseURL,
    locale: "th-TH",
    timezoneId: "Asia/Bangkok",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      // Logs in Owner (first Learner ever → Owner) then a regular Learner via
      // POST /api/auth/dev-login, saves storage states, and seeds test Questions via MCP.
      name: "setup",
      testMatch: /global\.setup\.ts/,
    },
    {
      name: "learner",
      dependencies: ["setup"],
      testIgnore: /global\.setup\.ts/,
      use: { ...devices["Desktop Chrome"], storageState: LEARNER_STATE },
    },
  ],
});
