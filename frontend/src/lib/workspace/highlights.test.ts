import { describe, expect, it } from "vitest";

import {
    APRESENTACAO_PADRAO,
    classesDaCor,
    contagemPorCor,
    CORES,
    ehCor,
    limitarPagina,
    precisaDoBotaoIrPara,
    regraIrAte,
    rotuloDaCor,
} from "@/lib/workspace/highlights";

const nome = (cor: string) => `cor ${cor}`;

describe("paleta", () => {
    it("tem cinco cores com classes literais", () => {
        expect(CORES).toHaveLength(5);
        for (const cor of CORES) {
            const c = classesDaCor(cor);
            expect(c.grifo).toMatch(/^bg-\S+ border-\S+$/);
            expect(c.chip).toMatch(/^bg-/);
            expect(c.secundario).toMatch(/^bg-\S+ border-\S+$/);
        }
        expect(ehCor("amarelo")).toBe(true);
        expect(ehCor("magenta")).toBe(false);
        expect(ehCor(3)).toBe(false);
    });
});

describe("rotuloDaCor", () => {
    it("usa a legenda quando ha rotulo e o nome da cor quando esta vazio ou so espacos", () => {
        expect(rotuloDaCor({ amarelo: "prazo" }, "amarelo", nome)).toBe("prazo");
        expect(rotuloDaCor({ amarelo: "   " }, "amarelo", nome)).toBe("cor amarelo");
        expect(rotuloDaCor({}, "roxo", nome)).toBe("cor roxo");
    });
});

describe("contagemPorCor", () => {
    it("conta so as marcas do documento pedido, com zero nas cores sem marca", () => {
        const marcas = [
            { paragraphNodeId: 1, documentId: 7, color: "amarelo" as const },
            { paragraphNodeId: 2, documentId: 7, color: "amarelo" as const },
            { paragraphNodeId: 3, documentId: 8, color: "verde" as const },
        ];
        expect(contagemPorCor(marcas, 7)).toEqual({ amarelo: 2, verde: 0, azul: 0, vermelho: 0, roxo: 0 });
        expect(contagemPorCor([], 7).azul).toBe(0);
    });
});

describe("regraIrAte e precisaDoBotaoIrPara", () => {
    it("em 'pagina' troca a pagina quando o destino e outra; em 'folhas' so centraliza", () => {
        expect(regraIrAte({ modo: "pagina", paginaAtual: 1 }, 7)).toEqual({ trocarPagina: 7, centralizar: true });
        expect(regraIrAte({ modo: "pagina", paginaAtual: 7 }, 7)).toEqual({ trocarPagina: null, centralizar: true });
        expect(regraIrAte(APRESENTACAO_PADRAO, 7)).toEqual({ trocarPagina: null, centralizar: true });
    });

    it("o botao aparece so quando a pagina destino nao esta a vista", () => {
        const visiveis = new Set([1, 2]);
        expect(precisaDoBotaoIrPara({ modo: "folhas", paginaAtual: 1 }, visiveis, 2)).toBe(false);
        expect(precisaDoBotaoIrPara({ modo: "folhas", paginaAtual: 1 }, visiveis, 7)).toBe(true);
        expect(precisaDoBotaoIrPara({ modo: "pagina", paginaAtual: 7 }, visiveis, 7)).toBe(false);
        expect(precisaDoBotaoIrPara({ modo: "pagina", paginaAtual: 1 }, visiveis, 7)).toBe(true);
        expect(precisaDoBotaoIrPara({ modo: "pagina", paginaAtual: 1 }, visiveis, null)).toBe(false);
    });

    it("limitarPagina prende ao intervalo e trunca decimais", () => {
        expect(limitarPagina(0, 30)).toBe(1);
        expect(limitarPagina(31, 30)).toBe(30);
        expect(limitarPagina(7.9, 30)).toBe(7);
        expect(limitarPagina(5, 0)).toBe(1);
    });
});
