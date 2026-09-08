import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/fetch-with-auth-redirect", () => ({
    fetchWithAuthRedirect: vi.fn(),
}));

import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import { deleteHighlight, fetchHighlights, fetchLegend, fetchViewState, putHighlight, putLegend, putViewState } from "@/lib/api/collections";

function resposta(status: number, corpo: unknown): Response {
    return { ok: status >= 200 && status < 300, status, json: async () => corpo } as unknown as Response;
}

describe("cliente de marcas e legenda", () => {
    beforeEach(() => {
        vi.mocked(fetchWithAuthRedirect).mockReset();
    });

    it("lista marcas pela rota da collection e normaliza", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(
            resposta(200, { status: 200, data: { highlights: [{ paragraph_node_id: 5, document_id: 2, color: "amarelo" }] } })
        );
        const marcas = await fetchHighlights(7);
        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0][0]).toBe("/api/documents/collections/7/highlights");
        expect(marcas).toEqual([{ paragraphNodeId: 5, documentId: 2, color: "amarelo" }]);
    });

    it("salva com PUT e corpo {color}, remove com DELETE", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(resposta(200, { data: { paragraph_node_id: 5, document_id: 2, color: "roxo" } }));
        await putHighlight(7, 5, "roxo");
        const [url, init] = vi.mocked(fetchWithAuthRedirect).mock.calls[0];
        expect(url).toBe("/api/documents/collections/7/highlights/5");
        expect(init?.method).toBe("PUT");
        expect(JSON.parse(String(init?.body))).toEqual({ color: "roxo" });

        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(resposta(200, { data: null }));
        await deleteHighlight(7, 5);
        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[1][1]?.method).toBe("DELETE");
    });

    it("legenda vai e volta como {labels}", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(resposta(200, { data: { labels: { amarelo: "prazo" }, colors: [] } }));
        expect(await fetchLegend(7)).toEqual({ amarelo: "prazo" });
        await putLegend(7, { verde: "requisito" });
        const [, init] = vi.mocked(fetchWithAuthRedirect).mock.calls[1];
        expect(JSON.parse(String(init?.body))).toEqual({ labels: { verde: "requisito" } });
    });

    it("erro da API vira Error com a mensagem da API", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(resposta(400, { message: "Cor invalida" }));
        await expect(putHighlight(7, 5, "amarelo")).rejects.toThrow("Cor invalida");
    });

    it("estado de visualizacao: GET normalizado e PUT com {state} e keepalive", async () => {
        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(resposta(200, { data: { state: { viewport: { x: 1, y: 2, escala: 0.5 } } } }));
        expect(await fetchViewState(7)).toEqual({ viewport: { x: 1, y: 2, escala: 0.5 } });
        expect(vi.mocked(fetchWithAuthRedirect).mock.calls[0][0]).toBe("/api/documents/collections/7/view-state");

        vi.mocked(fetchWithAuthRedirect).mockResolvedValue(resposta(200, { data: { state: {} } }));
        await putViewState(7, { marcadoresVisiveis: false }, { keepalive: true });
        const [, init, opcoes] = vi.mocked(fetchWithAuthRedirect).mock.calls[1];
        expect(init?.method).toBe("PUT");
        expect(JSON.parse(String(init?.body))).toEqual({ state: { marcadoresVisiveis: false } });
        expect(init?.keepalive).toBe(true);
        expect(opcoes).toEqual({ redirectOnUnauthorized: false });
    });
});
