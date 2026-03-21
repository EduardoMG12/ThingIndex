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

test("documents audit explorer applies filters and loads the next page", async ({ context, page }) => {
    await authenticate(context);

    const auditRequests: string[] = [];

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
            body: JSON.stringify(success([
                {
                    id: 7,
                    user_id: 1,
                    title: "Manual.pdf",
                    file_url: "documents/manual.pdf",
                    file_path: "documents/manual.pdf",
                    size: 2048,
                    mime_type: "application/pdf",
                    status: "INDEXED",
                    collection_id: 3,
                    collection: "Policies",
                    created_at: "2026-03-20T12:00:00Z",
                    updated_at: "2026-03-20T12:00:00Z",
                    processing_attempts: [],
                },
            ])),
        });
    });

    await page.route("**/api/documents/monitoring/summary", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                total_documents: 1,
                indexed_documents: 1,
                processing_documents: 0,
                queued_documents: 0,
                failed_documents: 0,
                permanent_failed_documents: 0,
                retrying_documents: 0,
                orphaned_documents: 0,
                dlq_documents: 0,
                documents_with_llm: 1,
                pricing_aware_documents: 1,
                total_llm_requests: 1,
                total_input_tokens: 100,
                total_output_tokens: 50,
                total_tokens: 150,
                total_estimated_cost_usd: 0.001,
                average_duration_ms: 2500,
                average_quality_score: 88,
                last_completed_at: "2026-03-20T12:00:00Z",
                model_breakdown: [],
            })),
        });
    });

    await page.route("**/api/documents/audit**", async (route) => {
        const requestURL = route.request().url();
        auditRequests.push(requestURL);

        if (requestURL.includes("cursor=cursor-doc-1")) {
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify(success({
                    items: [
                        {
                            id: 2,
                            user_id: 1,
                            document_id: 7,
                            collection_id: 3,
                            actor_type: "USER",
                            action: "DOCUMENT_RETRY_INDEX",
                            status: "FAILED",
                            title: "Manual.pdf",
                            detail: "retry failed",
                            metadata: {},
                            created_at: "2026-03-20T12:10:00Z",
                        },
                    ],
                    summary: {
                        total_events: 2,
                        unique_documents: 1,
                        unique_collections: 1,
                        unique_requests: 1,
                        action_breakdown: [{ key: "DOCUMENT_RETRY_INDEX", count: 1 }],
                        status_breakdown: [{ key: "FAILED", count: 1 }],
                    },
                    page_info: {
                        limit: 12,
                        returned: 1,
                        has_next: false,
                    },
                })),
            });
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                items: [
                    {
                        id: 1,
                        user_id: 1,
                        document_id: 7,
                        collection_id: 3,
                        actor_type: "USER",
                        action: "DOCUMENT_UPLOAD",
                        status: "SUCCESS",
                        title: "Manual.pdf",
                        detail: "uploaded",
                        metadata: {},
                        created_at: "2026-03-20T12:00:00Z",
                    },
                ],
                summary: {
                    total_events: 2,
                    unique_documents: 1,
                    unique_collections: 1,
                    unique_requests: 1,
                    action_breakdown: [{ key: "DOCUMENT_UPLOAD", count: 1 }],
                    status_breakdown: [{ key: "SUCCESS", count: 1 }],
                },
                page_info: {
                    limit: 12,
                    returned: 1,
                    has_next: true,
                    next_cursor: "cursor-doc-1",
                },
            })),
        });
    });

    await page.goto("/en/documents");

    await expect(page.getByText("Historico de auditoria documental")).toBeVisible();
    await page.getByLabel("Busca operacional").fill("manual");
    await page.getByRole("button", { name: "Aplicar filtros" }).click();

    await expect.poll(() => auditRequests.some((url) => url.includes("q=manual"))).toBeTruthy();

    await page.getByRole("button", { name: "Carregar mais eventos" }).click();
    await expect(page.getByText("retry failed")).toBeVisible();
});

test("provider audit explorer filters by the active provider scope", async ({ context, page }) => {
    await authenticate(context);

    const auditRequests: string[] = [];

    await page.route("**/api/user/providers", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    provider_type: "MINIO",
                    family: "STORAGE",
                    is_enabled: true,
                    enabled_for_indexing: false,
                    enabled_for_chat: false,
                    public_metadata: {},
                },
                {
                    provider_type: "OPENAI",
                    family: "LLM",
                    is_enabled: false,
                    enabled_for_indexing: true,
                    enabled_for_chat: false,
                    public_metadata: {
                        model: "gpt-4o-mini",
                    },
                },
            ])),
        });
    });

    await page.route("**/api/user/pricing-sources", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.route("**/api/user/prices", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.route("**/api/user/prices/history**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([])),
        });
    });

    await page.route("**/api/user/providers/audit**", async (route) => {
        const requestURL = route.request().url();
        auditRequests.push(requestURL);

        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                items: [
                    {
                        id: 5,
                        user_id: 1,
                        family: "LLM",
                        provider_type: "OPENAI",
                        actor_type: "USER",
                        action: "PROVIDER_CONFIG_SAVE",
                        status: "SUCCESS",
                        detail: "saved config",
                        metadata: {},
                        created_at: "2026-03-20T12:00:00Z",
                    },
                ],
                summary: {
                    total_events: 1,
                    unique_providers: 1,
                    unique_requests: 1,
                },
                page_info: {
                    limit: 10,
                    returned: 1,
                    has_next: false,
                },
            })),
        });
    });

    await page.goto("/en/settings");

    await expect(page.getByText("Mudancas sensiveis de LLMs")).toBeVisible();
    await page.getByRole("button", { name: "Ativo: OpenAI" }).click();
    await page.locator("#llm-provider-audit-query").fill("openai");
    await page.getByRole("button", { name: "Aplicar filtros" }).nth(1).click();

    await expect.poll(() =>
        auditRequests.some((requestURL) => requestURL.includes("provider_type=OPENAI") && requestURL.includes("q=openai"))
    ).toBeTruthy();
});
