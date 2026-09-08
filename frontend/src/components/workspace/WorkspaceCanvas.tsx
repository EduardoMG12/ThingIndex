"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Maximize2, Minus, Plus } from "lucide-react";

import { fetchNodeConnections } from "@/lib/api/collections";
import type { CollectionContext, OutlineNode } from "@/lib/api/contracts/document-context";
import type { NodeConnection } from "@/lib/api/contracts/node-connections";
import { rectVisivel, type Rect } from "@/lib/workspace/geometry";
import { escalaDeRenderizacao, escolherPaginasVivas, ZOOM_MINIMO_PARA_PAGINAS } from "@/lib/workspace/render-budget";
import { useViewport } from "@/components/workspace/useViewport";
import {
    alturaDoQuadro,
    larguraDoQuadro,
    PdfFrame,
    topoDaPagina,
} from "@/components/workspace/PdfFrame";
import type { TamanhoDePagina } from "@/components/workspace/pdf";

/**
 * Canvas navegavel dos documentos de uma colecao: pan, zoom e quadros
 * arrastaveis num "mundo" em CSS transform, mais a sidebar de estrutura.
 *
 * # Por que nao React Flow
 *
 * A spec 001 previa React Flow. Ao implementar ficou claro que a biblioteca
 * resolveria um problema que este painel nao tem: nao ha grafo persistido para
 * fazer layout automatico. Pan, zoom e arrasto cabem em ~150 linhas de CSS
 * transform (useViewport + lib/workspace/geometry), testaveis sem DOM. Se
 * minimapa ou layout automatico entrarem em escopo, revisitar.
 *
 * # O que este componente NAO faz
 *
 * Nao chama LLM. Titulos e resumos exibidos na sidebar ja estao no indice,
 * gravados na indexacao deterministica. A tela e leitura de dado existente, e
 * essa e a razao de ela ser instantanea.
 */

type WorkspaceCanvasProps = {
    context: CollectionContext;
};

type Posicao = { x: number; y: number };

const LARGURA_CARTAO = 260;
const ALTURA_CARTAO = 96;
/** Quadros de PDF tem ~612 px de largura; uma coluna por documento. */
const PASSO_ENTRE_QUADROS = 760;
/** Folga para comecar a abrir o PDF um pouco antes de ele entrar na tela. */
const FOLGA_DE_VISIBILIDADE = 240;

function ehPdf(documento: { mimeType?: string; filePath?: string }): boolean {
    return documento.mimeType === "application/pdf" || /\.pdf$/i.test(documento.filePath ?? "");
}

function chaveDeArmazenamento(collectionId: number | string): string {
    return `thingindex.workspace.positions.${collectionId}`;
}

/**
 * Posicao inicial em grade. Deterministica de proposito: a mesma colecao abre
 * sempre com o mesmo arranjo, o que torna a tela reproduzivel numa demonstracao.
 */
function posicaoInicial(indice: number): Posicao {
    return { x: indice * PASSO_ENTRE_QUADROS, y: 0 };
}

function lerPosicoesSalvas(collectionId: number | string): Record<string, Posicao> {
    if (typeof window === "undefined") {
        return {};
    }
    try {
        const bruto = window.localStorage.getItem(chaveDeArmazenamento(collectionId));
        if (!bruto) {
            return {};
        }
        const analisado = JSON.parse(bruto) as unknown;
        return typeof analisado === "object" && analisado !== null
            ? (analisado as Record<string, Posicao>)
            : {};
    } catch {
        // localStorage corrompido nao pode impedir a tela de abrir; o custo de
        // ignorar e reposicionar os cartoes, e nao perder o acesso.
        return {};
    }
}

