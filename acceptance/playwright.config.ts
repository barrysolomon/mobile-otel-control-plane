import { defineConfig, devices } from '@playwright/test'

/**
 * Acceptance test config. Boot script `acceptance/scripts/boot.mjs` is
 * responsible for starting gateway + UI before invoking `playwright test`;
 * this config does NOT use Playwright's built-in `webServer` because we need
 * two coordinated server processes (gateway + UI) and want explicit log
 * surfacing on failure.
 *
 * Read `GATEWAY_URL` and `UI_URL` from env — set by boot.mjs.
 */
export default defineConfig({
  testDir: './playwright/tests',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false, // tests share a single gateway DB; serial keeps them deterministic
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['github'], ['list']] : 'list',

  use: {
    baseURL: process.env.UI_URL ?? 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
