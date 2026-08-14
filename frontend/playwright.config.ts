import { existsSync } from "node:fs";
import { defineConfig } from "@playwright/test";

// A porta e configuravel de proposito. Com ela fixa, rodar a suite numa maquina
// que ja tem outro projeto na 3000 fazia o Playwright REAPROVEITAR aquele
// servidor (reuseExistingServer) e testar a aplicacao errada — as assercoes
// falhavam sem qualquer relacao com este projeto.
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
    // Um worker sempre, nao so no CI. Todos os workers compartilham o mesmo
    // servidor de dev, entao rodar em paralelo enfileirava a compilacao das
    // rotas e estourava o timeout — a suite inteira falhava em 3 testes que
    // passavam sozinhos. Serial demora ~1min30 e da o mesmo resultado do CI.
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