export function WorkspaceCanvas({ context }: WorkspaceCanvasProps) {
    const collectionIdBruto = context.collection?.id ?? 0;
    // A API aceita id numerico; o contrato do frontend admite string porque
    // algumas respostas antigas devolviam assim.
    const collectionId =
        typeof collectionIdBruto === "number" ? collectionIdBruto : Number(collectionIdBruto) || 0;
    const documentos = useMemo(() => context.documents ?? [], [context.documents]);
    const nosDoSumario = useMemo(() => context.outlineNodes ?? [], [context.outlineNodes]);

    // Lazy init + `key={collectionId}` na pagina: trocar de colecao remonta o
    // canvas, entao nao precisa de efeito para recarregar as posicoes.
    const [posicoes, setPosicoes] = useState<Record<string, Posicao>>(() => lerPosicoesSalvas(collectionId));
    const [documentoFocado, setDocumentoFocado] = useState<number | null>(null);
    const [noSelecionado, setNoSelecionado] = useState<string | null>(null);
    const [conexoes, setConexoes] = useState<NodeConnection[] | null>(null);
    const [carregandoConexoes, setCarregandoConexoes] = useState(false);
    const [erroConexoes, setErroConexoes] = useState<string | null>(null);

    const {
        containerRef,
        viewport,
        tamanho,
        arrastandoFundo,
        handlers,
        aproximar,
        afastar,
        enquadrar,
        clienteParaMundo,
    } = useViewport();

    // Medidas das paginas de cada PDF (pontos), reportadas pelo quadro ao abrir.
    const [medidas, setMedidas] = useState<Record<number, TamanhoDePagina[]>>({});
    const aoMedir = useCallback((documentId: number, paginas: TamanhoDePagina[]) => {
        setMedidas((atual) => ({ ...atual, [documentId]: paginas }));
    }, []);

    // Escala de rasterizacao so muda 150 ms depois de o zoom parar: no meio
    // do gesto o canvas escala por CSS, sem re-renderizar a cada tick.
    const [escalaEstavel, setEscalaEstavel] = useState(viewport.escala);
    useEffect(() => {
        const timer = window.setTimeout(() => setEscalaEstavel(viewport.escala), 150);
        return () => window.clearTimeout(timer);
    }, [viewport.escala]);
    const dpr = typeof window === "undefined" ? 1 : window.devicePixelRatio;
    const escalaRender = escalaDeRenderizacao(escalaEstavel, dpr);
    const modoMiniatura = viewport.escala < ZOOM_MINIMO_PARA_PAGINAS;

    // Arrasto de cartao em coordenadas do MUNDO: o offset e medido depois de
    // desfazer o transform, senao o cartao "foge" do cursor em qualquer zoom.
    const arrastando = useRef<{ documentId: number; offset: Posicao } | null>(null);
    const clienteParaMundoRef = useRef(clienteParaMundo);
    useEffect(() => {
        clienteParaMundoRef.current = clienteParaMundo;
    }, [clienteParaMundo]);

    const posicaoDe = useCallback(
        (documentId: number, indice: number): Posicao =>
            posicoes[String(documentId)] ?? posicaoInicial(indice),
        [posicoes]
    );

    const quadros = useMemo<Rect[]>(
        () =>
            documentos.map((documento, indice) => {
                const p = posicaoDe(documento.id, indice);
                if (!ehPdf(documento)) {
                    return { x: p.x, y: p.y, largura: LARGURA_CARTAO, altura: ALTURA_CARTAO };
                }
                const paginas = medidas[documento.id];
                return { x: p.x, y: p.y, largura: larguraDoQuadro(paginas), altura: alturaDoQuadro(paginas) };
            }),
        [documentos, posicaoDe, medidas]
    );

    // Orcamento global: quais paginas, somando todos os PDFs, ganham canvas.
    const paginasVivasPorDocumento = useMemo(() => {
        const candidatas: { id: string; rect: Rect }[] = [];
        documentos.forEach((documento, indice) => {
            const paginas = medidas[documento.id];
            if (!paginas) {
                return;
            }
            const quadro = quadros[indice];
            paginas.forEach((pagina, i) => {
                candidatas.push({
                    id: `${documento.id}:${i + 1}`,
                    rect: {
                        x: quadro.x + (quadro.largura - pagina.largura) / 2,
                        y: quadro.y + topoDaPagina(paginas, i + 1),
                        largura: pagina.largura,
                        altura: pagina.altura,
                    },
                });
            });
        });
        const vivas = escolherPaginasVivas(candidatas, viewport, tamanho);
        const porDocumento = new Map<number, Set<number>>();
        for (const id of vivas) {
            const [doc, pagina] = id.split(":");
            const conjunto = porDocumento.get(Number(doc)) ?? new Set<number>();
            conjunto.add(Number(pagina));
            porDocumento.set(Number(doc), conjunto);
        }
        return porDocumento;
    }, [documentos, medidas, quadros, viewport, tamanho]);

    const conjuntoVazio = useMemo(() => new Set<number>(), []);

    const quadroVisivel = useCallback(
        (rect: Rect) =>
            rectVisivel(
                {
                    x: rect.x - FOLGA_DE_VISIBILIDADE,
                    y: rect.y - FOLGA_DE_VISIBILIDADE,
                    largura: rect.largura + 2 * FOLGA_DE_VISIBILIDADE,
                    altura: rect.altura + 2 * FOLGA_DE_VISIBILIDADE,
                },
                viewport,
                tamanho
            ),
        [viewport, tamanho]
    );

    const aoIniciarArrasto = useCallback(
        (event: React.PointerEvent<HTMLElement>, documentId: number, indice: number) => {
            event.stopPropagation();
            const atual = posicaoDe(documentId, indice);
            const mundo = clienteParaMundoRef.current(event.clientX, event.clientY);
            arrastando.current = {
                documentId,
                offset: { x: mundo.x - atual.x, y: mundo.y - atual.y },
            };
        },
        [posicaoDe]
    );

    useEffect(() => {
        function aoMover(event: PointerEvent) {
            const estado = arrastando.current;
            if (!estado) {
                return;
            }
            const mundo = clienteParaMundoRef.current(event.clientX, event.clientY);
            setPosicoes((anterior) => ({
                ...anterior,
                [String(estado.documentId)]: {
                    x: mundo.x - estado.offset.x,
                    y: mundo.y - estado.offset.y,
                },
            }));
        }

        function aoSoltar() {
            if (!arrastando.current) {
                return;
            }
            arrastando.current = null;
            setPosicoes((atual) => {
                try {
                    window.localStorage.setItem(
                        chaveDeArmazenamento(collectionId),
                        JSON.stringify(atual)
                    );
                } catch {
                    // Quota estourada ou modo privativo: o arrasto continua
                    // valendo na sessao, so nao sobrevive ao recarregamento.
                }
                return atual;
            });
        }

        window.addEventListener("pointermove", aoMover);
        window.addEventListener("pointerup", aoSoltar);
        return () => {
            window.removeEventListener("pointermove", aoMover);
            window.removeEventListener("pointerup", aoSoltar);
        };
    }, [collectionId]);

    const limparSelecao = useCallback(() => {
        setDocumentoFocado(null);
        setNoSelecionado(null);
        setConexoes(null);
    }, []);

    useEffect(() => {
        function aoTeclar(event: KeyboardEvent) {
            if (event.key === "Escape") {
                limparSelecao();
            }
        }
        window.addEventListener("keydown", aoTeclar);
        return () => window.removeEventListener("keydown", aoTeclar);
    }, [limparSelecao]);

    const nosDoDocumentoFocado = useMemo<OutlineNode[]>(() => {
        if (documentoFocado === null) {
            return [];
        }
        return nosDoSumario.filter((no) => no.documentId === documentoFocado);
    }, [documentoFocado, nosDoSumario]);

    const pedirConexoes = useCallback(
        async (nodeId: string) => {
            setNoSelecionado(nodeId);
            setCarregandoConexoes(true);
            setErroConexoes(null);
            setConexoes(null);

            try {
                const resultado = await fetchNodeConnections(collectionId, nodeId);
                setConexoes(resultado.connections);
            } catch (erro) {
                setErroConexoes(
                    erro instanceof Error ? erro.message : "Falha ao buscar conexoes."
                );
            } finally {
                setCarregandoConexoes(false);
            }
        },
        [collectionId]
    );

    const documentoAtual = documentos.find((documento) => documento.id === documentoFocado);

    return (
        // Empilha em telas estreitas e divide em duas colunas a partir de lg.
        // `min-w-0` e obrigatorio: sem ele o item flex assume a largura do
        // conteudo e o canvas colapsa para dezenas de pixels em 780px (medido).
        <div className="flex h-full w-full flex-col gap-4 lg:flex-row">
            <div
                ref={containerRef}
                className={`relative h-[60vh] w-full min-w-0 flex-1 overflow-hidden rounded-lg border border-border bg-muted/20 touch-none lg:h-[calc(100vh-11rem)] ${
                    arrastandoFundo ? "cursor-grabbing" : "cursor-grab"
                }`}
                data-testid="workspace-canvas"
                {...handlers}
            >
                <div
                    className="absolute left-0 top-0 origin-top-left"
                    style={{
                        transform: `translate(${viewport.x}px, ${viewport.y}px) scale(${viewport.escala})`,
                    }}
                    data-testid="workspace-mundo"
                >
                    {documentos.map((documento, indice) => {
                        const posicao = posicaoDe(documento.id, indice);
                        const focado = documento.id === documentoFocado;
                        const trechos = nosDoSumario.filter((no) => no.documentId === documento.id).length;

                        if (ehPdf(documento)) {
                            return (
                                <div
                                    key={documento.id}
                                    className="absolute"
                                    style={{ transform: `translate(${posicao.x}px, ${posicao.y}px)` }}
                                >
                                    <PdfFrame
                                        documentId={documento.id}
                                        titulo={documento.title}
                                        subtitulo={`${trechos} trechos`}
                                        visivel={quadroVisivel(quadros[indice])}
                                        paginasVivas={paginasVivasPorDocumento.get(documento.id) ?? conjuntoVazio}
                                        escalaDeRenderizacao={escalaRender}
                                        modoMiniatura={modoMiniatura}
                                        focado={focado}
                                        onMedidas={aoMedir}
                                        onFocar={(id) => {
                                            setDocumentoFocado(id);
                                            setConexoes(null);
                                            setNoSelecionado(null);
                                        }}
                                        onPointerDownCabecalho={(event) => aoIniciarArrasto(event, documento.id, indice)}
                                    />
                                </div>
                            );
                        }

                        return (
                            <div
                                key={documento.id}
                                className="absolute"
                                style={{
                                    transform: `translate(${posicao.x}px, ${posicao.y}px)`,
                                    width: LARGURA_CARTAO,
                                }}
                            >
                                <button
                                    type="button"
                                    onPointerDown={(event) =>
                                        aoIniciarArrasto(event, documento.id, indice)
                                    }
                                    onClick={() => {
                                        setDocumentoFocado(documento.id);
                                        setConexoes(null);
                                        setNoSelecionado(null);
                                    }}
                                    className={`w-full cursor-grab rounded-lg border p-4 text-left transition ${
                                        focado
                                            ? "border-primary bg-card shadow-lg"
                                            : "border-border bg-card/80 hover:border-primary/50"
                                    }`}
                                >
                                    <span className="block text-sm font-medium">
                                        {documento.title}
                                    </span>
                                    <span className="mt-1 block text-xs text-muted-foreground">
                                        {trechos} trechos indexados
                                    </span>
                                </button>
                            </div>
                        );
                    })}
                </div>

                <div
                    className="absolute bottom-3 right-3 flex items-center gap-1 rounded-md border border-border bg-card/95 p-1 shadow-sm"
                    role="toolbar"
                    aria-label="Zoom do canvas"
                    onPointerDown={(event) => event.stopPropagation()}
                >
                    <button
                        type="button"
                        onClick={afastar}
                        aria-label="Afastar"
                        className="rounded p-1.5 hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                    >
                        <Minus className="h-4 w-4" aria-hidden />
                    </button>
                    <span className="min-w-[3.5rem] text-center text-xs tabular-nums text-muted-foreground" aria-live="polite">
                        {Math.round(viewport.escala * 100)}%
                    </span>
                    <button
                        type="button"
                        onClick={aproximar}
                        aria-label="Aproximar"
                        className="rounded p-1.5 hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                    >
                        <Plus className="h-4 w-4" aria-hidden />
                    </button>
                    <button
                        type="button"
                        onClick={() => enquadrar(quadros)}
                        aria-label="Ajustar à tela"
                        className="rounded p-1.5 hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                    >
                        <Maximize2 className="h-4 w-4" aria-hidden />
                    </button>
                </div>
            </div>

            <aside
                className="w-full shrink-0 overflow-y-auto rounded-lg border border-border bg-card p-4 lg:w-80"
                aria-label="Detalhes do documento"
            >
                {documentoFocado === null ? (
                    <p className="text-sm text-muted-foreground">
                        Selecione um documento no painel para ver sua estrutura.
                    </p>
                ) : (
                    <div className="space-y-4">
                        <h2 className="text-sm font-semibold">{documentoAtual?.title}</h2>

                        {nosDoDocumentoFocado.length === 0 ? (
                            <p className="text-sm text-muted-foreground">
                                Documento sem estrutura indexada.
                            </p>
                        ) : (
                            <ul className="space-y-3">
                                {nosDoDocumentoFocado.map((no) => (
                                    <li
                                        key={no.nodeId}
                                        className={`rounded-md border p-3 ${
                                            no.nodeId === noSelecionado
                                                ? "border-primary"
                                                : "border-border"
                                        }`}
                                    >
                                        <p className="text-sm font-medium">{no.title}</p>
                                        {no.summary ? (
                                            <p className="mt-1 text-xs text-muted-foreground">
                                                {no.summary}
                                            </p>
                                        ) : null}
                                        <button
                                            type="button"
                                            onClick={() => pedirConexoes(no.nodeId)}
                                            className="mt-2 text-xs font-medium text-primary underline-offset-2 hover:underline"
                                        >
                                            Ver conexões
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        )}

                        {carregandoConexoes ? (
                            <p className="text-xs text-muted-foreground">Buscando conexões…</p>
                        ) : null}

                        {erroConexoes ? (
                            <p className="text-xs text-destructive">{erroConexoes}</p>
                        ) : null}

                        {conexoes !== null && !carregandoConexoes ? (
                            conexoes.length === 0 ? (
                                // Ausencia DECLARADA, nunca area em branco. Uma
                                // regiao vazia seria lida como falha de
                                // carregamento, e o sistema perderia credito no
                                // unico ponto em que sabe dizer "nao encontrei".
                                <p className="text-xs text-muted-foreground">
                                    Nenhuma conexão acima do limiar para este trecho.
                                </p>
                            ) : (
                                <ul className="space-y-2 border-t border-border pt-3">
                                    {conexoes.map((conexao) => (
                                        <li
                                            key={`${conexao.documentId}-${conexao.nodeId}`}
                                            className="rounded-md bg-muted/40 p-2"
                                        >
                                            <p className="text-xs font-medium">{conexao.title}</p>
                                            <p className="mt-1 text-[11px] text-muted-foreground">
                                                Termos em comum:{" "}
                                                {conexao.matchedTerms.join(", ")}
                                            </p>
                                        </li>
                                    ))}
                                </ul>
                            )
                        ) : null}
                    </div>
                )}
            </aside>
        </div>
    );
}
