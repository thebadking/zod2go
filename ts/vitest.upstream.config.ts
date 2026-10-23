import { defineConfig } from "vitest/config";
import { resolve } from "node:path";
const root = resolve(import.meta.dirname, "..");
const source = resolve(root, "vendor/zod/packages/zod/src");
export default defineConfig({
  root,
  resolve: {
    alias: [
      { find: "zod/v4/core", replacement: resolve(source,"v4/core/index.ts") },
      { find: "zod/v4", replacement: resolve(source,"v4/classic/index.ts") },
      { find: "zod/v3", replacement: resolve(source,"v3/index.ts") },
      { find: "zod/mini", replacement: resolve(source,"v4/mini/index.ts") },
      { find: "zod/locales", replacement: resolve(source,"v4/locales/index.ts") },
      { find: /^zod$/, replacement: resolve(source,"index.ts") },
    ],
  },
  test: {
    include: ["vendor/zod/packages/zod/src/v4/**/*.test.ts"],
    testTimeout: 60_000,
    hookTimeout: 60_000,
    maxWorkers: 2,
  },
});
