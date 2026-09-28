import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './desktop/tests/electron',
  workers: 1,
  timeout: 30_000,
  fullyParallel: false,
  reporter: 'list',
  outputDir: 'test-results/electron',
});
