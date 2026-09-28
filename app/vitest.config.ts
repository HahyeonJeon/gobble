import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['contracts/tests/**/*.test.ts', 'desktop/tests/**/*.test.ts', 'tests/**/*.test.ts'],
    environment: 'node',
  },
});
