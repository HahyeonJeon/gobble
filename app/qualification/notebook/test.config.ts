import { defineConfig } from 'vitest/config';
export default defineConfig({
  test: { include: ['qualification/notebook/*.test.ts'], environment: 'node' },
});
