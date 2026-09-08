"use client";

import { useEffect, useRef, useState } from "react";
import type { PDFDocumentProxy, RenderTask } from "pdfjs-dist";

import { abrirDocumento, fecharDocumento, medirPaginas, type TamanhoDePagina } from "@/components/workspace/pdf";

/**
 * Quadro de um PDF no canvas: cabecalho, e as paginas empilhadas com o tamanho
 * real (pontos PDF = pixels do mundo em zoom 1). Cada pagina e um placeholder;
 * so as que o pai marca como "vivas" ganham <canvas>. E o pai quem conhece o
 * viewport e o orcamento global -- este componente nao decide nada sobre isso.
 *
 * Abre o documento quando `visivel` fica true pela primeira vez, mede as
 * paginas (barato) e avisa o pai com onMedidas. Fora do viewport por mais de
 * 30 s, fecha o documento e libera memoria; reabre ao voltar.
 */

export const ALTURA_DO_CABECALHO = 40;
export const ESPACO_ENTRE_PAGINAS = 16;
export const LARGURA_ESTIMADA = 612;
export const ALTURA_ESTIMADA = 792;
const SEGUNDOS_ATE_FECHAR = 30;

export type PdfFrameProps = {
    documentId: number;
    titulo: string;
    subtitulo: string;
    visivel: boolean;
    /** Paginas (1-based) que devem ter canvas vivo agora. */
    paginasVivas: ReadonlySet<number>;
    /** Escala de rasterizacao estavel (ja com DPR). Muda 150 ms apos o zoom parar. */
    escalaDeRenderizacao: number;
    /** Abaixo do zoom minimo: mostrar so a miniatura da primeira pagina. */
    modoMiniatura: boolean;
    focado: boolean;
    onMedidas: (documentId: number, paginas: TamanhoDePagina[]) => void;
    onFocar: (documentId: number) => void;
    onPointerDownCabecalho: (event: React.PointerEvent<HTMLElement>) => void;
    /** Camada desenhada sobre cada pagina (grifos, marcadores). */
    renderizarSobreposicao?: (numeroDaPagina: number, tamanho: TamanhoDePagina) => React.ReactNode;
    children?: React.ReactNode;
};

export function larguraDoQuadro(paginas: TamanhoDePagina[] | undefined): number {
    return paginas && paginas.length > 0 ? Math.max(...paginas.map((p) => p.largura)) : LARGURA_ESTIMADA;
}

export function alturaDoQuadro(paginas: TamanhoDePagina[] | undefined): number {
    if (!paginas || paginas.length === 0) {
        return ALTURA_DO_CABECALHO + ALTURA_ESTIMADA;
    }
    return (
        ALTURA_DO_CABECALHO +
        paginas.reduce((soma, p) => soma + p.altura, 0) +
        ESPACO_ENTRE_PAGINAS * (paginas.length - 1)
    );
}

/** Deslocamento vertical da pagina N (1-based) dentro do quadro. */
export function topoDaPagina(paginas: TamanhoDePagina[], numero: number): number {
    let topo = ALTURA_DO_CABECALHO;
    for (let i = 0; i < numero - 1 && i < paginas.length; i += 1) {
        topo += paginas[i].altura + ESPACO_ENTRE_PAGINAS;
    }
    return topo;
}

export function PdfFrame({
    documentId,
    titulo,
    subtitulo,
    visivel,
    paginasVivas,
    escalaDeRenderizacao,
    modoMiniatura,
    focado,
    onMedidas,
    onFocar,
    onPointerDownCabecalho,
    renderizarSobreposicao,
    children,
}: PdfFrameProps) {
    const [documento, setDocumento] = useState<PDFDocumentProxy | null>(null);
    const [paginas, setPaginas] = useState<TamanhoDePagina[] | null>(null);
    const [erro, setErro] = useState<string | null>(null);

    // Abre quando visivel e ainda nao aberto; fecha depois de um tempo fora.
    // Sem flag "ja abriu" em ref: o StrictMode do React roda o efeito duas
    // vezes em dev, e a flag fazia a segunda passada desistir com o documento
    // ainda por carregar. Reexecutar e barato porque abrirDocumento tem cache.
    useEffect(() => {
        if (!visivel) {
            if (!documento) {
                return;
            }
            const timer = window.setTimeout(() => {
                fecharDocumento(documentId);
                setDocumento(null);
            }, SEGUNDOS_ATE_FECHAR * 1000);
            return () => window.clearTimeout(timer);
        }
        if (documento) {
            return;
        }
        let cancelado = false;
        abrirDocumento(documentId)
            .then((doc) => {
                if (!cancelado) {
                    setDocumento(doc);
                    setErro(null);
                }
            })
            .catch((falha: unknown) => {
                if (!cancelado) {
                    setErro(falha instanceof Error ? falha.message : "Falha ao abrir o PDF.");
                }
            });
        return () => {
            cancelado = true;
        };
    }, [visivel, documentId, documento]);

    // Medir e um efeito separado de proposito: se estivesse no mesmo efeito,
    // o setDocumento acima o reexecutaria e o cleanup cancelaria a medicao em
    // voo -- foi exatamente o que aconteceu na primeira versao.
    useEffect(() => {
        if (!documento || paginas) {
            return;
        }
        let cancelado = false;
        medirPaginas(documento)
            .then((medidas) => {
                if (!cancelado) {
                    setPaginas(medidas);
                    onMedidas(documentId, medidas);
                }
            })
            .catch(() => undefined);
        return () => {
            cancelado = true;
        };
    }, [documento, paginas, documentId, onMedidas]);

    const largura = larguraDoQuadro(paginas ?? undefined);

    return (
        <div
            className={`rounded-lg border bg-card shadow-sm ${focado ? "border-primary ring-2 ring-primary/30" : "border-border"}`}
            style={{ width: largura }}
            data-testid={`pdf-frame-${documentId}`}
            data-paginas={paginas?.length ?? 0}
        >
            <header
                className="flex cursor-grab items-baseline justify-between gap-3 px-3 text-left"
                style={{ height: ALTURA_DO_CABECALHO }}
                onPointerDown={onPointerDownCabecalho}
                onClick={() => onFocar(documentId)}
            >
                <span className="truncate text-sm font-medium">{titulo}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{subtitulo}</span>
            </header>

            {erro ? (
                <p className="p-3 text-xs text-destructive" role="alert">
                    {erro}
                </p>
            ) : null}

            {children}

            {paginas === null ? (
                <div
                    className="mx-auto animate-pulse bg-muted/60"
                    style={{ width: LARGURA_ESTIMADA, height: ALTURA_ESTIMADA }}
                    aria-hidden
                />
            ) : modoMiniatura ? (
                <div className="relative" style={{ width: largura, height: alturaDoQuadro(paginas) - ALTURA_DO_CABECALHO }}>
                    {documento ? (
                        <Miniatura documento={documento} tamanho={paginas[0]} />
                    ) : null}
                </div>
            ) : (
                paginas.map((tamanho, indice) => {
                    const numero = indice + 1;
                    return (
                        <div
                            key={numero}
                            className="relative mx-auto bg-white shadow-inner"
                            style={{
                                width: tamanho.largura,
                                height: tamanho.altura,
                                marginTop: indice === 0 ? 0 : ESPACO_ENTRE_PAGINAS,
                            }}
                            data-testid={`pdf-pagina-${documentId}-${numero}`}
                        >
                            {documento && paginasVivas.has(numero) ? (
                                <PdfPagina
                                    documento={documento}
                                    numero={numero}
                                    tamanho={tamanho}
                                    escala={escalaDeRenderizacao}
                                />
                            ) : null}
                            {renderizarSobreposicao?.(numero, tamanho)}
                        </div>
                    );
                })
            )}
        </div>
    );
}

