import { describe, expect, it } from "vitest";

import { describeAggregatedCost } from "@/lib/aggregated-cost";

const base = { documents_with_llm: 0, pricing_aware_documents: 0, total_estimated_cost_usd: 0 };

describe("describeAggregatedCost: consumo nao informado nao e zero", () => {
    it("com documentos usando LLM e nenhum com preco, o custo e desconhecido (ex.: Ollama)", () => {
        expect(describeAggregatedCost({ ...base, documents_with_llm: 299, total_estimated_cost_usd: 0 })).toEqual({
            kind: "unknown",
        });
    });

    it("sem nenhum documento usando LLM, o custo e zero de verdade", () => {
        expect(describeAggregatedCost(base)).toEqual({ kind: "known", costUsd: 0 });
    });

    it("com preco em todos os documentos, o custo e conhecido", () => {
        expect(
            describeAggregatedCost({ documents_with_llm: 3, pricing_aware_documents: 3, total_estimated_cost_usd: 0.0134 })
        ).toEqual({ kind: "known", costUsd: 0.0134 });
    });

    it("com preco so em parte dos documentos, o valor e parcial e diz a cobertura", () => {
        expect(
            describeAggregatedCost({ documents_with_llm: 5, pricing_aware_documents: 2, total_estimated_cost_usd: 0.01 })
        ).toEqual({ kind: "partial", costUsd: 0.01, pricedDocuments: 2, documentsWithLlm: 5 });
    });
});
