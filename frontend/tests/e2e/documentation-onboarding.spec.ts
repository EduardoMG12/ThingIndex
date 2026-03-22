import { expect, test } from "@playwright/test";

test.use({ locale: "en-US" });

test("documentation page exposes the open-source onboarding flow", async ({ page }) => {
    await page.goto("/en/documentation");

    await expect(page.getByRole("heading", { level: 2, name: "Onboarding Open Source" })).toBeVisible();
    await expect(page.getByRole("heading", { level: 3, name: "Self-hosted runtime" })).toBeVisible();
    await expect(page.getByRole("heading", { level: 3, name: "Hosted API consumption" })).toBeVisible();
    await expect(page.getByText("./scripts/bootstrap.sh", { exact: true })).toBeVisible();
    await expect(page.getByText("./scripts/stack.sh up", { exact: true })).toBeVisible();
    await expect(page.getByText("./scripts/doctor.sh", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("http://127.0.0.1:8000/docs", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("http://127.0.0.1:8000/api/v1", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("docs/open-source-onboarding.md", { exact: true })).toBeVisible();
    await expect(page.getByText("examples/http/thingindex-api.http", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("api/docs/swagger.yaml", { exact: true }).first()).toBeVisible();

    await expect(page.getByRole("link", { name: "Open link" }).first()).toHaveAttribute("href", /127\.0\.0\.1:8000/);
    await expect(page.getByRole("link", { name: "Open link" }).nth(1)).toHaveAttribute("href", "http://127.0.0.1:8000/swagger/index.html");
    await expect(page.getByRole("link", { name: "Open link" }).nth(2)).toHaveAttribute("href", "http://127.0.0.1:8000/swagger/doc.json");
});
