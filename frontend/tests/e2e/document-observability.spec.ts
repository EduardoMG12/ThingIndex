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

// O workspace de documentos tem tres abas (Library, Operations, Audit) e abre
// na Library. O painel de observabilidade vive na aba Operations — o teste
// antigo esperava tudo na primeira tela.
test("documents observability panel highlights attention slices and filters the library", async ({ baseURL, context, page }) => {
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
            body: JSON.stringify(success([
                {
                    id: 7,
                    user_id: 1,
                    title: "Orphaned invoice.pdf",
                    file_url: "documents/orphaned.pdf",
                    file_path: "documents/orphaned.pdf",
                    size: 2048,
                    mime_type: "application/pdf",
                    status: "ORPHANED",
                    collection: "Operations",
                    created_at: "2026-03-20T12:00:00Z",
                    updated_at: "2026-03-20T12:10:00Z",
                    processing_attempts: [],
                },
                {
                    id: 8,
                    user_id: 1,
                    title: "Stalled parser.pdf",
                    file_url: "documents/stalled.pdf",
                    file_path: "documents/stalled.pdf",
                    size: 2048,
                    mime_type: "application/pdf",
                    status: "PROCESSING",
                    processing_stage: "PROCESSING",
                    collection: "Operations",
                    index_started_at: "2026-03-20T10:00:00Z",
                    created_at: "2026-03-20T10:00:00Z",
                    updated_at: "2026-03-20T10:00:00Z",
                    processing_attempts: [],
                },
                {
                    id: 9,
                    user_id: 1,
                    title: "Healthy guide.pdf",
                    file_url: "documents/healthy.pdf",
                    file_path: "documents/healthy.pdf",
                    size: 2048,
                    mime_type: "application/pdf",
                    status: "INDEXED",
                    collection: "Guides",
                    created_at: "2026-03-20T11:00:00Z",
                    updated_at: "2026-03-20T11:55:00Z",
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
                total_documents: 3,
                indexed_documents: 1,
                processing_documents: 1,
                queued_documents: 0,
                failed_documents: 0,
                permanent_failed_documents: 0,
                retrying_documents: 0,
                orphaned_documents: 1,
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
                last_completed_at: "2026-03-20T11:55:00Z",
                model_breakdown: [],
            })),
        });
    });

    await page.route("**/api/documents/*/index", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                document: {
                    id: 8,
                    user_id: 1,
                    title: "Stalled parser.pdf",
                    file_url: "documents/stalled.pdf",
                    file_path: "documents/stalled.pdf",
                    size: 2048,
                    mime_type: "application/pdf",
                    status: "PROCESSING",
                    created_at: "2026-03-20T10:00:00Z",
                    updated_at: "2026-03-20T10:00:00Z",
                },
                document_pages: [],
                paragraph_nodes: [],
                table_sections: [],
                metadata_pairs: [],
                chunks: [],
                outline_nodes: [],
                document_summary: "",
                monitoring: {
                    status: "PROCESSING",
                    stage: "PROCESSING",
                },
            })),
        });
    });

    await page.route("**/api/documents/audit**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                items: [],
                summary: {
                    total_events: 0,
                    unique_documents: 0,
                    unique_collections: 0,
                    unique_requests: 0,
                    action_breakdown: [],
                    status_breakdown: [],
                },
                page_info: {
                    limit: 12,
                    returned: 0,
                    has_next: false,
                },
            })),
        });
    });

    await page.goto("/en/documents");

    await page.getByRole("tab", { name: "Operations" }).click();

    await expect(page.getByRole("heading", { name: "Actionable monitoring" })).toBeVisible();
    await expect(page.getByText("Executive brief")).toBeVisible();
    await expect(page.getByText("Next operational actions")).toBeVisible();
    await expect(page.getByText("Anomaly map")).toBeVisible();
    await expect(page.getByText("Orphaned invoice.pdf", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("Stalled parser.pdf", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("Missing originals", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Open priority document" })).toBeVisible();

    // A fatia escolhida na triagem tem que valer tambem para a biblioteca — e
    // esse acoplamento entre as duas abas que o teste protege.
    await page.getByRole("button", { name: "Focus: Unblock the flow" }).click();
    await page.getByRole("tab", { name: "Library" }).click();

    await expect(page.getByText("1 of 3 document(s) in the operational slice")).toBeVisible();
    await expect(page.getByText("Stalled parser.pdf", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("Orphaned invoice.pdf", { exact: true })).toHaveCount(0);

    // A busca da biblioteca compartilha estado e rotulo com a da aba Operations.
    await page.getByLabel("Operational search and slice").fill("healthy");
    await expect(page.getByRole("heading", { name: "No documents match the current slice" })).toBeVisible();

    await page.getByRole("button", { name: "Clear operational slice" }).first().click();
    await expect(page.getByText("3 indexed document(s)")).toBeVisible();

    await page.getByRole("tab", { name: "Operations" }).click();
    await page.getByRole("button", { name: "Open priority document" }).click();
    await page.getByRole("tab", { name: "Index" }).click();
    await expect(page.getByRole("button", { name: "Download index JSON" })).toBeVisible();
});
