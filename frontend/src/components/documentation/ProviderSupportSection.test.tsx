import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createTestWrapper } from "@/test/test-utils";

const { fetchProviderSupportMatrixMock } = vi.hoisted(() => ({
    fetchProviderSupportMatrixMock: vi.fn(),
}));

vi.mock("@/lib/api/providers", async () => {
    const actual = await vi.importActual<typeof import("@/lib/api/providers")>("@/lib/api/providers");

    return {
        ...actual,
        fetchProviderSupportMatrix: fetchProviderSupportMatrixMock,
    };
});

import { ProviderSupportSection } from "@/components/documentation/ProviderSupportSection";

describe("ProviderSupportSection", () => {
    beforeEach(() => {
        fetchProviderSupportMatrixMock.mockReset();
    });

    it("renders storage and llm support summaries from the public matrix", async () => {
        fetchProviderSupportMatrixMock
            .mockResolvedValueOnce([
                {
                    family: "STORAGE",
                    provider_type: "MINIO",
                    display_name: "MinIO",
                    support_level: "READY",
                    transport: "S3_COMPATIBLE",
                    summary: "Ready for the main runtime.",
                    suggested_setup: {},
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
                    notes: ["Mainline storage path."],
                },
            ])
            .mockResolvedValueOnce([
                {
                    family: "LLM",
                    provider_type: "GOOGLE_GEMINI",
                    display_name: "Google Gemini",
                    support_level: "PARTIAL",
                    transport: "OPENAI_BRIDGE",
                    summary: "Available behind an OpenAI-compatible bridge.",
                    suggested_setup: {},
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
                    notes: ["Bridge path."],
                },
                {
                    family: "LLM",
                    provider_type: "OLLAMA",
                    display_name: "Ollama",
                    support_level: "EXPERIMENTAL",
                    transport: "OLLAMA_NATIVE",
                    summary: "Experimental local runtime path.",
                    suggested_setup: {},
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: true,
                        supports_streaming: true,
                        supports_indexing: true,
                        supports_chat: true,
                        supports_optional_live_tests: true,
                        requires_explicit_model: true,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Local model path."],
                },
            ]);

        render(<ProviderSupportSection />, { wrapper: createTestWrapper() });

        await waitFor(() => {
            expect(screen.getByRole("heading", { level: 3, name: "Storage runtime support" })).toBeInTheDocument();
        });

        expect(fetchProviderSupportMatrixMock).toHaveBeenNthCalledWith(1, {
            family: "STORAGE",
            signal: expect.any(AbortSignal),
        });
        expect(fetchProviderSupportMatrixMock).toHaveBeenNthCalledWith(2, {
            family: "LLM",
            signal: expect.any(AbortSignal),
        });
        expect(screen.getByRole("heading", { level: 3, name: "LLM runtime support" })).toBeInTheDocument();
        expect(screen.getByTestId("provider-brand-MINIO")).toBeInTheDocument();
        expect(screen.getByTestId("provider-brand-GOOGLE_GEMINI")).toBeInTheDocument();
        expect(screen.getByText("MinIO")).toBeInTheDocument();
        expect(screen.getByText("Google Gemini")).toBeInTheDocument();
        expect(screen.getByText("Ollama")).toBeInTheDocument();
        expect(screen.getAllByText("Ready", { exact: true }).length).toBeGreaterThan(0);
        expect(screen.getAllByText("Partial", { exact: true }).length).toBeGreaterThan(0);
        expect(screen.getAllByText("Experimental", { exact: true }).length).toBeGreaterThan(0);
    });
});
