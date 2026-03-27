import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/fetch-with-auth-redirect", () => ({
    fetchWithAuthRedirect: vi.fn(),
}));

import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import { fetchDocumentAuditEvents, fetchProviderAuditEvents } from "@/lib/api/audit";

describe("audit api", () => {
    beforeEach(() => {
        vi.mocked(fetchWithAuthRedirect).mockReset();
    });

    it("loads document audit queries with filters, summary and pagination", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        items: [
                            {
                                id: 3,
                                user_id: 9,
                                document_id: 21,
                                collection_id: 8,
                                actor_type: "USER",
                                action: "DOCUMENT_UPLOAD",
                                status: "SUCCESS",
                                title: "case.pdf",
                                metadata: {
                                    route_path: "/api/v1/documents/upload",
                                },
                                created_at: "2026-03-20T12:00:00Z",
                            },
                        ],
                        summary: {
                            total_events: 4,
                            unique_documents: 2,
                            unique_collections: 1,
                            unique_requests: 3,
                            action_breakdown: [
                                { key: "DOCUMENT_UPLOAD", count: 2 },
                            ],
                            status_breakdown: [
                                { key: "SUCCESS", count: 3 },
                            ],
                        },
                        page_info: {
                            limit: 5,
                            returned: 1,
                            has_next: true,
                            next_cursor: "cursor-123",
                        },
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const result = await fetchDocumentAuditEvents({
            action: "DOCUMENT_UPLOAD",
            status: "SUCCESS",
            actorType: "USER",
            actorId: 9,
            query: "case",
            correlationId: "req-77",
            documentId: 21,
            collectionId: 8,
            limit: 5,
            cursor: "cursor-001",
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe(
            "/api/documents/audit?limit=5&cursor=cursor-001&q=case&action=DOCUMENT_UPLOAD&status=SUCCESS&actor_type=USER&actor_id=9&correlation_id=req-77&document_id=21&collection_id=8"
        );
        expect(result).toEqual({
            items: [
                expect.objectContaining({
                    id: 3,
                    document_id: 21,
                    collection_id: 8,
                    action: "DOCUMENT_UPLOAD",
                    metadata: {
                        route_path: "/api/v1/documents/upload",
                    },
                }),
            ],
            summary: expect.objectContaining({
                total_events: 4,
                unique_documents: 2,
                action_breakdown: [{ key: "DOCUMENT_UPLOAD", count: 2 }],
            }),
            page_info: {
                limit: 5,
                returned: 1,
                has_next: true,
                next_cursor: "cursor-123",
            },
        });
    });

    it("loads provider audit queries with normalized defaults", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        items: [
                            {
                                id: 5,
                                user_id: 7,
                                family: "LLM",
                                provider_type: "OPENAI",
                                action: "PROVIDER_ACTIVATION_CHANGE",
                                status: "SUCCESS",
                                created_at: "2026-03-20T12:00:00Z",
                            },
                        ],
                        summary: {
                            total_events: 2,
                            unique_providers: 1,
                            unique_requests: 2,
                        },
                        page_info: {
                            limit: 10,
                            returned: 1,
                            has_next: false,
                        },
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const result = await fetchProviderAuditEvents({
            family: "LLM",
            providerType: "OPENAI",
            query: "openai",
            actorType: "USER",
            actorId: 52,
            limit: 10,
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe(
            "/api/user/providers/audit?limit=10&q=openai&family=LLM&provider_type=OPENAI&actor_type=USER&actor_id=52"
        );
        expect(result).toEqual({
            items: [
                expect.objectContaining({
                    id: 5,
                    provider_type: "OPENAI",
                    actor_type: "USER",
                    metadata: {},
                }),
            ],
            summary: expect.objectContaining({
                total_events: 2,
                unique_providers: 1,
            }),
            page_info: {
                limit: 10,
                returned: 1,
                has_next: false,
                next_cursor: undefined,
            },
        });
    });
});
