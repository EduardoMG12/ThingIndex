import { existsSync } from "node:fs";
import { defineConfig } from "@playwright/test";

// E2E sem mock, contra a stack do docker-compose. Sem webServer: a stack
// precisa estar de pe antes.
const baseURL = process.env.THINGINDEX_REAL_BASE_URL ?? "http://localhost:3988";
const chromiumExecutablePath = [
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
    "/snap/bin/chromium",
].find((candidate) => typeof candidate === "string" && candidate.length > 0 && existsSync(candidate));

export default defineConfig({
    testDir: "./tests/e2e-real",
    fullyParallel: false,
    // Indexacao real passa por fila e extracao, leva dezenas de segundos.
    timeout: 240_000,
    expect: {
        timeout: 30_000,
    },
    // Sem retry para nao mascarar intermitencia.
    retries: 0,
    workers: 1,
    reporter: "list",
    use: {
        baseURL,
        trace: "retain-on-failure",
        screenshot: "only-on-failure",
        video: "retain-on-failure",
        ...(chromiumExecutablePath ? { launchOptions: { executablePath: chromiumExecutablePath } } : {}),
    },
    projects: [
        {
            name: "chromium-stack-real",
            use: { browserName: "chromium" },
        },
    ],
});
