import { expect, test } from "@playwright/test";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

// Enderecos montados a partir de NEXT_PUBLIC_API_URL, sem porta fixa.
const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://127.0.0.1:8000/api/v1";
const apiOrigin = apiBase.replace(/\/api\/v1\/?$/, "");

test.use({ locale: "en-US" });

test("documentation page exposes the open-source onboarding flow", async ({ page }) => {
    // Pagina publica, mas o bootstrap de sessao roda igual; sem o mock o 401 leva ao login.
    await page.route("**/api/user/me", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({ id: 1, name: "Playwright", email: "playwright@example.com", avatarUrl: null, updatedAt: "2026-01-01T00:00:00Z" })),
        });
    });

    await page.route("**/api/runtime/profile", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                product_name: "ThingIndex",
                tagline: "Relational document intelligence for auditable operational knowledge.",
                primary_use_case: "Organize collections of operational, compliance, legal, and research documents with typed relational indexing.",
                core_problem: "Disconnected document tooling makes provenance and context recovery difficult.",
                value_proposition: "Relational-first context recovery with explicit provenance.",
                target_users: [
                    {
                        name: "Operations and compliance teams",
                        description: "Need searchable workspaces and explicit provenance.",
                    },
                    {
                        name: "Academic and research environments",
                        description: "Need inspectable and reproducible runtime behavior.",
                    },
                ],
                current_scope: [
                    "JWT-authenticated document workspace",
                    "Structured indexing with typed metadata",
                ],
                differentiators: [
                    "Relational-first context reconstruction instead of vector-first navigation.",
                ],
                non_goals: [
                    "Do not present the main runtime as a generic AI copilot detached from document provenance.",
                ],
                research_tracks: [
                    "Benchmark the relational baseline against vector retrieval.",
                ],
                artifact_refs: [
                    {
                        label: "Product focus RFC",
                        path: "rfc/0001-product-focus.md",
                        kind: "RFC",
                        summary: "Primary use case and backlog implications.",
                    },
                    {
                        label: "Academic workspace",
                        path: "docs/article/README.md",
                        kind: "ARTICLE_WORKSPACE",
                        summary: "Academic workspace notes.",
                    },
                ],
                last_reviewed_on: "2026-03-22",
            })),
        });
    });

    await page.route("**/api/providers/support**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    family: "STORAGE",
                    provider_type: "GCP_STORAGE",
                    display_name: "Google Cloud Storage",
                    support_level: "READY",
                    transport: "NATIVE_GCS",
                    summary: "Ready for the main runtime.",
                    suggested_setup: {
                        project_id: "thingindex-dev",
                        bucket_name: "thingindex-docs",
                    },
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: false,
                        supports_streaming: false,
                        supports_indexing: true,
                        supports_chat: false,
                        supports_optional_live_tests: true,
                        requires_explicit_model: false,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Supports service-account JSON or emulator_host."],
                },
                {
                    family: "LLM",
                    provider_type: "OPENAI",
                    display_name: "OpenAI",
                    support_level: "READY",
                    transport: "NATIVE_OPENAI",
                    summary: "Ready for the main runtime.",
                    suggested_setup: {
                        base_url: "https://api.openai.com/v1",
                        model: "gpt-4o-mini",
                    },
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: true,
                        supports_streaming: true,
                        supports_indexing: true,
                        supports_chat: true,
                        supports_optional_live_tests: true,
                        requires_explicit_model: false,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Direct OpenAI path."],
                },
                {
                    family: "LLM",
                    provider_type: "GOOGLE_GEMINI",
                    display_name: "Google Gemini",
                    support_level: "EXPERIMENTAL",
                    transport: "OPENAI_COMPATIBLE_BRIDGE",
                    summary: "Experimental bridge-only support.",
                    suggested_setup: {
                        base_url: "Bring your bridge URL",
                        model: "gemini-2.5-flash",
                    },
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: true,
                        supports_streaming: true,
                        supports_indexing: true,
                        supports_chat: true,
                        supports_optional_live_tests: true,
                        requires_explicit_model: true,
                        requires_explicit_base_url: true,
                        requires_openai_compatible_bridge: true,
                    },
                    notes: ["Native Gemini API integration is not implemented in the main runtime."],
                },
            ])),
        });
    });

    await page.route("**/api/runtime/indexing-benchmark", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                dataset_version: "2026-03-relational-baseline-v1",
                corpus_source_path: "internal/usecase/testdata/indexing-benchmark/corpus.json",
                last_reviewed_on: "2026-03-22",
                summary: "Controlled benchmark for the current indexing baseline.",
                methodology: ["Run the deterministic indexer."],
                pass_criteria: ["Top-1 exact match."],
                limitations: ["Small corpus."],
                heuristic_quality_note: "Heuristic is operational only.",
                corpus: [
                    {
                        id: "service-agreement",
                        title: "Master Service Agreement",
                        mime_type: "text/plain",
                        focus: "Payment deadlines.",
                        query_count: 1,
                        expected_outline_titles: ["2 Payment Terms"],
                        expected_table_count: 0,
                        outline_node_count: 4,
                        table_count: 0,
                        structure_passed: true,
                        retrieval_ready: true,
                        heuristic_quality_score: 98,
                    },
                ],
                metrics: {
                    corpus_documents: 3,
                    document_queries: 4,
                    collection_queries: 3,
                    structure_pass_rate: 33.33333333333333,
                    tree_node_top1_accuracy: 0,
                    tree_node_recall_at_3: 25,
                    tree_page_top1_accuracy: 75,
                    tree_page_recall_at_3: 75,
                    collection_document_top1_accuracy: 66.66666666666666,
                    collection_document_recall_at_3: 66.66666666666666,
                    exact_match_accuracy: 28.57142857142857,
                    average_heuristic_score: 98,
                    minimum_heuristic_score: 98,
                    maximum_heuristic_score: 98,
                },
                cases: [
                    {
                        id: "service-payment-deadline",
                        scope: "DOCUMENT_TREE",
                        query: "payment deadline invoice receipt",
                        document_title: "Master Service Agreement",
                        expected_node_title: "2 Payment Terms",
                        actual_node_title: "2 Payment Terms",
                        node_matched: true,
                        node_in_top_3: true,
                        expected_page_number: 1,
                        actual_page_number: 1,
                        page_matched: true,
                        page_in_top_3: true,
                        passed: true,
                    },
                ],
            })),
        });
    });

    await page.goto("/en/documentation");

    await expect(page.getByRole("heading", { level: 2, name: "Product focus" })).toBeVisible();
    await expect(page.getByText("/api/v1/meta/runtime-profile", { exact: true })).toBeVisible();
    await expect(page.getByText(/Operations and compliance teams/i)).toBeVisible();
    await expect(page.getByText("rfc/0001-product-focus.md", { exact: true })).toBeVisible();
    await expect(page.getByText("docs/article/README.md", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: "Indexing benchmark" })).toBeVisible();
    await expect(page.getByText("/api/v1/meta/indexing-benchmark", { exact: true })).toBeVisible();
    await expect(page.getByText("internal/usecase/testdata/indexing-benchmark/corpus.json", { exact: true })).toBeVisible();
    await expect(page.getByText("28.6%", { exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: "Open-source onboarding" })).toBeVisible();
    await expect(page.getByRole("heading", { level: 3, name: "Self-hosted runtime" })).toBeVisible();
    await expect(page.getByRole("heading", { level: 3, name: "Hosted API consumption" })).toBeVisible();
    await expect(page.getByText("./scripts/bootstrap.sh", { exact: true })).toBeVisible();
    await expect(page.getByText("./scripts/stack.sh up", { exact: true })).toBeVisible();
    await expect(page.getByText("./scripts/doctor.sh", { exact: true }).first()).toBeVisible();
    await expect(page.getByText(`${apiOrigin}/docs`, { exact: true }).first()).toBeVisible();
    await expect(page.getByText(apiBase, { exact: true }).first()).toBeVisible();
    await expect(page.getByText("docs/open-source-onboarding.md", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("examples/http/thingindex-api.http", { exact: true }).first()).toBeVisible();
    await expect(page.getByText("api/cmd/index-import", { exact: true })).toBeVisible();
    await expect(page.getByText("api/docs/swagger.yaml", { exact: true }).first()).toBeVisible();
    // O roadmap nao tem mais mensagem de backlog vazio.
    await expect(page.getByText("Ready", { exact: true })).toHaveCount(2);
    await expect(page.getByText("Experimental", { exact: true })).toHaveCount(2);
    await expect(page.getByText("Google Gemini", { exact: true }).first()).toBeVisible();
    await expect(page.getByTestId("provider-brand-GOOGLE_GEMINI").first()).toBeVisible();

    await expect(page.getByRole("link", { name: "Open link" }).first()).toHaveAttribute("href", new RegExp(apiOrigin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
    await expect(page.getByRole("link", { name: "Open link" }).nth(1)).toHaveAttribute("href", `${apiOrigin}/swagger/index.html`);
    await expect(page.getByRole("link", { name: "Open link" }).nth(2)).toHaveAttribute("href", `${apiOrigin}/swagger/doc.json`);
});
