import type { PropsWithChildren } from "react";

import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createTestWrapper } from "@/test/test-utils";

const {
    fetchProvidersMock,
    fetchProviderSupportMatrixMock,
} = vi.hoisted(() => ({
    fetchProvidersMock: vi.fn(),
    fetchProviderSupportMatrixMock: vi.fn(),
}));

vi.mock("@/components/layout/AppShell", () => ({
    __esModule: true,
    default: ({ children }: PropsWithChildren) => <div data-testid="app-shell">{children}</div>,
}));

vi.mock("@/lib/api/providers", async () => {
    const actual = await vi.importActual<typeof import("@/lib/api/providers")>("@/lib/api/providers");

    return {
        ...actual,
        fetchProviders: fetchProvidersMock,
        fetchProviderSupportMatrix: fetchProviderSupportMatrixMock,
    };
});

import SettingsPage from "@/app/settings/page";

describe("SettingsPage", () => {
    it("keeps provider controls visible when saved configs cannot be loaded", async () => {
        fetchProvidersMock.mockRejectedValue(new Error("Erro ao consultar providers."));
        fetchProviderSupportMatrixMock
            .mockResolvedValueOnce([
                {
                    family: "STORAGE",
                    provider_type: "MINIO",
                    display_name: "MinIO",
                    support_level: "READY",
                    transport: "S3_COMPATIBLE",
                    summary: "Ready storage path.",
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
                    notes: ["Storage note."],
                },
            ])
            .mockResolvedValueOnce([
                {
                    family: "LLM",
                    provider_type: "OPENAI",
                    display_name: "OpenAI",
                    support_level: "READY",
                    transport: "NATIVE_OPENAI",
                    summary: "Ready LLM path.",
                    suggested_setup: {},
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
                    notes: ["LLM note."],
                },
            ]);

        render(<SettingsPage />, {
            wrapper: createTestWrapper(),
        });

        await waitFor(() => {
            expect(screen.getByText(/Saved provider settings could not be loaded right now/i)).toBeInTheDocument();
        });

        expect(screen.getByTestId("app-shell")).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "Storage provider" })).toBeInTheDocument();
        expect(screen.getByRole("tab", { name: "LLM provider" })).toBeInTheDocument();
        expect(screen.getByText("MinIO")).toBeInTheDocument();
    });
});