type PdfPaginaProps = {
    documento: PDFDocumentProxy;
    numero: number;
    tamanho: TamanhoDePagina;
    escala: number;
};

/**
 * Rasteriza uma pagina no <canvas>. O canvas tem o tamanho fisico da escala
 * pedida, mas e exibido no tamanho do mundo (pontos): o zoom do canvas e feito
 * por CSS ate a proxima re-renderizacao, que so acontece quando `escala` muda.
 */
function PdfPagina({ documento, numero, tamanho, escala }: PdfPaginaProps) {
    const canvasRef = useRef<HTMLCanvasElement | null>(null);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) {
            return;
        }
        let tarefa: RenderTask | null = null;
        let cancelado = false;

        documento
            .getPage(numero)
            .then((pagina) => {
                if (cancelado) {
                    return;
                }
                const viewport = pagina.getViewport({ scale: escala });
                canvas.width = Math.ceil(viewport.width);
                canvas.height = Math.ceil(viewport.height);
                const contexto = canvas.getContext("2d");
                if (!contexto) {
                    return;
                }
                tarefa = pagina.render({ canvasContext: contexto, viewport, canvas });
                return tarefa.promise;
            })
            .catch((falha: unknown) => {
                // Cancelamento ao trocar de escala no meio e esperado; o resto nao.
                if (!(falha instanceof Error && falha.name === "RenderingCancelledException")) {
                    console.error("pdf.js: falha ao renderizar a pagina", numero, falha);
                }
            });

        return () => {
            cancelado = true;
            tarefa?.cancel();
        };
    }, [documento, numero, escala]);

    return (
        <canvas
            ref={canvasRef}
            className="absolute left-0 top-0"
            style={{ width: tamanho.largura, height: tamanho.altura }}
            data-pagina={numero}
            aria-label={`Página ${numero}`}
        />
    );
}

const miniaturas = new Map<string, string>();

function Miniatura({ documento, tamanho }: { documento: PDFDocumentProxy; tamanho: TamanhoDePagina }) {
    const chave = `${documento.fingerprints?.[0] ?? "doc"}`;
    const [url, setUrl] = useState<string | null>(() => miniaturas.get(chave) ?? null);

    useEffect(() => {
        if (url) {
            return;
        }
        let cancelado = false;
        documento
            .getPage(1)
            .then(async (pagina) => {
                const viewport = pagina.getViewport({ scale: 0.25 });
                const canvas = document.createElement("canvas");
                canvas.width = Math.ceil(viewport.width);
                canvas.height = Math.ceil(viewport.height);
                const contexto = canvas.getContext("2d");
                if (!contexto) {
                    return;
                }
                await pagina.render({ canvasContext: contexto, viewport, canvas }).promise;
                const dataUrl = canvas.toDataURL("image/png");
                miniaturas.set(chave, dataUrl);
                if (!cancelado) {
                    setUrl(dataUrl);
                }
            })
            .catch(() => undefined);
        return () => {
            cancelado = true;
        };
    }, [documento, chave, url]);

    if (!url) {
        return <div className="absolute left-0 top-0 bg-muted/60" style={{ width: tamanho.largura, height: tamanho.altura }} aria-hidden />;
    }
    // eslint-disable-next-line @next/next/no-img-element -- data URL local, sem otimizacao
    return <img src={url} alt="" className="absolute left-0 top-0" style={{ width: tamanho.largura, height: tamanho.altura }} draggable={false} />;
}
