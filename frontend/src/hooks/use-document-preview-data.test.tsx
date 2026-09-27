import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/documents", () => ({
    fetchDocumentIndex: vi.fn(),
}));

vi.mock("@/lib/api/collections", () => ({
    fetchCollectionContext: vi.fn(),
}));

import { useDocumentPreviewData } from "@/hooks/use-document-preview-data";
import { fetchCollectionContext } from "@/lib/api/collections";
import { fetchDocumentIndex, type Document } from "@/lib/api/documents";

function criarDocumento(overrides: Partial<Document> = {}): Document {
    return {
        id: 7,
        user_id: 1,
        title: "lei.pdf",
        file_url: "/documents/7",
        file_path: "documents/lei.pdf",
        size: 10,
        mime_type: "application/pdf",
        status: "COMPLETED",
        collection_id: 3,
        created_at: "2026-09-01T00:00:00Z",
        updated_at: "2026-09-01T00:00:00Z",
        ...overrides,
    };
}

describe("useDocumentPreviewData (F-07)", () => {
    beforeEach(() => {
        vi.mocked(fetchDocumentIndex).mockReset().mockResolvedValue({ document: { status: "COMPLETED" } } as never);
        vi.mocked(fetchCollectionContext).mockReset().mockResolvedValue({ collection: { id: 3 } } as never);
    });

    it("carrega indice e contexto do documento selecionado", async () => {
        const { result } = renderHook(() => useDocumentPreviewData(criarDocumento(), "en"));

        await waitFor(() => expect(result.current.isLoading).toBe(false));
        expect(fetchDocumentIndex).toHaveBeenCalledWith(7, expect.any(AbortSignal));
        expect(fetchCollectionContext).toHaveBeenCalledWith(3, expect.any(AbortSignal));
        expect(result.current.documentIndex).not.toBeNull();
    });

    it("nao refaz o fetch quando o poll entrega um objeto novo do mesmo documento sem mudanca", async () => {
        const { result, rerender } = renderHook(({ doc }) => useDocumentPreviewData(doc, "en"), {
            initialProps: { doc: criarDocumento() },
        });
        await waitFor(() => expect(result.current.isLoading).toBe(false));

        rerender({ doc: criarDocumento() });
        rerender({ doc: criarDocumento({ title: "lei.pdf" }) });

        expect(result.current.isLoading).toBe(false);
        expect(fetchDocumentIndex).toHaveBeenCalledTimes(1);
        expect(fetchCollectionContext).toHaveBeenCalledTimes(1);
    });

    it("refaz o fetch quando o status ou o updated_at do documento mudam", async () => {
        const { result, rerender } = renderHook(({ doc }) => useDocumentPreviewData(doc, "en"), {
            initialProps: { doc: criarDocumento({ status: "PROCESSING" }) },
        });
        await waitFor(() => expect(result.current.isLoading).toBe(false));

        rerender({ doc: criarDocumento({ status: "COMPLETED", updated_at: "2026-09-02T00:00:00Z" }) });

        await waitFor(() => expect(fetchDocumentIndex).toHaveBeenCalledTimes(2));
    });
});
