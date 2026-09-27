import type { DocumentMonitoringSummary } from "@/lib/api/documents";

export type AggregatedCostView =
    | { kind: "known"; costUsd: number }
    | { kind: "partial"; costUsd: number; pricedDocuments: number; documentsWithLlm: number }
    | { kind: "unknown" };

// Consumo nao informado e categoria distinta de zero. O backend so calcula
// custo para documentos cujo modelo tem preco (pricing_available); para os
// demais (ex.: Ollama local) total_cost_usd fica 0 sem significar "gratis".
// - nenhum documento usou LLM: zero de verdade;
// - nenhum documento com preco: desconhecido ("Nao informado");
// - preco em parte deles: o valor e um piso, com a cobertura explicita.
export function describeAggregatedCost(
    summary: Pick<DocumentMonitoringSummary, "documents_with_llm" | "pricing_aware_documents" | "total_estimated_cost_usd">
): AggregatedCostView {
    const withLlm = summary.documents_with_llm;
    const priced = Math.min(summary.pricing_aware_documents, withLlm);

    if (withLlm === 0) {
        return { kind: "known", costUsd: 0 };
    }
    if (priced === 0) {
        return { kind: "unknown" };
    }
    if (priced < withLlm) {
        return { kind: "partial", costUsd: summary.total_estimated_cost_usd, pricedDocuments: priced, documentsWithLlm: withLlm };
    }
    return { kind: "known", costUsd: summary.total_estimated_cost_usd };
}
