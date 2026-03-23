import { expect, test, type BrowserContext } from "@playwright/test";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

async function authenticate(context: BrowserContext) {
    await context.addCookies([
        {
            name: "token",
            value: "playwright-token",
            url: "http://127.0.0.1:3000",
        },
    ]);
}

test.use({ locale: "en-US" });

test("shows the active chat runtime in the main chat flow @smoke", async ({ context, page }) => {
    await authenticate(context);

    await page.route("**/api/user/providers/runtime", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                chat: {
                    family: "LLM",
                    activation_target: "CHAT",
                    provider_type: "OLLAMA",
                    public_metadata: {
                        model: "llama3.1:8b",
                    },
                    updated_at: "2026-03-20T12:05:00Z",
                },
            })),
        });
    });

    await page.route("**/api/collections", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    id: 7,
                    name: "Operations",
                    document_count: 1,
                    metadata: {},
                },
            ])),
        });
    });

    await page.route("**/api/chat/conversations**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.goto("/en/chat");

    await expect(page.getByRole("heading", { name: "Active chat LLM" })).toBeVisible();
    await expect(page.getByTestId("provider-brand-OLLAMA").first()).toBeVisible();
    await expect(page.getByText("Ollama", { exact: true })).toBeVisible();
    await expect(page.getByText("llama3.1:8b", { exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: /Review in settings/i })).toBeVisible();
});

test("shows the active indexing runtime in the main documents flow @smoke", async ({ context, page }) => {
    await authenticate(context);

    await page.route("**/api/user/providers/runtime", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                indexing: {
                    family: "LLM",
                    activation_target: "INDEXING",
                    provider_type: "OPENAI",
                    public_metadata: {
                        model: "gpt-4o-mini",
                    },
                    updated_at: "2026-03-20T12:00:00Z",
                },
            })),
        });
    });

    await page.route("**/api/documents", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.route("**/api/documents/monitoring/summary", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success(null)),
        });
    });

    await page.route("**/api/documents/audit**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.goto("/en/documents");

    await expect(page.getByRole("heading", { name: "Active indexing LLM" })).toBeVisible();
    await expect(page.getByTestId("provider-brand-OPENAI").first()).toBeVisible();
    await expect(page.getByText("OpenAI", { exact: true })).toBeVisible();
    await expect(page.getByText("gpt-4o-mini", { exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: /Review in settings/i })).toBeVisible();
});
