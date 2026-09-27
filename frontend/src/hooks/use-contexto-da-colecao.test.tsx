import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/collections", () => ({
    fetchCollectionContext: vi.fn(),
    fetchHighlights: vi.fn(),
    fetchLegend: vi.fn(),
    fetchViewState: vi.fn(),
}));

import { useContextoDaColecao } from "@/hooks/use-contexto-da-colecao";
import { fetchCollectionContext, fetchHighlights, fetchLegend, fetchViewState } from "@/lib/api/collections";

type Pendente = { resolve: (valor: unknown) => void; signal: AbortSignal; collectionId: number };

function contexto(id: number) {
    return { collection: { id }, documents: [] } as never;
}

describe("useContextoDaColecao (F-08)", () => {
    let pendentes: Pendente[];

    beforeEach(() => {
        pendentes = [];
        vi.mocked(fetchCollectionContext).mockReset().mockImplementation((collectionId, signal) => {
            return new Promise((resolve) => {
                pendentes.push({ resolve, signal: signal!, collectionId: Number(collectionId) });
            }) as never;
        });
        vi.mocked(fetchHighlights).mockReset().mockResolvedValue([]);
        vi.mocked(fetchLegend).mockReset().mockResolvedValue({});
        vi.mocked(fetchViewState).mockReset().mockResolvedValue({});
    });

    it("carrega o contexto da colecao selecionada", async () => {
        const { result } = renderHook(() => useContextoDaColecao(1, "erro"));

        await waitFor(() => expect(pendentes).toHaveLength(1));
        await act(async () => pendentes[0]!.resolve(contexto(1)));

        await waitFor(() => expect(result.current.context).toEqual(contexto(1)));
        expect(result.current.carregando).toBe(false);
    });

    it("recarregar aborta a requisicao anterior ainda pendente", async () => {
        const { result } = renderHook(() => useContextoDaColecao(1, "erro"));
        await waitFor(() => expect(pendentes).toHaveLength(1));

        act(() => result.current.recarregar());
        await waitFor(() => expect(pendentes).toHaveLength(2));

        expect(pendentes[0]!.signal.aborted).toBe(true);
        expect(pendentes[1]!.signal.aborted).toBe(false);
    });

    it("descarta a resposta de uma recarga da colecao antiga que chega depois da troca", async () => {
        const { result, rerender } = renderHook(({ id }) => useContextoDaColecao(id, "erro"), {
            initialProps: { id: 1 },
        });
        await waitFor(() => expect(pendentes).toHaveLength(1));
        await act(async () => pendentes[0]!.resolve(contexto(1)));

        // Recarga silenciosa da colecao A (tick de polling) fica pendurada...
        act(() => result.current.recarregar());
        await waitFor(() => expect(pendentes).toHaveLength(2));

        // ...a pessoa troca para a colecao B, que responde primeiro.
        rerender({ id: 2 });
        await waitFor(() => expect(pendentes).toHaveLength(3));
        expect(pendentes[1]!.signal.aborted).toBe(true);
        await act(async () => pendentes[2]!.resolve(contexto(2)));
        await waitFor(() => expect(result.current.context).toEqual(contexto(2)));

        // A resposta velha da colecao A chega agora e nao pode sobrescrever B.
        await act(async () => pendentes[1]!.resolve(contexto(1)));

        expect(result.current.context).toEqual(contexto(2));
    });

    it("aborta a recarga pendente ao desmontar", async () => {
        const { result, unmount } = renderHook(() => useContextoDaColecao(1, "erro"));
        await waitFor(() => expect(pendentes).toHaveLength(1));
        act(() => result.current.recarregar());
        await waitFor(() => expect(pendentes).toHaveLength(2));

        unmount();

        expect(pendentes.every((p) => p.signal.aborted)).toBe(true);
    });
});

describe("useContextoDaColecao estados de carga", () => {
    it("recarga silenciosa nao volta a marcar carregando", async () => {
        const pendentes: Array<(valor: unknown) => void> = [];
        vi.mocked(fetchCollectionContext).mockReset().mockImplementation(() => {
            return new Promise((resolve) => pendentes.push(resolve)) as never;
        });
        const { result } = renderHook(() => useContextoDaColecao(1, "erro"));
        expect(result.current.carregando).toBe(true);
        await waitFor(() => expect(pendentes).toHaveLength(1));
        await act(async () => pendentes[0]!({ collection: { id: 1 }, documents: [] }));
        await waitFor(() => expect(result.current.carregando).toBe(false));

        act(() => result.current.recarregar());

        expect(result.current.carregando).toBe(false);
    });

    it("expoe a mensagem de erro so para a colecao que falhou", async () => {
        vi.mocked(fetchCollectionContext).mockReset().mockRejectedValue(new Error("fora do ar"));
        const { result } = renderHook(() => useContextoDaColecao(1, "erro"));

        await waitFor(() => expect(result.current.erro).toBe("fora do ar"));
        expect(result.current.carregando).toBe(false);
    });
});
