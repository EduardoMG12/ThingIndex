import { expect, test } from "@playwright/test";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

test.use({ locale: "en-US" });

test("documentation page exposes the open-source onboarding flow", async ({ page }) => {
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
                        label: "SBC article draft",
                        path: "article/thingindex-sbc-article.md",
                        kind: "ARTICLE_DRAFT",
                        summary: "Versioned article draft.",
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
                    family: "LLM",
                    provider_type: "OPENAI",
                    display_name: "OpenAI",
                    support_level: "READY",
                    transport: "NATIVE_OPENAI",
                    summary: "Ready for the main runtime.",
                    suggested_base_url: "https://api.openai.com/v1",
                    suggested_model: "gpt-4o-mini",
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

    await page.goto("/en/documentation");

    await expect(page.getByRole("heading", { level: 2, name: "Foco do Produto" })).toBeVisible();
    await expect(page.getByText("/api/v1/meta/runtime-profile", { exact: true })).toBeVisible();
    await expect(page.getByText(/Operations and compliance teams/i)).toBeVisible();
    await expect(page.getByText("rfc/0001-product-focus.md", { exact: true })).toBeVisible();
    await expect(page.getByText("article/thingindex-sbc-article.md", { exact: true })).toBeVisible();
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
    await expect(page.getByText("Ready", { exact: true })).toBeVisible();
    await expect(page.getByText("Experimental", { exact: true })).toBeVisible();
    await expect(page.getByText("Google Gemini", { exact: true })).toBeVisible();

    await expect(page.getByRole("link", { name: "Open link" }).first()).toHaveAttribute("href", /127\.0\.0\.1:8000/);
    await expect(page.getByRole("link", { name: "Open link" }).nth(1)).toHaveAttribute("href", "http://127.0.0.1:8000/swagger/index.html");
    await expect(page.getByRole("link", { name: "Open link" }).nth(2)).toHaveAttribute("href", "http://127.0.0.1:8000/swagger/doc.json");
});
