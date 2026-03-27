import { defineConfig } from "vitest/config";
import path from "node:path";

export default defineConfig({
    test: {
        environment: "jsdom",
        globals: true,
        setupFiles: ["./vitest.setup.tsx"],
        include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
        exclude: ["tests/e2e/**", "test-results/**", "playwright-report/**"],
        css: true,
        coverage: {
            reporter: ["text", "html"],
        },
    },
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        },
    },
});
