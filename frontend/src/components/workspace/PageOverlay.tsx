"use client";

import { useState } from "react";

import { readBox, type OutlineNode, type ParagraphNode } from "@/lib/api/contracts/document-context";
import { useI18n } from "@/lib/i18n/provider";
import { retryDocumentIndexing } from "@/lib/api/documents";
import { boxParaPixels } from "@/lib/workspace/geometry";
import type { TamanhoDePagina } from "@/components/workspace/pdf";

/**
 * Camada desenhada sobre uma pagina do PDF: um grifo por paragrafo com
 * bounding box, um marcador na margem por secao que comeca nesta pagina e um
 * cabecalho discreto com numero e origem do texto. E a prova visual da tese --
 * cada trecho recuperado sabe de que pagina e de que posicao veio.
 *
 * As coordenadas vem em pontos PDF e a pagina e desenhada em pontos (1 ponto
 * = 1 px do mundo em zoom 1), entao a conversao e a identidade; boxParaPixels
 * fica aqui mesmo assim para o dia em que a pagina for renderizada em outro
 * tamanho.
 */

export type ParagrafoNaPagina = {
    no: ParagraphNode;
    secao: OutlineNode | null;
};

export type PageOverlayProps = {
    numero: number;
    tamanho: TamanhoDePagina;
    paragrafos: ParagrafoNaPagina[];
    /** Secoes cujo start_page e esta pagina. */
    secoes: OutlineNode[];
    /** Primeiro paragrafo com box de cada secao (por slug), para ancorar o marcador. */
    ancoraDaSecao: (secao: OutlineNode) => ParagraphNode | null;
    origem: string | null;
    selecionadoId: ParagraphNode["id"] | null;
    /** Paragrafos da secao destino de uma conexao ativa, em cor secundaria. */
    secundarios: ReadonlySet<ParagraphNode["id"]>;
    onSelecionar: (paragrafo: ParagraphNode, secao: OutlineNode | null) => void;
};

/** Marcadores so ate o nivel 2: um edital tem 300+ nos de sumario, e cada "1.1" virando etiqueta e ruido, nao mapa. */
export const NIVEL_MAXIMO_DO_MARCADOR = 2;
const ALTURA_DO_MARCADOR = 14;

/**
 * Posicoes verticais dos marcadores sem sobreposicao: ordena pela ancora e
 * empurra para baixo quem colidiria com o anterior.
 */
export function distribuirMarcadores<T>(itens: { item: T; top: number }[]): { item: T; top: number }[] {
    const ordenados = [...itens].sort((a, b) => a.top - b.top);
    let ultimo = -Infinity;
    return ordenados.map(({ item, top }) => {
        const ajustado = Math.max(top, ultimo + ALTURA_DO_MARCADOR);
        ultimo = ajustado;
        return { item, top: ajustado };
    });
}

export function sectionKeyDe(no: Pick<ParagraphNode, "metadata">): string | null {
    const chave = no.metadata?.section_key;
    return typeof chave === "string" && chave.trim() !== "" ? chave : null;
}

export function origemDe(no: Pick<ParagraphNode, "metadata">): string | null {
    const origem = no.metadata?.source;
    return typeof origem === "string" && origem.trim() !== "" ? origem : null;
}

/** Origem mais frequente entre os paragrafos de uma pagina. */
export function origemMajoritaria(nos: Pick<ParagraphNode, "metadata">[]): string | null {
    const contagem = new Map<string, number>();
    for (const no of nos) {
        const origem = origemDe(no);
        if (origem) {
            contagem.set(origem, (contagem.get(origem) ?? 0) + 1);
        }
    }
    let melhor: string | null = null;
    let maior = 0;
    for (const [origem, n] of contagem) {
        if (n > maior) {
            melhor = origem;
            maior = n;
        }
    }
    return melhor;
}

