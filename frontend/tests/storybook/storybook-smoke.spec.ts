import { expect, test } from "@playwright/test";

test("storybook renders the introduction docs and core stories", async ({ page }) => {
    await page.goto("/iframe.html?id=introduction-workspace--docs&viewMode=docs");

    await expect(page.getByRole("heading", { name: "ThingIndex Storybook" })).toBeVisible();
    await expect(page.getByText("This workspace documents the frontend building blocks used across ThingIndex.")).toBeVisible();

    await page.goto("/iframe.html?id=ui-button--primary&viewMode=story");
    await expect(page.getByRole("button", { name: "Launch flow" })).toBeVisible();

    await page.goto("/iframe.html?id=runtime-providerruntimecallout--chat-active-runtime&viewMode=story");
    await expect(page.getByRole("heading", { name: "Active chat LLM" })).toBeVisible();

    await page.goto("/iframe.html?id=documents-operationalobservabilitypanel--default&viewMode=story");
    await expect(page.getByRole("heading", { name: "Actionable monitoring" })).toBeVisible();
    await expect(page.getByText("Critical attention")).toBeVisible();
});
