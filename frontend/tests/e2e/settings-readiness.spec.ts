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

test("shows provider readiness details after a successful connection test", async ({ context, page }) => {
    await authenticate(context);

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
                    summary: "Ready for the main runtime with direct OpenAI defaults.",
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

    await page.route("**/api/user/providers", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    id: 1,
                    user_id: 1,
                    family: "STORAGE",
                    provider_type: "MINIO",
                    is_enabled: true,
                    enabled_for_indexing: false,
                    enabled_for_chat: false,
                    public_metadata: {
                        bucket_name: "thingindex-docs",
                    },
                    created_at: "2026-03-20T12:00:00Z",
                    updated_at: "2026-03-20T12:00:00Z",
                },
                {
                    id: 2,
                    user_id: 1,
                    family: "LLM",
                    provider_type: "OPENAI",
                    is_enabled: false,
                    enabled_for_indexing: true,
                    enabled_for_chat: false,
                    public_metadata: {
                        model: "gpt-4o-mini",
                    },
                    created_at: "2026-03-20T12:00:00Z",
                    updated_at: "2026-03-20T12:00:00Z",
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
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                items: [],
                summary: {
                    total_events: 0,
                    unique_providers: 0,
                    unique_requests: 0,
                },
                page_info: {
                    limit: 10,
                    returned: 0,
                    has_next: false,
                },
            })),
        });
    });

    await page.route("**/api/user/providers/test", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                family: "LLM",
                provider_type: "OPENAI",
                status: "READY",
                checked_at: "2026-03-22T12:00:00Z",
                checks: [
                    { code: "CREDENTIALS", status: "PASSED" },
                    { code: "CONNECTION", status: "PASSED" },
                    { code: "MODEL_DISCOVERY", status: "PASSED" },
                ],
                discovered_models: ["gpt-4o-mini", "gpt-4.1-mini"],
            })),
        });
    });

    await page.goto("/en/settings");

    await expect(page.getByText("Provider readiness").first()).toBeVisible();
    await expect(page.getByText("Ready for main runtime").first()).toBeVisible();
    await expect(page.getByText("Native OpenAI path", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Test connection" }).nth(1).click();

    await expect(page.getByText("Ready", { exact: true })).toBeVisible();
    await expect(page.getByText(/Validated live connectivity and model discovery for OpenAI/i)).toBeVisible();
    await expect(page.getByText("gpt-4.1-mini", { exact: true })).toBeVisible();
});
