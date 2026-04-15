import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  /* Tests share server state — run serially to avoid delete tests corrupting others */
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  /* list   → live terminal output during the run
     html   → interactive report served after the run on port 9323
     json   → machine-readable results file for archiving / future parsing */
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'test-results/results.json' }],
    // In CI: emit GitHub annotations so failures appear inline on commits/PRs
    ...(process.env.CI ? [['github'] as [string]] : []),
  ],
  use: {
    /* baseURL is overridden via BASE_URL when running inside Docker */
    baseURL: process.env.BASE_URL || 'http://localhost:8025',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
