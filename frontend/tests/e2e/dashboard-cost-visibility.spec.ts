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

test("cost visibility page explains trust and pricing coverage from runtime data", async ({ baseURL, context, page }) => {
    await authenticate(context, baseURL);

    await page.route("**/api/categories", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.route("**/api/collections", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

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
                        model: "gpt-4.1-mini",
                    },
                    updated_at: "2026-03-22T12:00:00Z",
                },
            })),
        });
    });

    await page.route("**/api/documents/monitoring/summary", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                total_documents: 4,
                indexed_documents: 4,
                processing_documents: 0,
                queued_documents: 0,
                failed_documents: 0,
                permanent_failed_documents: 0,
                retrying_documents: 0,
                orphaned_documents: 0,
                dlq_documents: 0,
                documents_with_llm: 3,
                pricing_aware_documents: 3,
                total_llm_requests: 8,
                total_input_tokens: 1400,
                total_output_tokens: 400,
                total_tokens: 1800,
                total_estimated_cost_usd: 0.0123,
                average_duration_ms: 2400,
                average_quality_score: 88,
                last_completed_at: "2026-03-22T11:55:00Z",
                model_breakdown: [
                    {
                        provider_type: "OPENAI",
                        model: "gpt-4.1-mini",
                        document_count: 3,
                        request_count: 8,
                        total_tokens: 1800,
                        total_cost_usd: 0.0123,
                    },
                ],
            })),
        });
    });

    await page.route("**/api/user/pricing-sources", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    id: 1,
                    user_id: 1,
                    provider_type: "OPENAI",
                    name: "OpenAI Pricing",
                    source_url: "https://example.com/openai",
                    is_enabled: true,
                    last_synced_at: "2026-03-22T11:30:00Z",
                    last_sync_status: "SUCCESS",
                    last_sync_error: "",
                    last_sync_duration_ms: 1200,
                    last_price_count: 3,
                    created_at: "2026-03-22T10:00:00Z",
                    updated_at: "2026-03-22T11:30:00Z",
                },
            ])),
        });
    });

    await page.route("**/api/user/prices", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    id: 11,
                    pricing_source_id: 1,
                    provider_type: "OPENAI",
                    model_key: "gpt-4.1-mini",
                    model_name: "gpt-4.1-mini",
                    unit: "per 1M tokens",
                    currency: "USD",
                    flat_price_usd: 0,
                    input_cost_per_million_usd: 0.2,
                    output_cost_per_million_usd: 0.8,
                    cached_input_cost_per_million_usd: 0.1,
                    context_cost_per_million_usd: 0,
                    price_notes: "",
                    last_synced_at: "2026-03-22T11:30:00Z",
                },
            ])),
        });
    });

    await page.goto("/en/cost-visibility");

    await expect(page.getByText("Operational cost estimate")).toBeVisible();
    await expect(page.getByText("Trusted estimate")).toBeVisible();
    await expect(page.getByText("$0.012300", { exact: true })).toBeVisible();
    await expect(page.getByText("3/3", { exact: true })).toBeVisible();
    await expect(page.getByText("Pricing source health")).toBeVisible();
    await expect(page.getByText("OpenAI Pricing", { exact: true })).toBeVisible();
    await expect(page.getByText("Observed cost drivers")).toBeVisible();
    await expect(page.getByText("gpt-4.1-mini", { exact: true })).toHaveCount(2);
    await expect(page.getByText("Grounded chat usage, providers without synced pricing, and future pricing changes stay outside this view.")).toBeVisible();
    await expect(page.getByRole("link", { name: "Open pricing monitor" })).toHaveAttribute("href", "/en/settings");
    await expect(page.getByRole("link", { name: "Open document telemetry" })).toHaveAttribute("href", "/en/documents");
});
