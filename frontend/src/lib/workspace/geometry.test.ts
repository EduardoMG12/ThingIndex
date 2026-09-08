import { describe, expect, it } from "vitest";

import {
    ancorasDeLinha,
    aplicarZoom,
    boxParaPixels,
    centralizarEm,
    enquadrar,
    rectVisivel,
    telaParaMundo,
} from "@/lib/workspace/geometry";

describe("boxParaPixels", () => {
    it("converte pontos PDF em pixels proporcionalmente ao tamanho renderizado", () => {
        const box = { xMin: 72, xMax: 603, yMin: 85.9, yMax: 98.9, pageWidth: 612, pageHeight: 792 };
        const px = boxParaPixels(box, 1224, 1584); // 2x
        expect(px).toEqual({ left: 144, top: 171.8, width: 1062, height: 26 });
        expect(px.left / 1224).toBeCloseTo(72 / 612);
    });
});

describe("aplicarZoom", () => {
    it("mantem o ponto de mundo sob o cursor", () => {
        const antes = { x: 40, y: -20, escala: 1 };
        const cursor = { x: 100, y: 100 };
        const mundoAntes = telaParaMundo(cursor, antes);
        const depois = aplicarZoom(antes, 1.5, cursor);
        expect(depois.escala).toBe(1.5);
        const mundoDepois = telaParaMundo(cursor, depois);
        expect(mundoDepois.x).toBeCloseTo(mundoAntes.x);
        expect(mundoDepois.y).toBeCloseTo(mundoAntes.y);
    });

    it("respeita os limites e nao move o viewport quando o zoom nao muda", () => {
        const noTeto = { x: 0, y: 0, escala: 4 };
        expect(aplicarZoom(noTeto, 2, { x: 10, y: 10 })).toBe(noTeto);
        expect(aplicarZoom({ x: 0, y: 0, escala: 0.2 }, 0.1, { x: 0, y: 0 }).escala).toBe(0.1);
    });
});

describe("enquadrar e centralizarEm", () => {
    const quadros = [
        { x: 0, y: 0, largura: 300, altura: 200 },
        { x: 900, y: 500, largura: 300, altura: 400 },
    ];
    const tela = { largura: 800, altura: 600 };

    it("deixa todos os quadros visiveis com margem", () => {
        const vp = enquadrar(quadros, tela, 40);
        for (const q of quadros) {
            expect(rectVisivel(q, vp, tela)).toBe(true);
        }
        // Uniao 1200x900 em 720x520 -> escala limitada pela altura.
        expect(vp.escala).toBeCloseTo(Math.min(720 / 1200, 520 / 900));
    });

    it("poe o centro do rect no centro da tela", () => {
        const vp = centralizarEm({ x: 100, y: 100, largura: 200, altura: 100 }, tela, 1);
        expect(vp.x + 200 * vp.escala).toBe(400);
        expect(vp.y + 150 * vp.escala).toBe(300);
    });
});

describe("ancorasDeLinha", () => {
    it("sai pela borda voltada ao destino e entra pelo lado mais proximo", () => {
        const origem = { x: 0, y: 0, largura: 100, altura: 20 };
        const destinoADireita = { x: 500, y: 0, largura: 100, altura: 20 };
        const { de, para } = ancorasDeLinha(origem, destinoADireita);
        expect(de).toEqual({ x: 100, y: 10 });
        expect(para).toEqual({ x: 500, y: 10 });

        const destinoAbaixo = { x: 0, y: 400, largura: 100, altura: 20 };
        const baixo = ancorasDeLinha(origem, destinoAbaixo);
        expect(baixo.para).toEqual({ x: 50, y: 400 });
    });
});