export function PageOverlay({
    numero,
    tamanho,
    paragrafos,
    secoes,
    ancoraDaSecao,
    origem,
    selecionadoId,
    secundarios,
    onSelecionar,
}: PageOverlayProps) {
    const { t } = useI18n();
    return (
        <div className="pointer-events-none absolute inset-0" data-testid={`overlay-pagina-${numero}`}>
            <span
                className="absolute right-1 top-1 rounded bg-background/80 px-1.5 py-0.5 text-[10px] leading-none text-muted-foreground"
                data-testid="cabecalho-pagina"
            >
                {t("workspace.page.header", { number: numero })}
                {origem ? ` · ${origem}` : ""}
            </span>

            {distribuirMarcadores(
                secoes
                    .filter((secao) => secao.level <= NIVEL_MAXIMO_DO_MARCADOR)
                    .map((secao) => {
                        const ancora = ancoraDaSecao(secao);
                        const box = ancora ? readBox(ancora) : null;
                        // Sem ancora nao ha marcador: posicao inventada engana.
                        return box ? { item: secao, top: boxParaPixels(box, tamanho.largura, tamanho.altura).top } : null;
                    })
                    .filter((m): m is { item: OutlineNode; top: number } => m !== null)
            ).map(({ item: secao, top }) => (
                // Dentro da margem esquerda da pagina (72 pt num A4/carta), para
                // nunca invadir o quadro vizinho nem o texto.
                <span
                    key={secao.nodeId}
                    className="pointer-events-auto absolute left-0.5 w-[64px] truncate rounded-sm border border-primary/40 bg-primary/10 px-1 text-[8px] leading-[12px] text-primary"
                    style={{ top }}
                    title={secao.title}
                    data-testid="marcador-secao"
                >
                    {secao.title}
                </span>
            ))}

            {paragrafos.map(({ no, secao }) => {
                const box = readBox(no);
                if (!box) {
                    return null;
                }
                const px = boxParaPixels(box, tamanho.largura, tamanho.altura);
                const selecionado = no.id === selecionadoId;
                const secundario = secundarios.has(no.id);
                return (
                    <button
                        key={String(no.id)}
                        type="button"
                        onClick={(event) => {
                            event.stopPropagation();
                            onSelecionar(no, secao);
                        }}
                        onPointerDown={(event) => event.stopPropagation()}
                        className={`pointer-events-auto absolute rounded-sm border transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary ${
                            selecionado
                                ? "border-primary bg-primary/35"
                                : secundario
                                  ? "border-amber-500/70 bg-amber-400/30"
                                  : "border-transparent bg-primary/10 hover:bg-primary/25"
                        }`}
                        style={{ left: px.left, top: px.top, width: px.width, height: px.height }}
                        aria-label={
                            secao
                                ? t("workspace.page.paragraphInSection", { number: no.sequenceOrder + 1, page: numero, section: secao.title })
                                : t("workspace.page.paragraph", { number: no.sequenceOrder + 1, page: numero })
                        }
                        aria-pressed={selecionado}
                        data-testid="grifo"
                        data-paragrafo={String(no.id)}
                    />
                );
            })}
        </div>
    );
}

type AvisoSemPosicaoProps = {
    documentId: number;
    totalParagrafos: number;
};

/**
 * PDF indexado antes de a bounding box existir (ou por um caminho que nao a
 * produz). Nunca reindexa sozinho: com LLM configurado, reindexar custa
 * dinheiro, e a decisao e de quem esta olhando.
 */
export function AvisoSemPosicao({ documentId, totalParagrafos }: AvisoSemPosicaoProps) {
    const { t } = useI18n();
    const [estado, setEstado] = useState<"parado" | "enviando" | "na-fila" | "erro">("parado");

    async function reindexar() {
        setEstado("enviando");
        try {
            await retryDocumentIndexing(documentId);
            setEstado("na-fila");
        } catch {
            setEstado("erro");
        }
    }

    return (
        <div
            className="flex items-center justify-between gap-3 border-b border-amber-500/40 bg-amber-100/60 px-3 py-2 text-xs text-amber-900 dark:bg-amber-900/30 dark:text-amber-100"
            role="status"
            data-testid="aviso-sem-posicao"
        >
            <span>{t("workspace.warning.noBoxes", { count: totalParagrafos })}</span>
            {estado === "na-fila" ? (
                <span className="shrink-0 font-medium">{t("workspace.warning.queued")}</span>
            ) : (
                <button
                    type="button"
                    onClick={reindexar}
                    disabled={estado === "enviando"}
                    onPointerDown={(event) => event.stopPropagation()}
                    className="shrink-0 rounded border border-amber-700/50 px-2 py-1 font-medium hover:bg-amber-200/60 disabled:opacity-60"
                >
                    {estado === "enviando"
                        ? t("workspace.warning.sending")
                        : estado === "erro"
                          ? t("workspace.warning.retry")
                          : t("workspace.warning.reindex")}
                </button>
            )}
        </div>
    );
}
