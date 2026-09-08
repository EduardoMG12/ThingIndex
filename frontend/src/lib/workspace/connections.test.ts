import { describe, expect, it } from "vitest";

import type { OutlineNode, ParagraphNode } from "@/lib/api/contracts/document-context";
import type { NodeConnection } from "@/lib/api/contracts/node-connections";
import {
    caminhoDaLinha,
    espessuraPorScore,
    montarLinhas,
    paragrafosSecundarios,
    rotuloDoScore,
} from "@/lib/workspace/connections";

const SUMARIO: OutlineNode[] = [
    { id: 1, nodeId: "node-001-penalidades", documentId: 2, sequenceOrder: 1, level: 1, title: "2 DAS PENALIDADES", slug: "penalidades" },
    { id: 2, nodeId: "node-002-prazos", documentId: 2, sequenceOrder: 2, level: 1, title: "3 PRAZOS", slug: "prazos" },
];

const PARAGRAFOS: ParagraphNode[] = [
    { id: 10, documentId: 2, sequenceOrder: 0, content: "a", metadata: { section_key: "penalidades" } },
    { id: 11, documentId: 2, sequenceOrder: 1, content: "b", metadata: { section_key: "penalidades" } },
    { id: 12, documentId: 2, sequenceOrder: 2, content: "c", metadata: { section_key: "prazos" } },
    { id: 13, documentId: 1, sequenceOrder: 0, content: "d", metadata: { section_key: "penalidades" } },
];

const CONEXAO: NodeConnection = {
    documentId: 2,
    nodeId: "node-001-penalidades",
    title: "2 DAS PENALIDADES",
    summary: "",
    startPage: 1,
    score: 3,
    matchedTerms: ["multa", "fatura", "descumprimento"],
};

describe("paragrafosSecundarios", () => {
    it("marca so os paragrafos da secao destino, no documento destino", () => {
        const ids = paragrafosSecundarios([CONEXAO], SUMARIO, PARAGRAFOS);
        expect([...ids].sort()).toEqual([10, 11]);
    });

    it("sem conexao, conjunto vazio", () => {
        expect(paragrafosSecundarios([], SUMARIO, PARAGRAFOS).size).toBe(0);
    });
});

describe("montarLinhas", () => {
    const origem = { x: 0, y: 100, largura: 400, altura: 20 };

    it("usa o rect preciso quando existe e marca como aproximada quando cai no quadro", () => {
        const linhas = montarLinhas([CONEXAO, { ...CONEXAO, nodeId: "node-002-prazos", score: 2 }], {
            origem,
            destinoPreciso: (c) => (c.nodeId === "node-001-penalidades" ? { x: 800, y: 300, largura: 60, altura: 12 } : null),
            quadroDoDocumento: () => ({ x: 760, y: 0, largura: 612, altura: 5000 }),
        });
        expect(linhas).toHaveLength(2);
        expect(linhas[0].aproximada).toBe(false);
        expect(linhas[0].de).toEqual({ x: 400, y: 110 });
        // Destino bem abaixo e estreito: a linha entra por cima do marcador.
        expect(linhas[0].para).toEqual({ x: 830, y: 300 });
        expect(linhas[1].aproximada).toBe(true);
        expect(linhas[1].para.x).toBe(760);
    });

    it("sem origem nao desenha nada; sem destino conhecido pula a conexao", () => {
        expect(montarLinhas([CONEXAO], { origem: null, destinoPreciso: () => null, quadroDoDocumento: () => null })).toEqual([]);
        expect(montarLinhas([CONEXAO], { origem, destinoPreciso: () => null, quadroDoDocumento: () => null })).toEqual([]);
    });
});

describe("rotulos e espessura", () => {
    it("fala em termos, nunca em porcentagem, e engrossa com o score ate um teto", () => {
        expect(rotuloDoScore(1)).toBe("1 termo");
        expect(rotuloDoScore(3)).toBe("3 termos");
        expect(espessuraPorScore(2)).toBe(2.5);
        expect(espessuraPorScore(60)).toBe(espessuraPorScore(6));
    });

    it("caminho e uma cubica que comeca e termina nos pontos", () => {
        const d = caminhoDaLinha({ x: 0, y: 0 }, { x: 200, y: 100 });
        expect(d.startsWith("M 0 0 C ")).toBe(true);
        expect(d.endsWith(", 200 100")).toBe(true);
    });
});
