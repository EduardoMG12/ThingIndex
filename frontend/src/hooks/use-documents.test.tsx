import { renderHook, waitFor, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/documents", () => ({
    fetchDocuments: vi.fn(),
}));

import { useDocuments } from "@/hooks/use-documents";
import { fetchDocuments } from "@/lib/api/documents";

function createDocument(overrides: Record<string, unknown> = {}) {
    return {
        id: 1,
        user_id: 2,
        title: "case.pdf",
        file_url: "/documents/1",
        file_path: "documents/case.pdf",
        size: 1024,
        mime_type: "application/pdf",
        status: "INDEXED",
        created_at: "2026-03-19T00:00:00Z",
        updated_at: "2026-03-19T00:00:00Z",
        ...overrides,
    };
}

describe("useDocuments", () => {
    beforeEach(() => {
        vi.mocked(fetchDocuments).mockReset();
    });

    it("loads documents on mount", async () => {
        vi.mocked(fetchDocuments).mockResolvedValue([createDocument()]);

        const { result } = renderHook(() => useDocuments());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(fetchDocuments).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(result.current.error).toBeNull();
        expect(result.current.documents).toEqual([createDocument()]);
    });

    it("reloads documents after an initial failure", async () => {
        vi.mocked(fetchDocuments)
            .mockRejectedValueOnce(new Error("backend unavailable"))
            .mockResolvedValueOnce([createDocument({ id: 2, title: "updated.pdf" })]);

        const { result } = renderHook(() => useDocuments());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(result.current.error).toBe("backend unavailable");
        expect(result.current.documents).toEqual([]);

        await act(async () => {
            await result.current.reloadDocuments();
        });

        await waitFor(() => {
            expect(result.current.error).toBeNull();
        });

        expect(fetchDocuments).toHaveBeenNthCalledWith(2, undefined);
        expect(result.current.documents).toEqual([createDocument({ id: 2, title: "updated.pdf" })]);
    });
});
