import { expect, test, type BrowserContext } from "@playwright/test";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

// A URL do cookie acompanha a baseURL. Fixar a porta 3000 fazia o cookie ser
// descartado sempre que a suite rodava em outra porta, e todo teste autenticado
// caia na tela de login sem explicar o motivo.
// Alem do cookie, o bootstrap do app busca /api/user/me. Sem esse mock a
// chamada ia para a API real com um token falso, recebia 401 e a pagina
// redirecionava para o login — todo teste autenticado falhava com "elemento nao
// encontrado", escondendo a causa real.
async function authenticateRoutes(context: BrowserContext) {
    await context.route("**/api/user/me", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify({
                status: 200,
                type: "success",
                message: "ok",
                data: {
                    id: 1,
                    name: "Playwright",
                    email: "playwright@example.com",
                    avatarUrl: null,
                    updatedAt: "2026-01-01T00:00:00Z",
                },
            }),
        });
    });
}

async function authenticate(context: BrowserContext, baseURL: string | undefined) {
    await authenticateRoutes(context);
    await context.addCookies([
        {
            name: "token",
            value: "playwright-token",
            url: baseURL ?? "http://127.0.0.1:3000",
        },
    ]);
}

test.use({ locale: "en-US" });

test("shows the active chat runtime in the main chat flow @smoke", async ({ baseURL, context, page }) => {
    await authenticate(context, baseURL);

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

test("shows the active indexing runtime in the main documents flow @smoke", async ({ baseURL, context, page }) => {
    await authenticate(context, baseURL);

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
