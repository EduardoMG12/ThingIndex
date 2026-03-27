import { existsSync } from "node:fs";
import { defineConfig } from "@playwright/test";

const storybookBaseURL = process.env.THINGINDEX_STORYBOOK_BASE_URL ?? "http://127.0.0.1:6006";
const chromiumExecutablePath = [
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
    "/snap/bin/chromium",
].find((candidate) => typeof candidate === "string" && candidate.length > 0 && existsSync(candidate));

export default defineConfig({
    testDir: "./tests/storybook",
    fullyParallel: false,
    timeout: 90_000,
    expect: {
        timeout: 15_000,
    },
    retries: process.env.CI ? 1 : 0,
    workers: process.env.CI ? 1 : undefined,
    reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
    use: {
        baseURL: storybookBaseURL,
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
        command: "pnpm run storybook:test-server",
        url: storybookBaseURL,
        timeout: 180_000,
        cwd: ".",
        reuseExistingServer: !process.env.CI,
    },
});
