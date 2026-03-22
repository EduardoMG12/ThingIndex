import { expect, test } from "@playwright/test";

test.use({ locale: "en-US" });

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
        window.localStorage.setItem("thingindex_last_splash", Date.now().toString());
    });
});

test("redirects the bare root to the localized landing page @smoke", async ({ page }) => {
    await page.goto("/");

    await expect(page).toHaveURL(/\/en\/?$/);
    await expect(page.getByRole("link", { name: "Documentation", exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: /Login/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /Start free/i })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: "Explicit product focus" })).toBeVisible();
    await expect(page.getByText("Primary use case", { exact: true })).toBeVisible();
});

test("opens the documentation page from the landing navigation @smoke", async ({ page }) => {
    await page.goto("/en");

    await page.getByRole("link", { name: "Documentation", exact: true }).click();

    await expect(page).toHaveURL(/\/en\/documentation$/);
    await expect(
        page.getByRole("heading", {
            level: 1,
            name: /estado atual do projeto/i,
        })
    ).toBeVisible();
});

test("opens the localized register page from the landing CTA @smoke", async ({ page }) => {
    await page.goto("/en");

    await page.getByRole("link", { name: /Start now/i }).click();

    await expect(page).toHaveURL(/\/en\/register$/);
    await expect(page.getByRole("button", { name: /Create account/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /Sign in/i })).toBeVisible();
});

test("redirects protected routes to the localized login page when unauthenticated @smoke", async ({ page }) => {
    await page.goto("/documents");

    await expect(page).toHaveURL(/\/en\/login\?next=%2Fen%2Fdocuments$/);
    await expect(page.getByRole("button", { name: /Sign in/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /Create account/i })).toBeVisible();
});

test("redirects authenticated users away from auth routes to the localized dashboard @smoke", async ({ context, page }) => {
    await context.addCookies([
        {
            name: "token",
            value: "playwright-token",
            url: "http://127.0.0.1:3000",
        },
    ]);

    await page.goto("/en/login");

    await expect(page).toHaveURL(/\/en\/dashboard$/);
    await expect(page.getByRole("heading", { level: 1, name: /thingindex dashboard/i })).toBeVisible();
});
