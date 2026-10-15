// Conexoes recalculadas pela API a cada chamada (nao ha tabela de arestas).
// Mesmos parametros, mesmo resultado.

export type NodeConnection = {
    documentId: number;
    nodeId: string;
    title: string;
    summary: string;
    startPage: number | null;
    score: number;
    /** Termos da origem encontrados no destino. */
    matchedTerms: string[];
    /** "termo-raro" (padrao) ou "mesma-norma": os dois trechos citam a mesma lei/resolucao. */
    kind: "termo-raro" | "mesma-norma" | "mesma-disciplina";
};

export type NodeConnectionsResult = {
    nodeId: string;
    minScore: number;
    connections: NodeConnection[];
};

function asRecord(value: unknown): Record<string, unknown> | null {
    if (typeof value !== "object" || value === null || Array.isArray(value)) {
        return null;
    }
    return value as Record<string, unknown>;
}

function asString(value: unknown): string {
    return typeof value === "string" ? value : "";
}

function asNumber(value: unknown): number {
    return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function asNullableNumber(value: unknown): number | null {
    return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function normalizeConnection(value: unknown): NodeConnection | null {
    const record = asRecord(value);
    if (!record) {
        return null;
    }

    const nodeId = asString(record.node_id).trim();
    if (!nodeId) {
        // Sem identificador nao ha como focar o no ao clicar; renderizar um
        // item que nao leva a lugar nenhum e pior do que omiti-lo.
        return null;
    }

    const rawTerms = Array.isArray(record.matched_terms) ? record.matched_terms : [];

    return {
        documentId: asNumber(record.document_id),
        nodeId,
        title: asString(record.title),
        summary: asString(record.summary),
        startPage: asNullableNumber(record.start_page),
        score: asNumber(record.score),
        matchedTerms: rawTerms.filter((term): term is string => typeof term === "string"),
        kind: record.kind === "mesma-norma" || record.kind === "mesma-disciplina" ? record.kind : "termo-raro",
    };
}

// Lista vazia nao e erro: nenhuma conexao passou do limiar.
export function normalizeNodeConnections(value: unknown): NodeConnectionsResult {
    const record = asRecord(value);
    if (!record) {
        return { nodeId: "", minScore: 0, connections: [] };
    }

    const rawConnections = Array.isArray(record.connections) ? record.connections : [];

    return {
        nodeId: asString(record.node_id),
        minScore: asNumber(record.min_score),
        // Ja vem por score decrescente da API; nao reordenar.
        connections: rawConnections
            .map(normalizeConnection)
            .filter((connection): connection is NodeConnection => connection !== null),
    };
}
