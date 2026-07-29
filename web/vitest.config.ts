import { defineConfig } from "vitest/config";

// Pure-logic tests run under the fast `node` environment (matching the rest of
// the workspace); files that exercise React hooks opt into jsdom per-file with a
// `// @vitest-environment jsdom` docblock at the top of the test.
export default defineConfig({
  test: {
    environment: "node",
    include: ["src/**/*.test.{ts,tsx}"],
    globals: true,
  },
});
