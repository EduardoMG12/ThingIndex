/**
 * Estado de visualizacao do workspace, por usuario e colecao, como a API
 * guarda (objeto opaco para o backend; o formato e nosso). Normalizador
 * tolerante: chave desconhecida, numero nao finito ou modo invalido somem em
 * vez de quebrar a tela.
 */

export type ViewportSalvo = { x: number; y: number; escala: number };
export type ApresentacaoSalva = { modo: "folhas" | "pagina"; paginaAtual: number };

export type WorkspaceViewState = {
    viewport?: ViewportSalvo;
    quadros?: Record<string, { x: number; y: number }>;
    apresentacao?: Record<string, ApresentacaoSalva>;
    marcadoresVisiveis?: boolean;
    documentoFocado?: number;
};

function asRecord(value: unknown): Record<string, unknown> | null {
    return typeof value === "object" && value !== null && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
}

function finito(value: unknown): value is number {
    return typeof value === "number" && Number.isFinite(value);
}

export function normalizeViewState(value: unknown): WorkspaceViewState {
    const raiz = asRecord(value);
    const record = asRecord(raiz?.state) ?? raiz;
    const saida: WorkspaceViewState = {};
    if (!record) {
        return saida;
    }
    const vp = asRecord(record.viewport);
    if (vp && finito(vp.x) && finito(vp.y) && finito(vp.escala) && vp.escala > 0) {
        saida.viewport = { x: vp.x, y: vp.y, escala: vp.escala };
    }
    const quadros = asRecord(record.quadros);
    if (quadros) {
        const limpos: Record<string, { x: number; y: number }> = {};
        for (const [id, pos] of Object.entries(quadros)) {
            const p = asRecord(pos);
            if (p && finito(p.x) && finito(p.y)) {
                limpos[id] = { x: p.x, y: p.y };
            }
        }
        if (Object.keys(limpos).length > 0) {
            saida.quadros = limpos;
        }
    }
    const apresentacao = asRecord(record.apresentacao);
    if (apresentacao) {
        const limpas: Record<string, ApresentacaoSalva> = {};
        for (const [id, a] of Object.entries(apresentacao)) {
            const r = asRecord(a);
            if (r && (r.modo === "folhas" || r.modo === "pagina") && finito(r.paginaAtual) && r.paginaAtual >= 1) {
                limpas[id] = { modo: r.modo, paginaAtual: Math.trunc(r.paginaAtual) };
            }
        }
        if (Object.keys(limpas).length > 0) {
            saida.apresentacao = limpas;
        }
    }
    if (typeof record.marcadoresVisiveis === "boolean") {
        saida.marcadoresVisiveis = record.marcadoresVisiveis;
    }
    if (finito(record.documentoFocado) && record.documentoFocado > 0) {
        saida.documentoFocado = record.documentoFocado;
    }
    return saida;
}
