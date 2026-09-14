import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  timeout: process.env.CLUB_TEST_URL ? 90000 : 45000,
  expect: { timeout: process.env.CLUB_TEST_URL ? 12000 : 7000 },
  workers: 1,
  use: {
    baseURL: process.env.CLUB_TEST_URL || "http://127.0.0.1:18080",
    viewport: { width: 375, height: 812 },
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  reporter: "list",
});
