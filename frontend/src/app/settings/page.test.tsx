import type { PropsWithChildren } from "react";

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createTestWrapper } from "@/test/test-utils";

const {
    fetchProvidersMock,
    fetchProviderSupportMatrixMock,
    saveProviderMock,
    pathnameState,
} = vi.hoisted(() => ({
    fetchProvidersMock: vi.fn(),
    fetchProviderSupportMatrixMock: vi.fn(),
    saveProviderMock: vi.fn(),
    pathnameState: { value: "/en/settings" },
}));

vi.mock("next/navigation", () => ({
    usePathname: () => pathnameState.value,
    useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn(), prefetch: vi.fn(), back: vi.fn() }),
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
        saveProvider: saveProviderMock,
    };
});

import SettingsPage from "@/app/settings/page";

describe("SettingsPage", () => {
    beforeEach(() => {
        fetchProvidersMock.mockReset();
        fetchProviderSupportMatrixMock.mockReset();
        saveProviderMock.mockReset();
        pathnameState.value = "/en/settings";
    });

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

    it("mostra o preco salvo do modelo, valida e envia no payload", async () => {
        const user = userEvent.setup();
        const ollamaConfig = {
            id: 3,
            user_id: 1,
            family: "LLM",
            provider_type: "OLLAMA",
            is_enabled: false,
            enabled_for_indexing: true,
            enabled_for_chat: true,
            public_metadata: {
                model: "llama3.1:8b",
                input_cost_per_million_usd: "0.2",
                output_cost_per_million_usd: "0.8",
            },
            created_at: "2026-01-01T00:00:00Z",
            updated_at: "2026-01-01T00:00:00Z",
        };
        fetchProvidersMock.mockResolvedValue([ollamaConfig]);
        fetchProviderSupportMatrixMock.mockResolvedValue([]);
        saveProviderMock.mockResolvedValue(undefined);
        pathnameState.value = "/pt-BR/settings";

        render(<SettingsPage />, {
            wrapper: createTestWrapper("pt-BR"),
        });

        await user.click(await screen.findByRole("tab", { name: /LLM/i }));

        const inputField = await screen.findByLabelText(/Preço por 1 milhão de tokens de entrada \(US\$\)/);
        const outputField = screen.getByLabelText(/Preço por 1 milhão de tokens de saída \(US\$\)/);
        expect(inputField).toHaveValue("0.2");
        expect(outputField).toHaveValue("0.8");
        expect(
            screen.getByText("Opcional. Usado para estimar o custo das respostas; deixe em branco se o modelo for gratuito ou local."),
        ).toBeInTheDocument();

        await user.clear(inputField);
        await user.type(inputField, "-1");
        expect(inputField).toHaveAttribute("aria-invalid", "true");
        expect(screen.getAllByText(/Informe um número maior ou igual a zero/).length).toBeGreaterThan(0);

        await user.click(screen.getByRole("button", { name: "Salvar e habilitar Ollama" }));
        expect(saveProviderMock).not.toHaveBeenCalled();

        await user.clear(inputField);
        await user.type(inputField, "0,15");
        await user.clear(outputField);
        expect(inputField).not.toHaveAttribute("aria-invalid");

        await user.click(screen.getByRole("button", { name: "Salvar e habilitar Ollama" }));

        await waitFor(() => {
            expect(saveProviderMock).toHaveBeenCalledTimes(1);
        });
        const payload = saveProviderMock.mock.calls[0][0];
        expect(payload.providerType).toBe("OLLAMA");
        expect(payload.credentials.input_cost_per_million_usd).toBe("0.15");
        expect(payload.credentials).not.toHaveProperty("output_cost_per_million_usd");
    });
});
