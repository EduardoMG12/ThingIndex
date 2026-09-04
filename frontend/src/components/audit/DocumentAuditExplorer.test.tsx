import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("@/lib/api/audit", () => ({
    fetchDocumentAuditEvents: vi.fn(),
}));

import { fetchDocumentAuditEvents } from "@/lib/api/audit";
import DocumentAuditExplorer from "@/components/audit/DocumentAuditExplorer";

describe("DocumentAuditExplorer", () => {
    beforeEach(() => {
        vi.mocked(fetchDocumentAuditEvents).mockReset();
    });

    it("loads audit data, applies document scope filters and appends the next page", async () => {
        vi.mocked(fetchDocumentAuditEvents)
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 1,
                        user_id: 9,
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
                    last_event_at: "2026-03-20T12:00:00Z",
                    action_breakdown: [{ key: "DOCUMENT_UPLOAD", count: 1 }],
                    status_breakdown: [{ key: "SUCCESS", count: 1 }],
                },
                page_info: {
                    limit: 12,
                    returned: 1,
                    has_next: true,
                    next_cursor: "cursor-doc-1",
                },
            })
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 1,
                        user_id: 9,
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
                    last_event_at: "2026-03-20T12:00:00Z",
                    action_breakdown: [{ key: "DOCUMENT_UPLOAD", count: 1 }],
                    status_breakdown: [{ key: "SUCCESS", count: 1 }],
                },
                page_info: {
                    limit: 12,
                    returned: 1,
                    has_next: true,
                    next_cursor: "cursor-doc-1",
                },
            })
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 2,
                        user_id: 9,
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
                    last_event_at: "2026-03-20T12:10:00Z",
                    action_breakdown: [{ key: "DOCUMENT_RETRY_INDEX", count: 1 }],
                    status_breakdown: [{ key: "FAILED", count: 1 }],
                },
                page_info: {
                    limit: 12,
                    returned: 1,
                    has_next: false,
                },
            });

        const user = userEvent.setup();

        render(
            <DocumentAuditExplorer
                locale="pt-BR"
                selectedDocument={{
                    id: 7,
                    title: "Manual.pdf",
                    collection_id: 3,
                    collection: "Policies",
                }}
            />
        );

        await waitFor(() => {
            expect(fetchDocumentAuditEvents).toHaveBeenCalledWith(
                expect.objectContaining({
                    limit: 12,
                    documentId: undefined,
                    collectionId: undefined,
                })
            );
        });

        expect(await screen.findByText("uploaded")).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: /Documento selecionado/i }));
        await user.type(screen.getByLabelText("Encontre um evento"), "manual");
        await user.click(screen.getByRole("button", { name: /Mostrar filtros avancados/i }));
        await user.selectOptions(screen.getByLabelText("Actor"), "USER");
        await user.type(screen.getByLabelText("Actor ID"), "9");
        await user.click(screen.getByRole("button", { name: /Buscar no historico/i }));

        await waitFor(() => {
            expect(fetchDocumentAuditEvents).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    query: "manual",
                    documentId: 7,
                    actorType: "USER",
                    actorId: 9,
                })
            );
        });

        expect(await screen.findByText("Filtros ativos")).toBeInTheDocument();
        expect(screen.getAllByText("Usuario").length).toBeGreaterThan(0);

        await user.click(screen.getByRole("button", { name: /Carregar mais eventos/i }));

        await waitFor(() => {
            expect(fetchDocumentAuditEvents).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    cursor: "cursor-doc-1",
                    query: "manual",
                    documentId: 7,
                })
            );
        });

        expect(await screen.findByText("retry failed")).toBeInTheDocument();
    });

    it("applies the failure quick filter from the summary shortcuts", async () => {
        vi.mocked(fetchDocumentAuditEvents)
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 1,
                        user_id: 9,
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
                    total_events: 3,
                    unique_documents: 1,
                    unique_collections: 1,
                    unique_requests: 1,
                    last_event_at: "2026-03-20T12:00:00Z",
                    action_breakdown: [{ key: "DOCUMENT_UPLOAD", count: 1 }],
                    status_breakdown: [{ key: "FAILED", count: 2 }],
                },
                page_info: {
                    limit: 12,
                    returned: 1,
                    has_next: false,
                },
            })
            .mockResolvedValueOnce({
                items: [
                    {
                        id: 2,
                        user_id: 9,
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
                    last_event_at: "2026-03-20T12:10:00Z",
                    action_breakdown: [{ key: "DOCUMENT_RETRY_INDEX", count: 1 }],
                    status_breakdown: [{ key: "FAILED", count: 2 }],
                },
                page_info: {
                    limit: 12,
                    returned: 1,
                    has_next: false,
                },
            });

        const user = userEvent.setup();

        render(<DocumentAuditExplorer locale="pt-BR" />);

        await screen.findByText("uploaded");

        await user.click(screen.getByRole("button", { name: /Somente falhas/i }));

        await waitFor(() => {
            expect(fetchDocumentAuditEvents).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    status: "FAILED",
                })
            );
        });

        expect(await screen.findByText("retry failed")).toBeInTheDocument();
    });

    it("preserva o termo digitado ao clicar num filtro rapido", async () => {
        // Cenario: o usuario digita no campo de busca e, ANTES de submeter,
        // clica no atalho "Somente falhas".
        //
        // A versao anterior montava o proximo estado como
        // `{...filters, ...appliedFilters, ...partial}` -- appliedFilters vinha
        // depois e reimpunha o valor vazio ja submetido, apagando o texto do
        // campo na frente do usuario e ignorando o termo na consulta.
        const page = {
            items: [
                {
                    id: 1,
                    user_id: 9,
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
                total_events: 1,
                unique_documents: 1,
                unique_collections: 1,
                unique_requests: 1,
                last_event_at: "2026-03-20T12:00:00Z",
                action_breakdown: [{ key: "DOCUMENT_UPLOAD", count: 1 }],
                status_breakdown: [{ key: "FAILED", count: 1 }],
            },
            page_info: { limit: 12, returned: 1, has_next: false },
        };
        vi.mocked(fetchDocumentAuditEvents).mockResolvedValue(page);

        const user = userEvent.setup();
        render(<DocumentAuditExplorer locale="pt-BR" />);

        await screen.findByText("uploaded");

        const searchField = screen.getByLabelText("Encontre um evento");
        await user.type(searchField, "contrato");
        expect(searchField).toHaveValue("contrato");

        await user.click(screen.getByRole("button", { name: /Somente falhas/i }));

        // O texto digitado continua no campo...
        await waitFor(() => {
            expect(searchField).toHaveValue("contrato");
        });

        // ...e vai junto na consulta, com o filtro rapido aplicado.
        await waitFor(() => {
            expect(fetchDocumentAuditEvents).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    status: "FAILED",
                    query: "contrato",
                })
            );
        });
    });
});
