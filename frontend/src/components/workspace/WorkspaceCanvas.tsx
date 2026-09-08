"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Maximize2, Minus, Plus } from "lucide-react";

import { fetchNodeConnections } from "@/lib/api/collections";
import { readBox, type CollectionContext, type OutlineNode, type ParagraphNode } from "@/lib/api/contracts/document-context";
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
import {
    AvisoSemPosicao,
    origemMajoritaria,
    PageOverlay,
    sectionKeyDe,
    type ParagrafoNaPagina,
} from "@/components/workspace/PageOverlay";
import { ConnectionLayer } from "@/components/workspace/ConnectionLayer";
import { idDaLinha, montarLinhas, paragrafosSecundarios, type Linha } from "@/lib/workspace/connections";
import { boxParaPixels } from "@/lib/workspace/geometry";

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
/**
 * Linhas desenhadas por vez. Uma secao de edital devolveu 109 conexoes acima
 * do limiar: desenhar todas vira um novelo ilegivel. A API ja ordena por
 * score, entao as N primeiras sao as mais fortes; a sidebar lista todas.
 */
export const LINHAS_MAXIMAS = 12;
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
    const paragrafos = useMemo(() => context.paragraphNodes ?? [], [context.paragraphNodes]);

    // Indices por documento: secao por slug, paragrafos por pagina (com a secao
    // resolvida por section_key === slug), secoes por pagina de inicio, e a
    // contagem de box para o aviso de reindexacao. Tudo derivado do contexto
    // ja carregado -- nenhuma chamada extra.
    const indiceDoContexto = useMemo(() => {
        const secaoPorSlug = new Map<number, Map<string, OutlineNode>>();
        for (const no of nosDoSumario) {
            if (!no.slug) {
                continue;
            }
            const mapa = secaoPorSlug.get(no.documentId) ?? new Map<string, OutlineNode>();
            mapa.set(no.slug, no);
            secaoPorSlug.set(no.documentId, mapa);
        }
        const porNodeId = new Map<string, OutlineNode>();
        for (const no of nosDoSumario) {
            porNodeId.set(no.nodeId, no);
        }
        const porPagina = new Map<number, Map<number, ParagrafoNaPagina[]>>();
        const ancora = new Map<number, Map<string, ParagraphNode>>();
        const porSequencia = new Map<number, Map<number, ParagraphNode>>();
        const contagem = new Map<number, { total: number; comBox: number }>();
        for (const no of paragrafos) {
            const c = contagem.get(no.documentId) ?? { total: 0, comBox: 0 };
            c.total += 1;
            const box = readBox(no);
            if (box) {
                c.comBox += 1;
                const seq = porSequencia.get(no.documentId) ?? new Map<number, ParagraphNode>();
                seq.set(no.sequenceOrder, no);
                porSequencia.set(no.documentId, seq);
            }
            contagem.set(no.documentId, c);
            const chave = sectionKeyDe(no);
            const secao = chave ? (secaoPorSlug.get(no.documentId)?.get(chave) ?? null) : null;
            if (box && chave) {
                const mapa = ancora.get(no.documentId) ?? new Map<string, ParagraphNode>();
                if (!mapa.has(chave)) {
                    mapa.set(chave, no);
                }
                ancora.set(no.documentId, mapa);
            }
            if (!no.pageNumber) {
                continue;
            }
            const paginas = porPagina.get(no.documentId) ?? new Map<number, ParagrafoNaPagina[]>();
            const lista = paginas.get(no.pageNumber) ?? [];
            lista.push({ no, secao });
            paginas.set(no.pageNumber, lista);
            porPagina.set(no.documentId, paginas);
        }
        const secoesPorPagina = new Map<number, Map<number, OutlineNode[]>>();
        for (const no of nosDoSumario) {
            if (!no.startPage) {
                continue;
            }
            const paginas = secoesPorPagina.get(no.documentId) ?? new Map<number, OutlineNode[]>();
            const lista = paginas.get(no.startPage) ?? [];
            lista.push(no);
            paginas.set(no.startPage, lista);
            secoesPorPagina.set(no.documentId, paginas);
        }
        return { secaoPorSlug, porNodeId, porPagina, ancora, porSequencia, contagem, secoesPorPagina };
    }, [nosDoSumario, paragrafos]);

    // Ancora de uma secao: o paragrafo em que ela comeca (paragraph_start, que
    // a indexacao grava no no) se ele tiver box; senao o primeiro paragrafo
    // com o section_key dela; senao nada -- um marcador em posicao inventada
    // seria pior que nenhum.
    const ancoraDaSecao = useCallback(
        (secao: OutlineNode): ParagraphNode | null => {
            const inicio = secao.metadata?.paragraph_start;
            if (typeof inicio === "number") {
                const porInicio = indiceDoContexto.porSequencia.get(secao.documentId)?.get(inicio);
                if (porInicio) {
                    return porInicio;
                }
            }
            return secao.slug ? (indiceDoContexto.ancora.get(secao.documentId)?.get(secao.slug) ?? null) : null;
        },
        [indiceDoContexto]
    );

    const [paragrafoSelecionado, setParagrafoSelecionado] = useState<{ no: ParagraphNode; secao: OutlineNode | null } | null>(null);
    const [linhaRealcada, setLinhaRealcada] = useState<string | null>(null);

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
        centralizarEm,
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

    const indiceDoDocumento = useMemo(() => new Map(documentos.map((d, i) => [d.id, i])), [documentos]);

    /** Rect, em coordenadas do mundo, da pagina N de um documento medido. */
    const rectDaPagina = useCallback(
        (documentId: number, numero: number): Rect | null => {
            const i = indiceDoDocumento.get(documentId);
            const paginas = medidas[documentId];
            if (i === undefined || !paginas || numero < 1 || numero > paginas.length) {
                return null;
            }
            const quadro = quadros[i];
            const pagina = paginas[numero - 1];
            return {
                x: quadro.x + (quadro.largura - pagina.largura) / 2,
                y: quadro.y + topoDaPagina(paginas, numero),
                largura: pagina.largura,
                altura: pagina.altura,
            };
        },
        [indiceDoDocumento, medidas, quadros]
    );

    /** Rect do grifo de um paragrafo no mundo, ou null sem box ou sem pagina medida. */
    const rectDoParagrafo = useCallback(
        (no: ParagraphNode): Rect | null => {
            const box = readBox(no);
            if (!box || !no.pageNumber) {
                return null;
            }
            const pagina = rectDaPagina(no.documentId, no.pageNumber);
            if (!pagina) {
                return null;
            }
            const px = boxParaPixels(box, pagina.largura, pagina.altura);
            return { x: pagina.x + px.left, y: pagina.y + px.top, largura: px.width, altura: px.height };
        },
        [rectDaPagina]
    );

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
        setParagrafoSelecionado(null);
    }, []);

    const selecionarParagrafo = useCallback((no: ParagraphNode, secao: OutlineNode | null) => {
        setParagrafoSelecionado({ no, secao });
        setDocumentoFocado(no.documentId);
        setNoSelecionado(secao?.nodeId ?? null);
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

    const secundarios = useMemo(
        () => (conexoes ? paragrafosSecundarios(conexoes, nosDoSumario, paragrafos) : new Set<ParagraphNode["id"]>()),
        [conexoes, nosDoSumario, paragrafos]
    );

    const linhas = useMemo<Linha[]>(() => {
        if (!conexoes || conexoes.length === 0) {
            return [];
        }
        // Origem: o grifo do paragrafo selecionado; sem ele, o marcador da
        // secao pedida na sidebar; sem box, o quadro do documento focado.
        const secaoDeOrigem = noSelecionado ? indiceDoContexto.porNodeId.get(noSelecionado) : undefined;
        const origem =
            (paragrafoSelecionado ? rectDoParagrafo(paragrafoSelecionado.no) : null) ??
            (secaoDeOrigem ? (() => { const a = ancoraDaSecao(secaoDeOrigem); return a ? rectDoParagrafo(a) : null; })() : null) ??
            (documentoFocado !== null && indiceDoDocumento.has(documentoFocado) ? quadros[indiceDoDocumento.get(documentoFocado)!] : null);
        return montarLinhas(conexoes.slice(0, LINHAS_MAXIMAS), {
            origem,
            destinoPreciso: (conexao) => {
                const no = indiceDoContexto.porNodeId.get(conexao.nodeId);
                if (!no || no.documentId !== conexao.documentId) {
                    return null;
                }
                const a = ancoraDaSecao(no);
                return a ? rectDoParagrafo(a) : null;
            },
            quadroDoDocumento: (documentId) => {
                const i = indiceDoDocumento.get(documentId);
                return i === undefined ? null : quadros[i];
            },
        });
    }, [conexoes, noSelecionado, paragrafoSelecionado, documentoFocado, indiceDoContexto, indiceDoDocumento, quadros, ancoraDaSecao, rectDoParagrafo]);

    const irParaConexao = useCallback(
        (conexao: { documentId: number; nodeId: string }) => {
            const no = indiceDoContexto.porNodeId.get(conexao.nodeId);
            const a = no ? ancoraDaSecao(no) : null;
            const alvo = (a ? rectDoParagrafo(a) : null) ?? (indiceDoDocumento.has(conexao.documentId) ? quadros[indiceDoDocumento.get(conexao.documentId)!] : null);
            if (alvo) {
                centralizarEm({ ...alvo, altura: Math.min(alvo.altura, 600) }, 1);
            }
        },
        [indiceDoContexto, ancoraDaSecao, rectDoParagrafo, indiceDoDocumento, quadros, centralizarEm]
    );

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
                                        renderizarSobreposicao={(numero, tamanhoDaPagina) => {
                                            const nosDaPagina = indiceDoContexto.porPagina.get(documento.id)?.get(numero) ?? [];
                                            const secoes = indiceDoContexto.secoesPorPagina.get(documento.id)?.get(numero) ?? [];
                                            return (
                                                <PageOverlay
                                                    numero={numero}
                                                    tamanho={tamanhoDaPagina}
                                                    paragrafos={nosDaPagina}
                                                    secoes={secoes}
                                                    ancoraDaSecao={ancoraDaSecao}
                                                    origem={origemMajoritaria(nosDaPagina.map((p) => p.no))}
                                                    selecionadoId={paragrafoSelecionado?.no.id ?? null}
                                                    secundarios={secundarios}
                                                    onSelecionar={selecionarParagrafo}
                                                />
                                            );
                                        }}
                                    >
                                        {(() => {
                                            const c = indiceDoContexto.contagem.get(documento.id);
                                            return c && c.total > 0 && c.comBox === 0 ? (
                                                <AvisoSemPosicao documentId={documento.id} totalParagrafos={c.total} />
                                            ) : null;
                                        })()}
                                    </PdfFrame>
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
                    <ConnectionLayer
                        linhas={linhas}
                        realcada={linhaRealcada}
                        onRealcar={setLinhaRealcada}
                        onEscolher={irParaConexao}
                    />
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

                        {(() => {
                            const c = indiceDoContexto.contagem.get(documentoFocado);
                            const semPosicao = c ? c.total - c.comBox : 0;
                            return semPosicao > 0 ? (
                                <p className="text-xs text-muted-foreground" data-testid="sem-posicao">
                                    {semPosicao} de {c!.total} trechos sem posição na página.
                                </p>
                            ) : null;
                        })()}

                        {paragrafoSelecionado && paragrafoSelecionado.no.documentId === documentoFocado ? (
                            <div className="rounded-md border border-primary/60 bg-primary/5 p-3" data-testid="trecho-selecionado">
                                <p className="text-[10px] uppercase tracking-wide text-muted-foreground">
                                    Trecho selecionado · p. {paragrafoSelecionado.no.pageNumber ?? "?"}
                                    {paragrafoSelecionado.secao ? ` · ${paragrafoSelecionado.secao.title}` : ""}
                                </p>
                                <p className="mt-1 line-clamp-6 text-xs">{paragrafoSelecionado.no.content}</p>
                                <button
                                    type="button"
                                    onClick={() => paragrafoSelecionado.secao && pedirConexoes(paragrafoSelecionado.secao.nodeId)}
                                    disabled={!paragrafoSelecionado.secao}
                                    title={paragrafoSelecionado.secao ? undefined : "Trecho sem seção no sumário: não há nó para consultar."}
                                    className="mt-2 text-xs font-medium text-primary underline-offset-2 hover:underline disabled:cursor-not-allowed disabled:opacity-50"
                                >
                                    Ver conexões
                                </button>
                            </div>
                        ) : null}

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
                                    {conexoes.length > LINHAS_MAXIMAS ? (
                                        <li className="text-[11px] text-muted-foreground" data-testid="aviso-linhas-limitadas">
                                            {conexoes.length} conexões acima do limiar; as {LINHAS_MAXIMAS} mais fortes estão desenhadas no canvas.
                                        </li>
                                    ) : null}
                                    {conexoes.map((conexao) => {
                                        const id = idDaLinha(conexao);
                                        return (
                                            <li
                                                key={id}
                                                className={`rounded-md p-2 ${linhaRealcada === id ? "bg-primary/15 ring-1 ring-primary/50" : "bg-muted/40"}`}
                                                onMouseEnter={() => setLinhaRealcada(id)}
                                                onMouseLeave={() => setLinhaRealcada(null)}
                                            >
                                                <button
                                                    type="button"
                                                    onClick={() => irParaConexao(conexao)}
                                                    className="w-full text-left"
                                                    title="Centralizar no destino"
                                                >
                                                    <p className="text-xs font-medium">
                                                        {conexao.title}
                                                        <span className="ml-2 rounded bg-primary/10 px-1 text-[10px] text-primary">
                                                            {conexao.score} termos
                                                        </span>
                                                    </p>
                                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                                        Termos em comum:{" "}
                                                        {conexao.matchedTerms.join(", ")}
                                                    </p>
                                                </button>
                                            </li>
                                        );
                                    })}
                                </ul>
                            )
                        ) : null}
                    </div>
                )}
            </aside>
        </div>
    );
}
