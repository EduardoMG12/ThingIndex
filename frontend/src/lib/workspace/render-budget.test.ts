import { describe, expect, it } from "vitest";

import { escalaDeRenderizacao, escolherPaginasVivas } from "@/lib/workspace/render-budget";

const tela = { largura: 1000, altura: 800 };
// 3 documentos x 30 paginas empilhadas, quadros lado a lado.
const paginas = Array.from({ length: 90 }, (_, i) => ({
    id: `d${Math.floor(i / 30)}-p${i % 30}`,
    rect: { x: Math.floor(i / 30) * 700, y: (i % 30) * 820, largura: 612, altura: 792 },
}));

describe("escolherPaginasVivas", () => {
    it("nunca passa do teto e so inclui pagina visivel", () => {
        const vivas = escolherPaginasVivas(paginas, { x: 0, y: 0, escala: 0.5 }, tela);
        expect(vivas.size).toBeLessThanOrEqual(12);
        expect(vivas.size).toBeGreaterThan(0);
        for (const id of vivas) {
            const p = paginas.find((c) => c.id === id)!;
            expect(p.rect.y * 0.5).toBeLessThan(tela.altura);
        }
    });

    it("prioriza as paginas mais proximas do centro da tela", () => {
        const vp = { x: 0, y: -820 * 10 * 0.5, escala: 0.5 };
        const vivas = escolherPaginasVivas(paginas, vp, tela, 2);
        // Linhas 10 e 11 sao as que cruzam o centro; a coluna vencedora e a
        // mais proxima do centro horizontal, nao importa qual.
        expect([...vivas].every((id) => /-p1[01]$/.test(id))).toBe(true);
        expect(vivas.size).toBe(2);
    });

    it("abaixo do zoom minimo nao renderiza nada", () => {
        expect(escolherPaginasVivas(paginas, { x: 0, y: 0, escala: 0.3 }, tela).size).toBe(0);
    });
});

describe("escalaDeRenderizacao", () => {
    it("limita zoom e DPR a 2", () => {
        expect(escalaDeRenderizacao(1, 1)).toBe(1);
        expect(escalaDeRenderizacao(4, 3)).toBe(4);
        expect(escalaDeRenderizacao(0.5, 2)).toBe(1);
    });
});
