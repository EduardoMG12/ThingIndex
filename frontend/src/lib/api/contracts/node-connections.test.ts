import { describe, expect, it } from "vitest";

import { normalizeNodeConnections } from "@/lib/api/contracts/node-connections";

describe("normalizeNodeConnections", () => {
    it("normaliza a resposta da API", () => {
        const resultado = normalizeNodeConnections({
            node_id: "node-001-penalidades",
            min_score: 2,
            connections: [
                {
                    document_id: 35,
                    node_id: "node-001-das-penalidades",
                    title: "2 DAS PENALIDADES",
                    summary: "A multa por descumprimento sera de 5 por cento.",
                    start_page: 1,
                    score: 11,
                    matched_terms: ["multa", "fatura", "descumprimento"],
                },
            ],
        });

        expect(resultado.nodeId).toBe("node-001-penalidades");
        expect(resultado.minScore).toBe(2);
        expect(resultado.connections).toHaveLength(1);
        expect(resultado.connections[0]).toMatchObject({
            documentId: 35,
            nodeId: "node-001-das-penalidades",
            score: 11,
        });
        expect(resultado.connections[0].matchedTerms).toEqual([
            "multa",
            "fatura",
            "descumprimento",
        ]);
    });

    it("trata lista vazia como resultado legitimo, nao como erro", () => {
        // Lista vazia significa "nenhuma conexao supera o limiar" -- e a
        // resposta CORRETA quando nao ha relacao real. Tratar como erro faria a
        // interface esconder justamente o comportamento que o trabalho defende:
        // saber dizer "nao encontrei".
        const resultado = normalizeNodeConnections({
            node_id: "no-isolado",
            min_score: 99,
            connections: [],
        });

        expect(resultado.connections).toEqual([]);
        expect(resultado.minScore).toBe(99);
    });

    it("descarta conexao sem identificador em vez de renderizar item quebrado", () => {
        const resultado = normalizeNodeConnections({
            node_id: "origem",
            min_score: 2,
            connections: [
                { document_id: 1, node_id: "", score: 5, matched_terms: [] },
                { document_id: 2, node_id: "valido", score: 4, matched_terms: ["termo"] },
            ],
        });

        expect(resultado.connections).toHaveLength(1);
        expect(resultado.connections[0].nodeId).toBe("valido");
    });

    it("sobrevive a payload malformado sem explodir a tela", () => {
        expect(normalizeNodeConnections(null).connections).toEqual([]);
        expect(normalizeNodeConnections(undefined).connections).toEqual([]);
        expect(normalizeNodeConnections("texto").connections).toEqual([]);
        expect(normalizeNodeConnections({ connections: "nao e array" }).connections).toEqual([]);
    });

    it("preserva a ordem devolvida pela API", () => {
        // A API ja ordena por score decrescente. Reordenar no cliente poderia
        // divergir do backend e quebrar o determinismo que o trabalho afirma.
        const resultado = normalizeNodeConnections({
            node_id: "origem",
            min_score: 1,
            connections: [
                { document_id: 1, node_id: "a", score: 9, matched_terms: [] },
                { document_id: 2, node_id: "b", score: 3, matched_terms: [] },
                { document_id: 3, node_id: "c", score: 7, matched_terms: [] },
            ],
        });

        expect(resultado.connections.map((c) => c.nodeId)).toEqual(["a", "b", "c"]);
    });

    it("ignora termos que nao sao string", () => {
        const resultado = normalizeNodeConnections({
            node_id: "origem",
            min_score: 2,
            connections: [
                {
                    document_id: 1,
                    node_id: "alvo",
                    score: 3,
                    matched_terms: ["valido", 42, null, "outro"],
                },
            ],
        });

        expect(resultado.connections[0].matchedTerms).toEqual(["valido", "outro"]);
    });
});
