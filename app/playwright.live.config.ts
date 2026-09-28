import { defineConfig } from '@playwright/test';

/** Explicit opt-in: this suite needs real Docker and never substitutes a runtime. */
export default defineConfig({
  testDir: './desktop/tests/live',
  workers: 1,
  timeout: 240_000,
  fullyParallel: false,
  reporter: 'list',
  outputDir: 'test-results/live-runtime',
});
