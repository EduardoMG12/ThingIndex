import { describe, expect, it } from "vitest";

import { normalizeHighlight, normalizeHighlights, normalizeLegend } from "@/lib/api/contracts/highlights";

describe("normalizeHighlights", () => {
    it("aceita snake_case e camelCase e descarta cor fora da paleta ou id invalido", () => {
        const saida = normalizeHighlights({
            highlights: [
                { paragraph_node_id: 5, document_id: 2, color: "amarelo" },
                { paragraphNodeId: 6, documentId: 2, color: "roxo" },
                { paragraph_node_id: 7, document_id: 2, color: "magenta" },
                { paragraph_node_id: "8", document_id: 2, color: "verde" },
                null,
            ],
        });
        expect(saida).toEqual([
            { paragraphNodeId: 5, documentId: 2, color: "amarelo" },
            { paragraphNodeId: 6, documentId: 2, color: "roxo" },
        ]);
        expect(normalizeHighlight("x")).toBeNull();
        expect(normalizeHighlights(undefined)).toEqual([]);
    });
});

describe("normalizeLegend", () => {
    it("guarda so cores da paleta com rotulo nao vazio", () => {
        expect(normalizeLegend({ labels: { amarelo: "prazo", verde: "", magenta: "x", azul: 3 } })).toEqual({ amarelo: "prazo" });
        expect(normalizeLegend({ amarelo: "direto" })).toEqual({ amarelo: "direto" });
        expect(normalizeLegend(null)).toEqual({});
    });
});
