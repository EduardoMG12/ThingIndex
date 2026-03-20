import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/fetch-with-auth-redirect", () => ({
    fetchWithAuthRedirect: vi.fn(),
}));

import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import {
    deleteCollectionDocuments,
    downloadDocumentOriginal,
    fetchDocumentIndex,
    fetchDocumentMonitoringSummary,
    fetchDocuments,
    uploadDocument,
} from "@/lib/api/documents";

function createDocumentRecord(overrides: Record<string, unknown> = {}) {
    return {
        id: 1,
        user_id: 3,
        title: "case.pdf",
        file_path: "documents/case.pdf",
        size: 1024,
        mime_type: "application/pdf",
        status: "INDEXED",
        created_at: "2026-03-19T00:00:00Z",
        updated_at: "2026-03-19T00:00:00Z",
        ...overrides,
    };
}

describe("documents api", () => {
    beforeEach(() => {
        vi.mocked(fetchWithAuthRedirect).mockReset();
    });

    it("uploads documents with trimmed metadata and normalizes the response", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "uploaded",
                    data: createDocumentRecord({
                        file_url: "/documents/1",
                        collection: {
                            id: 9,
                            name: "Case A",
                            document_count: 5,
                            metadata: {
                                source: "manual",
                            },
                        },
                        category_id: 4,
                        category: {
                            id: 4,
                            user_id: 3,
                            name: "Contracts",
                            description: "",
                            created_at: "2026-03-19T00:00:00Z",
                            updated_at: "2026-03-19T00:00:00Z",
                        },
                        metadata_pairs: [
                            {
                                key: "document_summary",
                                value: "Key commercial clauses",
                            },
                        ],
                    }),
                }),
                {
                    status: 200,
                    headers: {
                        "Content-Type": "application/json",
                    },
                }
            )
        );

        const file = new File(["pdf"], "case.pdf", { type: "application/pdf" });
        const document = await uploadDocument({
            title: "case.pdf",
            file,
            category: "  Contracts  ",
            collection: "  Case A  ",
        });

        const request = vi.mocked(fetchWithAuthRedirect).mock.calls[0];
        const formData = request[1]?.body as FormData;

        expect(request[0]).toBe("/api/documents/upload");
        expect(formData.get("title")).toBe("case.pdf");
        expect(formData.get("file")).toBe(file);
        expect(formData.get("category")).toBe("Contracts");
        expect(formData.get("collection")).toBe("Case A");

        expect(document).toMatchObject({
            id: 1,
            file_url: "/documents/1",
            file_path: "documents/case.pdf",
            collection_id: 9,
            collection: "Case A",
            category_id: 4,
            document_summary: "Key commercial clauses",
        });
        expect(document.collection_ref).toMatchObject({
            id: 9,
            name: "Case A",
            documentCount: 5,
            metadata: {
                source: "manual",
            },
        });
    });

    it("normalizes fetched document lists, including legacy collection names", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: [
                        createDocumentRecord({
                            collection: " Legacy Collection ",
                        }),
                    ],
                }),
                {
                    status: 200,
                    headers: {
                        "Content-Type": "application/json",
                    },
                }
            )
        );

        const documents = await fetchDocuments();

        expect(documents).toHaveLength(1);
        expect(documents[0]).toMatchObject({
            collection: "Legacy Collection",
        });
        expect(documents[0].collection_ref).toMatchObject({
            id: "legacy:legacy collection",
            name: "Legacy Collection",
            documentCount: 0,
        });
    });

    it("returns a zero deletion count when the backend omits the payload", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "deleted",
                }),
                {
                    status: 200,
                    headers: {
                        "Content-Type": "application/json",
                    },
                }
            )
        );

        await expect(deleteCollectionDocuments(9)).resolves.toEqual({
            deleted_documents: 0,
        });
    });

    it("downloads the original file using the filename from content-disposition", async () => {
        vi.useFakeTimers();

        try {
            vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
                new Response("file-body", {
                    status: 200,
                    headers: {
                        "content-disposition": "attachment; filename*=UTF-8''relatorio%20final.pdf",
                    },
                })
            );

            const appendSpy = vi.spyOn(document.body, "appendChild");
            const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
            const createObjectURLSpy = vi.spyOn(window.URL, "createObjectURL").mockReturnValue("blob:test-download");
            const revokeObjectURLSpy = vi.spyOn(window.URL, "revokeObjectURL").mockImplementation(() => undefined);

            await downloadDocumentOriginal({
                id: 7,
                title: "fallback.pdf",
            });

            vi.runAllTimers();

            expect(clickSpy).toHaveBeenCalledTimes(1);
            expect(createObjectURLSpy).toHaveBeenCalledTimes(1);
            expect(revokeObjectURLSpy).toHaveBeenCalledWith("blob:test-download");

            const anchor = appendSpy.mock.calls[0][0] as HTMLAnchorElement;
            expect(anchor.href).toBe("blob:test-download");
            expect(anchor.download).toBe("relatorio final.pdf");
        } finally {
            vi.useRealTimers();
        }
    });

    it("normalizes document index payloads and monitoring metadata", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        document: createDocumentRecord({
                            collection: {
                                id: 11,
                                name: "Operations",
                                document_count: 2,
                            },
                        }),
                        paragraph_nodes: [
                            {
                                document_id: 1,
                                sequence_order: 1,
                                content: "Paragraph A",
                                page_number: 3,
                            },
                        ],
                        table_sections: [
                            {
                                document_id: 1,
                                sequence_order: 2,
                                title: "Table A",
                                headers: ["Clause", "Value"],
                                rows: [["Term", "12 months"]],
                            },
                        ],
                        metadata_pairs: [
                            {
                                document_id: 1,
                                sequence_order: 1,
                                key: "language",
                                value: "en",
                            },
                        ],
                        chunks: [
                            {
                                document_id: 1,
                                sequence_order: 1,
                                chunk_type: "paragraph",
                                content: "Chunk A",
                            },
                            {
                                document_id: "invalid",
                            },
                        ],
                        outline_nodes: [
                            {
                                document_id: 1,
                                sequence_order: 1,
                                level: 1,
                                title: "Overview",
                            },
                        ],
                        document_summary: "Summary A",
                        monitoring: {
                            status: "DONE",
                            stage: "POST_PROCESSING",
                            duration_ms: 3421,
                            tree_verification_passed: true,
                            estimated_quality_score: 0.93,
                            llm: {
                                provider_type: "OPENAI",
                                model: "gpt-4o-mini",
                                request_count: 2,
                                estimated_input_tokens: 1200,
                                estimated_output_tokens: 400,
                                estimated_total_tokens: 1600,
                                pricing_available: true,
                                total_cost_usd: 0.012,
                            },
                        },
                    },
                }),
                {
                    status: 200,
                    headers: {
                        "Content-Type": "application/json",
                    },
                }
            )
        );

        const index = await fetchDocumentIndex(1);

        expect(index.document).toMatchObject({
            id: 1,
            collection_id: 11,
            collection: "Operations",
        });
        expect(index.paragraph_nodes).toHaveLength(1);
        expect(index.table_sections).toHaveLength(1);
        expect(index.metadata_pairs).toHaveLength(1);
        expect(index.outline_nodes).toHaveLength(1);
        expect(index.chunks).toEqual([
            expect.objectContaining({
                id: "chunk:1:1",
                document_id: 1,
                sequence_order: 1,
                chunk_type: "paragraph",
                content: "Chunk A",
                token_estimate: 0,
                metadata: {},
            }),
        ]);
        expect(index.document_summary).toBe("Summary A");
        expect(index.monitoring).toMatchObject({
            status: "DONE",
            stage: "POST_PROCESSING",
            duration_ms: 3421,
            tree_verification_passed: true,
            estimated_quality_score: 0.93,
            llm: {
                provider_type: "OPENAI",
                model: "gpt-4o-mini",
                request_count: 2,
                estimated_input_tokens: 1200,
                estimated_output_tokens: 400,
                estimated_total_tokens: 1600,
                pricing_available: true,
                input_cost_usd: 0,
                output_cost_usd: 0,
                total_cost_usd: 0.012,
            },
        });
    });

    it("returns an empty monitoring summary when the backend payload is missing", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                }),
                {
                    status: 200,
                    headers: {
                        "Content-Type": "application/json",
                    },
                }
            )
        );

        await expect(fetchDocumentMonitoringSummary()).resolves.toEqual({
            total_documents: 0,
            indexed_documents: 0,
            processing_documents: 0,
            queued_documents: 0,
            failed_documents: 0,
            retrying_documents: 0,
            orphaned_documents: 0,
            dlq_documents: 0,
            documents_with_llm: 0,
            pricing_aware_documents: 0,
            total_llm_requests: 0,
            total_input_tokens: 0,
            total_output_tokens: 0,
            total_tokens: 0,
            total_estimated_cost_usd: 0,
            average_duration_ms: 0,
            average_quality_score: 0,
            last_completed_at: undefined,
            model_breakdown: [],
        });
    });
});
