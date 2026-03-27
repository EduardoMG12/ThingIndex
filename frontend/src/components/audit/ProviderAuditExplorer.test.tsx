import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/audit", () => ({
    fetchProviderAuditEvents: vi.fn(),
}));

import { fetchProviderAuditEvents } from "@/lib/api/audit";
import ProviderAuditExplorer from "@/components/audit/ProviderAuditExplorer";

describe("ProviderAuditExplorer", () => {
    beforeEach(() => {
        vi.mocked(fetchProviderAuditEvents).mockReset();
    });

    it("loads provider audit data and applies active-provider filters", async () => {
        vi.mocked(fetchProviderAuditEvents)
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 5,
                        user_id: 7,
                        provider_type: "OPENAI",
                        family: "LLM",
                        actor_type: "USER",
                        action: "PROVIDER_CONFIG_SAVE",
                        status: "SUCCESS",
                        detail: "saved config",
                        metadata: {},
                        created_at: "2026-03-20T12:00:00Z",
                    },
                ],
                summary: {
                    total_events: 2,
                    unique_providers: 1,
                    unique_requests: 1,
                    last_event_at: "2026-03-20T12:00:00Z",
                    action_breakdown: [{ key: "PROVIDER_CONFIG_SAVE", count: 1 }],
                    status_breakdown: [{ key: "SUCCESS", count: 1 }],
                },
                page_info: {
                    limit: 10,
                    returned: 1,
                    has_next: false,
                },
            })
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 6,
                        user_id: 7,
                        provider_type: "OPENAI",
                        family: "LLM",
                        actor_type: "USER",
                        action: "PROVIDER_ACTIVATION_CHANGE",
                        status: "SUCCESS",
                        detail: "enabled for chat",
                        metadata: {},
                        created_at: "2026-03-20T12:10:00Z",
                    },
                ],
                summary: {
                    total_events: 1,
                    unique_providers: 1,
                    unique_requests: 1,
                    last_event_at: "2026-03-20T12:10:00Z",
                    action_breakdown: [{ key: "PROVIDER_ACTIVATION_CHANGE", count: 1 }],
                    status_breakdown: [{ key: "SUCCESS", count: 1 }],
                },
                page_info: {
                    limit: 10,
                    returned: 1,
                    has_next: false,
                },
            });

        const user = userEvent.setup();

        render(
            <ProviderAuditExplorer
                locale="pt-BR"
                family="LLM"
                activeProviderName="OpenAI"
                activeProviderType="OPENAI"
            />
        );

        await waitFor(() => {
            expect(fetchProviderAuditEvents).toHaveBeenCalledWith(
                expect.objectContaining({
                    family: "LLM",
                    providerType: undefined,
                })
            );
        });

        expect(await screen.findByText("saved config")).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: "Ativo: OpenAI" }));
        await user.type(screen.getByLabelText("Busca operacional"), "chat");
        await user.type(screen.getByLabelText("Actor"), "USER");
        await user.type(screen.getByLabelText("Actor ID"), "52");
        await user.click(screen.getByRole("button", { name: /Aplicar filtros/i }));

        await waitFor(() => {
            expect(fetchProviderAuditEvents).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    family: "LLM",
                    providerType: "OPENAI",
                    query: "chat",
                    actorType: "USER",
                    actorId: 52,
                })
            );
        });

        expect(await screen.findByText("Filtros ativos")).toBeInTheDocument();

        expect(await screen.findByText("enabled for chat")).toBeInTheDocument();
    });

    it("applies the failure quick filter from the shortcuts", async () => {
        vi.mocked(fetchProviderAuditEvents)
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 5,
                        user_id: 7,
                        provider_type: "OPENAI",
                        family: "LLM",
                        actor_type: "USER",
                        action: "PROVIDER_CONFIG_SAVE",
                        status: "SUCCESS",
                        detail: "saved config",
                        metadata: {},
                        created_at: "2026-03-20T12:00:00Z",
                    },
                ],
                summary: {
                    total_events: 3,
                    unique_providers: 1,
                    unique_requests: 1,
                    last_event_at: "2026-03-20T12:00:00Z",
                    action_breakdown: [{ key: "PROVIDER_CONFIG_SAVE", count: 1 }],
                    status_breakdown: [{ key: "FAILED", count: 2 }],
                },
                page_info: {
                    limit: 10,
                    returned: 1,
                    has_next: false,
                },
            })
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 6,
                        user_id: 7,
                        provider_type: "OPENAI",
                        family: "LLM",
                        actor_type: "USER",
                        action: "PROVIDER_ACTIVATION_CHANGE",
                        status: "FAILED",
                        detail: "activation failed",
                        metadata: {},
                        created_at: "2026-03-20T12:10:00Z",
                    },
                ],
                summary: {
                    total_events: 2,
                    unique_providers: 1,
                    unique_requests: 1,
                    last_event_at: "2026-03-20T12:10:00Z",
                    action_breakdown: [{ key: "PROVIDER_ACTIVATION_CHANGE", count: 1 }],
                    status_breakdown: [{ key: "FAILED", count: 2 }],
                },
                page_info: {
                    limit: 10,
                    returned: 1,
                    has_next: false,
                },
            });

        const user = userEvent.setup();

        render(
            <ProviderAuditExplorer
                locale="pt-BR"
                family="LLM"
                activeProviderName="OpenAI"
                activeProviderType="OPENAI"
            />
        );

        await screen.findByText("saved config");

        await user.click(screen.getByRole("button", { name: /Somente falhas/i }));

        await waitFor(() => {
            expect(fetchProviderAuditEvents).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    family: "LLM",
                    status: "FAILED",
                })
            );
        });

        expect(await screen.findByText("activation failed")).toBeInTheDocument();
    });
});
