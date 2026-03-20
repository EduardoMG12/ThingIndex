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

    it("loads document audit events with query filters and normalized metadata", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: [
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
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const events = await fetchDocumentAuditEvents({
            action: "DOCUMENT_UPLOAD",
            status: "SUCCESS",
            documentId: 21,
            collectionId: 8,
            limit: 5,
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe(
            "/api/documents/audit?limit=5&action=DOCUMENT_UPLOAD&status=SUCCESS&document_id=21&collection_id=8"
        );
        expect(events).toEqual([
            expect.objectContaining({
                id: 3,
                document_id: 21,
                collection_id: 8,
                action: "DOCUMENT_UPLOAD",
                metadata: {
                    route_path: "/api/v1/documents/upload",
                },
            }),
        ]);
    });

    it("loads provider audit events with normalized defaults", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: [
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
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const events = await fetchProviderAuditEvents({
            family: "LLM",
            providerType: "OPENAI",
            limit: 10,
        });

        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0]?.[0]).toBe(
            "/api/user/providers/audit?limit=10&family=LLM&provider_type=OPENAI"
        );
        expect(events).toEqual([
            expect.objectContaining({
                id: 5,
                provider_type: "OPENAI",
                actor_type: "USER",
                metadata: {},
            }),
        ]);
    });
});
