import { expect, test } from "@playwright/test";

test.use({ locale: "en-US" });

test("redirects the bare root to the localized landing page @smoke", async ({ page }) => {
    await page.goto("/");

    await expect(page).toHaveURL(/\/en\/?$/);
    await expect(page.getByRole("link", { name: "Documentation", exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: /Login/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /Start free/i })).toBeVisible();
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

test("redirects protected routes to the localized login page when unauthenticated @smoke", async ({ page }) => {
    await page.goto("/documents");

    await expect(page).toHaveURL(/\/en\/login\?next=%2Fen%2Fdocuments$/);
    await expect(page.getByRole("button", { name: /Sign in/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /Create account/i })).toBeVisible();
});
