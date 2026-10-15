import { existsSync } from "node:fs";
import { defineConfig } from "@playwright/test";

// Porta configuravel: fixa em 3000, o reuseExistingServer pegava o servidor
// de outro projeto rodando na maquina.
const e2ePort = process.env.THINGINDEX_E2E_PORT ?? "3000";
const frontendBaseURL = process.env.THINGINDEX_FRONTEND_BASE_URL ?? `http://127.0.0.1:${e2ePort}`;
const apiBaseURL = process.env.NEXT_PUBLIC_API_URL ?? process.env.THINGINDEX_API_BASE_URL ?? "http://127.0.0.1:8000/api/v1";
const chromiumExecutablePath = [
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
    "/snap/bin/chromium",
].find((candidate) => typeof candidate === "string" && candidate.length > 0 && existsSync(candidate));

export default defineConfig({
    testDir: "./tests/e2e",
    fullyParallel: false,
    timeout: 90_000,
    expect: {
        timeout: 15_000,
    },
    retries: process.env.CI ? 1 : 0,
    // Um worker so: em paralelo o dev server enfileira a compilacao das rotas e
    // estoura o timeout.
    workers: 1,
    reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
    use: {
        baseURL: frontendBaseURL,
        trace: "retain-on-failure",
        screenshot: "only-on-failure",
        video: "retain-on-failure",
        launchOptions: chromiumExecutablePath
            ? {
                  executablePath: chromiumExecutablePath,
              }
            : undefined,
    },
    webServer: {
        command: "pnpm run dev:e2e",
        url: frontendBaseURL,
        timeout: 180_000,
        reuseExistingServer: !process.env.CI,
        env: {
            API_INTERNAL_URL: process.env.API_INTERNAL_URL ?? apiBaseURL,
            NEXT_PUBLIC_API_URL: apiBaseURL,
            THINGINDEX_E2E_PORT: e2ePort,
        },
    },
});
