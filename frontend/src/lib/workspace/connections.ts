/**
 * Logica pura das conexoes desenhadas no canvas: de uma lista de conexoes
 * derivadas (API) para linhas em coordenadas do mundo, mais o conjunto de
 * paragrafos da secao destino que ganham grifo secundario. Sem React.
 *
 * O score e a contagem de termos significativos da origem encontrados no
 * destino (inteiro, minimo 2 por padrao). Nao e probabilidade, e a espessura
 * da linha e o rotulo dizem isso: "3 termos", nunca "0,75".
 */

import type { OutlineNode, ParagraphNode } from "@/lib/api/contracts/document-context";
import type { NodeConnection } from "@/lib/api/contracts/node-connections";
import { ancorasDeLinha, type Ponto, type Rect } from "@/lib/workspace/geometry";

export type Linha = {
    id: string;
    documentId: number;
    nodeId: string;
    titulo: string;
    score: number;
    termos: string[];
    de: Ponto;
    para: Ponto;
    /** true quando a linha termina na borda do quadro, sem ancora precisa. */
    aproximada: boolean;
};

export function idDaLinha(conexao: Pick<NodeConnection, "documentId" | "nodeId">): string {
    return `${conexao.documentId}:${conexao.nodeId}`;
}

export function espessuraPorScore(score: number): number {
    return 1.5 + Math.min(Math.max(score, 0), 6) * 0.5;
}

export function rotuloDoScore(score: number): string {
    return score === 1 ? "1 termo" : `${score} termos`;
}

export function slugDoNodeId(nodeId: string, sumario: OutlineNode[]): string | null {
    return sumario.find((no) => no.nodeId === nodeId)?.slug ?? null;
}

export function sectionKeyDoParagrafo(no: Pick<ParagraphNode, "metadata">): string | null {
    const chave = no.metadata?.section_key;
    return typeof chave === "string" && chave !== "" ? chave : null;
}

/** Paragrafos das secoes destino: mesmo documento e section_key igual ao slug do no. */
export function paragrafosSecundarios(
    conexoes: NodeConnection[],
    sumario: OutlineNode[],
    paragrafos: ParagraphNode[]
): Set<ParagraphNode["id"]> {
    const alvos = new Set<string>();
    for (const conexao of conexoes) {
        const slug = slugDoNodeId(conexao.nodeId, sumario);
        if (slug) {
            alvos.add(`${conexao.documentId}:${slug}`);
        }
    }
    const saida = new Set<ParagraphNode["id"]>();
    if (alvos.size === 0) {
        return saida;
    }
    for (const no of paragrafos) {
        const chave = sectionKeyDoParagrafo(no);
        if (chave && alvos.has(`${no.documentId}:${chave}`)) {
            saida.add(no.id);
        }
    }
    return saida;
}

export type ResolvedorDeRects = {
    origem: Rect | null;
    /** Rect preciso do destino (marcador de secao) ou null se nao ha como saber. */
    destinoPreciso: (conexao: NodeConnection) => Rect | null;
    /** Rect do quadro do documento destino, usado como fallback. */
    quadroDoDocumento: (documentId: number) => Rect | null;
};

export function montarLinhas(conexoes: NodeConnection[], rects: ResolvedorDeRects): Linha[] {
    const origem = rects.origem;
    if (!origem) {
        return [];
    }
    const linhas: Linha[] = [];
    for (const conexao of conexoes) {
        const preciso = rects.destinoPreciso(conexao);
        const destino = preciso ?? rects.quadroDoDocumento(conexao.documentId);
        if (!destino) {
            continue;
        }
        const { de, para } = ancorasDeLinha(origem, destino);
        linhas.push({
            id: idDaLinha(conexao),
            documentId: conexao.documentId,
            nodeId: conexao.nodeId,
            titulo: conexao.title,
            score: conexao.score,
            termos: conexao.matchedTerms,
            de,
            para,
            aproximada: preciso === null,
        });
    }
    return linhas;
}

/** Curva de Bezier com controles horizontais: legivel mesmo entre quadros lado a lado. */
export function caminhoDaLinha(de: Ponto, para: Ponto): string {
    const dx = Math.max(Math.abs(para.x - de.x) / 2, 40);
    const c1 = { x: de.x + (para.x >= de.x ? dx : -dx), y: de.y };
    const c2 = { x: para.x - (para.x >= de.x ? dx : -dx), y: para.y };
    return `M ${de.x} ${de.y} C ${c1.x} ${c1.y}, ${c2.x} ${c2.y}, ${para.x} ${para.y}`;
}
