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

    it("remove da lista o documento que o polling nao devolve mais", async () => {
        // Um processando para o polling ligar; o 1 e excluido em outra aba.
        vi.useFakeTimers();

        try {
            vi.mocked(fetchDocuments)
                .mockResolvedValueOnce([
                    createDocument({ id: 1, title: "excluido.pdf" }),
                    createDocument({ id: 2, title: "processando.pdf", status: "PROCESSING" }),
                ])
                .mockResolvedValue([createDocument({ id: 2, title: "processando.pdf", status: "PROCESSING" })]);

            const { result } = renderHook(() => useDocuments());

            await vi.waitFor(() => {
                expect(result.current.isLoading).toBe(false);
            });
            expect(result.current.documents).toHaveLength(2);

            // Deixa uma rodada de polling acontecer.
            await act(async () => {
                await vi.advanceTimersByTimeAsync(3100);
            });

            await vi.waitFor(() => {
                expect(fetchDocuments).toHaveBeenCalledTimes(2);
            });

            // O documento 1 sumiu do servidor, entao precisa sumir da tela.
            expect(result.current.documents.map((doc) => doc.id)).toEqual([2]);
        } finally {
            vi.useRealTimers();
        }
    });
});

describe("useDocuments preserva referencias no polling (F-07)", () => {
    it("mantem o mesmo objeto quando o poll devolve o documento sem mudanca", async () => {
        vi.useFakeTimers();

        try {
            vi.mocked(fetchDocuments).mockReset().mockImplementation(async () => [
                createDocument({ id: 1, status: "COMPLETED" }),
                createDocument({ id: 2, status: "PROCESSING" }),
            ]);

            const { result } = renderHook(() => useDocuments());

            await vi.waitFor(() => {
                expect(result.current.isLoading).toBe(false);
            });
            const [antes1, antes2] = result.current.documents;

            await act(async () => {
                await vi.advanceTimersByTimeAsync(3100);
            });
            await vi.waitFor(() => {
                expect(fetchDocuments).toHaveBeenCalledTimes(2);
            });

            expect(result.current.documents[0]).toBe(antes1);
            expect(result.current.documents[1]).toBe(antes2);
        } finally {
            vi.useRealTimers();
        }
    });

    it("troca o objeto quando o servidor muda o documento", async () => {
        vi.useFakeTimers();

        try {
            vi.mocked(fetchDocuments)
                .mockReset()
                .mockResolvedValueOnce([createDocument({ id: 2, status: "PROCESSING" })])
                .mockResolvedValue([
                    createDocument({ id: 2, status: "COMPLETED", updated_at: "2026-03-20T00:00:00Z" }),
                ]);

            const { result } = renderHook(() => useDocuments());

            await vi.waitFor(() => {
                expect(result.current.isLoading).toBe(false);
            });
            const antes = result.current.documents[0];

            await act(async () => {
                await vi.advanceTimersByTimeAsync(3100);
            });
            await vi.waitFor(() => {
                expect(result.current.documents[0]?.status).toBe("COMPLETED");
            });

            expect(result.current.documents[0]).not.toBe(antes);
        } finally {
            vi.useRealTimers();
        }
    });
});
