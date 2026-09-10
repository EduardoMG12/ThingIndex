/**
 * Conexoes derivadas entre nos de documentos distintos.
 *
 * "Derivada" e a palavra que carrega o peso: a API recalcula estas relacoes a
 * cada chamada, a partir do texto do proprio no, e nao existe tabela de arestas
 * no banco. O principio II da constituicao do projeto admite grafos apenas como
 * trilha comparativa, jamais como camada primaria de recuperacao.
 *
 * Consequencia pratica para quem consome: nao ha o que invalidar em cache de
 * escrita, porque nao ha escrita. Duas chamadas com os mesmos parametros
 * devolvem o mesmo resultado.
 */

export type NodeConnection = {
    documentId: number;
    nodeId: string;
    title: string;
    summary: string;
    startPage: number | null;
    score: number;
    /**
     * Termos da origem encontrados no destino. Sao a justificativa da aresta --
     * sem eles a conexao seria tao opaca quanto uma vizinhanca vetorial, que e
     * exatamente o que este projeto evita.
     */
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

/**
 * Converte a resposta da API no formato usado pela interface.
 *
 * Lista vazia NAO e erro: significa que nenhuma conexao superou o limiar, que e
 * a resposta correta quando nao existe relacao real. Este trabalho mede taxa de
 * falso positivo de 1,000 nos seis bracos avaliados justamente porque nenhum
 * sabe se calar; esconder o vazio na interface desfaria o unico ponto em que o
 * sistema acerta esse comportamento.
 */
export function normalizeNodeConnections(value: unknown): NodeConnectionsResult {
    const record = asRecord(value);
    if (!record) {
        return { nodeId: "", minScore: 0, connections: [] };
    }

    const rawConnections = Array.isArray(record.connections) ? record.connections : [];

    return {
        nodeId: asString(record.node_id),
        minScore: asNumber(record.min_score),
        // A ordem vem da API, ja por score decrescente. Reordenar aqui poderia
        // divergir do backend e quebrar o determinismo que o trabalho afirma.
        connections: rawConnections
            .map(normalizeConnection)
            .filter((connection): connection is NodeConnection => connection !== null),
    };
}
