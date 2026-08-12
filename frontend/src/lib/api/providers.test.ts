import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/fetch-with-auth-redirect", () => ({
    fetchWithAuthRedirect: vi.fn(),
}));

import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import { fetchProviderRuntimeOverview, fetchProviderSupportMatrix, listProviderModels, testProviderConnection } from "@/lib/api/providers";

describe("providers api", () => {
    beforeEach(() => {
        vi.mocked(fetchWithAuthRedirect).mockReset();
    });

    it("loads and normalizes discovered models", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        models: ["gpt-4o-mini", " gpt-4.1-mini ", "", null],
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const models = await listProviderModels({
            family: "LLM",
            providerType: "OPENAI",
            credentials: {
                api_key: "secret",
            },
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe("/api/user/providers/models");
        expect(models).toEqual(["gpt-4o-mini", "gpt-4.1-mini"]);
    });

    it("loads and normalizes the active provider runtime overview", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        indexing: {
                            family: "LLM",
                            activation_target: "INDEXING",
                            provider_type: "OPENAI",
                            public_metadata: {
                                model: "gpt-4o-mini",
                            },
                            updated_at: "2026-03-20T12:00:00Z",
                        },
                        chat: {
                            family: "LLM",
                            activation_target: "CHAT",
                            provider_type: "OLLAMA",
                            public_metadata: {
                                model: "llama3.1:8b",
                            },
                            updated_at: "2026-03-20T12:05:00Z",
                        },
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const overview = await fetchProviderRuntimeOverview();

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe("/api/user/providers/runtime");
        expect(overview.storage).toBeNull();
        expect(overview.indexing?.provider_type).toBe("OPENAI");
        expect(overview.indexing?.public_metadata.model).toBe("gpt-4o-mini");
        expect(overview.chat?.provider_type).toBe("OLLAMA");
    });

    it("loads and normalizes the provider readiness report", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        family: "LLM",
                        provider_type: "OPENAI",
                        status: "READY",
                        checked_at: "2026-03-22T12:00:00Z",
                        checks: [
                            { code: "CREDENTIALS", status: "PASSED" },
                            { code: "CONNECTION", status: "PASSED" },
                            { code: "MODEL_DISCOVERY", status: "PASSED" },
                        ],
                        discovered_models: ["gpt-4o-mini", " gpt-4.1-mini ", ""],
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const report = await testProviderConnection({
            family: "LLM",
            providerType: "OPENAI",
            credentials: {
                api_key: "secret",
                model: "gpt-4o-mini",
            },
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe("/api/user/providers/test");
        expect(report?.status).toBe("READY");
        expect(report?.checks).toHaveLength(3);
        expect(report?.discovered_models).toEqual(["gpt-4o-mini", "gpt-4.1-mini"]);
    });

    it("loads and normalizes the provider support matrix", async () => {
        vi.stubGlobal(
            "fetch",
            vi.fn().mockResolvedValue(
                new Response(
                    JSON.stringify({
                        status: 200,
                        type: "success",
                        message: "ok",
                        data: [
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
                                notes: ["Direct OpenAI path.", "", null],
                            },
                        ],
                    }),
                    { status: 200, headers: { "Content-Type": "application/json" } }
                )
            )
        );

        const items = await fetchProviderSupportMatrix({ family: "LLM" });

        expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe("/api/providers/support?family=LLM");
        expect(items).toEqual([
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
        ]);
    });
});
