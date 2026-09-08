import { describe, expect, it } from "vitest";

import { normalizeViewState } from "@/lib/api/contracts/view-state";

describe("normalizeViewState", () => {
    it("aceita o envelope {state} ou o objeto direto e descarta o que e invalido", () => {
        const saida = normalizeViewState({
            state: {
                viewport: { x: 10, y: -5, escala: 0.75 },
                quadros: { "3": { x: 0, y: 0 }, "4": { x: "a", y: 1 } },
                apresentacao: { "3": { modo: "pagina", paginaAtual: 4.7 }, "4": { modo: "grade", paginaAtual: 1 }, "5": { modo: "folhas", paginaAtual: 0 } },
                marcadoresVisiveis: false,
                documentoFocado: 3,
                lixo: true,
            },
        });
        expect(saida).toEqual({
            viewport: { x: 10, y: -5, escala: 0.75 },
            quadros: { "3": { x: 0, y: 0 } },
            apresentacao: { "3": { modo: "pagina", paginaAtual: 4 } },
            marcadoresVisiveis: false,
            documentoFocado: 3,
        });
        expect(normalizeViewState({ viewport: { x: 1, y: 2, escala: 0 } })).toEqual({});
        expect(normalizeViewState(null)).toEqual({});
    });
});
