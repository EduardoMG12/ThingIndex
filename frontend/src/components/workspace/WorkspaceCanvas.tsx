"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { ChevronLeft, ChevronRight, Maximize2, Minus, Plus } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";

import { deleteHighlight, fetchNodeConnections, putHighlight, putLegend } from "@/lib/api/collections";
import type { WorkspaceViewState } from "@/lib/api/contracts/view-state";
import { useSalvarEstado } from "@/components/workspace/useSalvarEstado";
import { Tag } from "lucide-react";
import { LegendPanel } from "@/components/workspace/LegendPanel";
import { classesDaCor, CORES, rotuloDaCor, type HighlightColor as CorDeMarca } from "@/lib/workspace/highlights";
import type { Highlight, HighlightColor, Legend } from "@/lib/api/contracts/highlights";
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
import { APRESENTACAO_PADRAO, limitarPagina, precisaDoBotaoIrPara, regraIrAte, type Apresentacao } from "@/lib/workspace/highlights";

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
    /** Carregados pela pagina junto com o contexto, para o canvas abrir ja no estado certo. */
    marcasIniciais?: Highlight[];
    legendaInicial?: Legend;
    estadoInicial?: WorkspaceViewState;
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

function chaveDeApresentacao(collectionId: number | string): string {
    return `thingindex.workspace.apresentacao.${collectionId}`;
}

/**
 * Migracao unica do que ficava no localStorage (posicoes e apresentacao)
 * para o estado no servidor: le, apaga as chaves e devolve o que achou.
 * Quando o servidor ja tem estado, o local e so apagado.
 */
function migrarDoLocalStorage(collectionId: number | string, servidorTemEstado: boolean): Pick<WorkspaceViewState, "quadros" | "apresentacao"> | null {
    if (typeof window === "undefined") {
        return null;
    }
    const quadros = lerPosicoesSalvas(collectionId);
    const apresentacao = lerApresentacoesSalvas(collectionId);
    try {
        window.localStorage.removeItem(chaveDeArmazenamento(collectionId));
        window.localStorage.removeItem(chaveDeApresentacao(collectionId));
    } catch {
        // sem localStorage nao ha o que migrar
    }
    if (servidorTemEstado || (Object.keys(quadros).length === 0 && Object.keys(apresentacao).length === 0)) {
        return null;
    }
    return { quadros, apresentacao };
}

function lerApresentacoesSalvas(collectionId: number | string): Record<string, Apresentacao> {
    if (typeof window === "undefined") {
        return {};
    }
    try {
        const bruto = window.localStorage.getItem(chaveDeApresentacao(collectionId));
        const analisado = bruto ? (JSON.parse(bruto) as unknown) : null;
        if (typeof analisado !== "object" || analisado === null) {
            return {};
        }
        const saida: Record<string, Apresentacao> = {};
        for (const [id, valor] of Object.entries(analisado as Record<string, unknown>)) {
            const v = valor as Partial<Apresentacao> | null;
            if (v && (v.modo === "folhas" || v.modo === "pagina") && typeof v.paginaAtual === "number") {
                saida[id] = { modo: v.modo, paginaAtual: v.paginaAtual };
            }
        }
        return saida;
    } catch {
        return {};
    }
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

export function WorkspaceCanvas({ context, marcasIniciais = [], legendaInicial = {}, estadoInicial = {} }: WorkspaceCanvasProps) {
    const { t } = useI18n();
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
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

    // Marcas coloridas e legenda: carregadas junto com a colecao, salvas de
    // forma otimista (a cor aparece na hora; se a API falhar, volta e avisa).
    const [marcas, setMarcas] = useState<Map<ParagraphNode["id"], HighlightColor>>(
        () => new Map(marcasIniciais.map((m) => [m.paragraphNodeId as ParagraphNode["id"], m.color]))
    );
    const [legenda, setLegenda] = useState<Legend>(legendaInicial);
    const [erroDeMarca, setErroDeMarca] = useState<string | null>(null);


    const marcar = useCallback(
        (no: ParagraphNode, cor: HighlightColor | null) => {
            const id = Number(no.id);
            if (!Number.isFinite(id)) {
                return;
            }
            const anterior = marcas.get(no.id) ?? null;
            if (anterior === cor) {
                return;
            }
            setErroDeMarca(null);
            setMarcas((atual) => {
                const proxima = new Map(atual);
                if (cor) {
                    proxima.set(no.id, cor);
                } else {
                    proxima.delete(no.id);
                }
                return proxima;
            });
            const pedido = cor ? putHighlight(collectionId, id, cor) : deleteHighlight(collectionId, id);
            pedido.catch((falha: unknown) => {
                // Rollback: a cor volta ao que era e o erro aparece na sidebar.
                setMarcas((atual) => {
                    const proxima = new Map(atual);
                    if (anterior) {
                        proxima.set(no.id, anterior);
                    } else {
                        proxima.delete(no.id);
                    }
                    return proxima;
                });
                setErroDeMarca(falha instanceof Error ? falha.message : t("workspace.highlight.saveError"));
            });
        },
        [collectionId, marcas, t]
    );

    // Lazy init + `key={collectionId}` na pagina: trocar de colecao remonta o
    // canvas, entao nao precisa de efeito para recarregar as posicoes.
    // Estado de visualizacao: vem do servidor (props); o que estava no
    // localStorage e migrado uma vez e apagado.
    const servidorTemEstado = Boolean(estadoInicial.viewport || estadoInicial.quadros || estadoInicial.apresentacao);
    const [migrado] = useState(() => migrarDoLocalStorage(collectionId, servidorTemEstado));
    const [posicoes, setPosicoes] = useState<Record<string, Posicao>>(() => estadoInicial.quadros ?? migrado?.quadros ?? {});
    const [apresentacoes, setApresentacoes] = useState<Record<string, Apresentacao>>(
        () => estadoInicial.apresentacao ?? migrado?.apresentacao ?? {}
    );
    const [marcadoresVisiveis, setMarcadoresVisiveis] = useState<boolean>(estadoInicial.marcadoresVisiveis ?? true);
    const apresentacaoDe = useCallback(
        (documentId: number): Apresentacao => apresentacoes[String(documentId)] ?? APRESENTACAO_PADRAO,
        [apresentacoes]
    );
    const mudarApresentacao = useCallback(
        (documentId: number, proxima: Apresentacao) => {
            setApresentacoes((atual) => ({ ...atual, [String(documentId)]: proxima }));
        },
        []
    );
    // Foco compartilhavel pela URL (?doc=): lido uma vez ao montar (o canvas
    // remonta por colecao), escrito com replace para nao empilhar historico.
    const [documentoFocadoBruto, setDocumentoFocado] = useState<number | null>(() => {
        const doc = Number(searchParams.get("doc"));
        // ?doc= vence o estado salvo; sem ele, volta ao ultimo documento focado.
        return Number.isFinite(doc) && doc > 0 ? doc : (estadoInicial.documentoFocado ?? null);
    });
    // Um ?doc= de outra colecao nao aponta para nada aqui: vale como "nenhum".
    const documentoFocado = documentoFocadoBruto !== null && documentos.some((d) => d.id === documentoFocadoBruto) ? documentoFocadoBruto : null;

    useEffect(() => {
        const params = new URLSearchParams(searchParams.toString());
        if (documentoFocado === null) {
            params.delete("doc");
        } else {
            params.set("doc", String(documentoFocado));
        }
        const next = params.toString();
        if (next !== searchParams.toString()) {
            router.replace(next ? `${pathname}?${next}` : pathname, { scroll: false });
        }
        // Converge: o replace muda searchParams, o efeito roda de novo e
        // next === current impede a segunda escrita (mesmo padrao do chat).
    }, [documentoFocado, pathname, router, searchParams]);
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
    } = useViewport(estadoInicial.viewport);

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
                return {
                    x: p.x,
                    y: p.y,
                    largura: larguraDoQuadro(paginas),
                    altura: alturaDoQuadro(paginas, apresentacaoDe(documento.id)),
                };
            }),
        [documentos, posicaoDe, medidas, apresentacaoDe]
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
            const apresentacao = apresentacaoDe(documento.id);
            paginas.forEach((pagina, i) => {
                // Em "pagina" so a pagina atual existe no canvas.
                if (apresentacao.modo === "pagina" && i + 1 !== limitarPagina(apresentacao.paginaAtual, paginas.length)) {
                    return;
                }
                candidatas.push({
                    id: `${documento.id}:${i + 1}`,
                    rect: {
                        x: quadro.x + (quadro.largura - pagina.largura) / 2,
                        y: quadro.y + topoDaPagina(paginas, i + 1, apresentacao),
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
    }, [documentos, medidas, quadros, viewport, tamanho, apresentacaoDe]);

    const conjuntoVazio = useMemo(() => new Set<number>(), []);

    const indiceDoDocumento = useMemo(() => new Map(documentos.map((d, i) => [d.id, i])), [documentos]);

    /**
     * Rect, em coordenadas do mundo, da pagina N de um documento medido.
     * `comoMostrada` calcula onde a pagina VAI ficar se virar a atual (para
     * centralizar depois de trocar de pagina); sem ele, pagina fora da tela em
     * modo "pagina" devolve null, e a linha de conexao cai na borda do quadro.
     */
    const rectDaPagina = useCallback(
        (documentId: number, numero: number, comoMostrada = false): Rect | null => {
            const i = indiceDoDocumento.get(documentId);
            const paginas = medidas[documentId];
            if (i === undefined || !paginas || numero < 1 || numero > paginas.length) {
                return null;
            }
            const apresentacao = apresentacaoDe(documentId);
            if (apresentacao.modo === "pagina" && !comoMostrada && limitarPagina(apresentacao.paginaAtual, paginas.length) !== numero) {
                return null;
            }
            const quadro = quadros[i];
            const pagina = paginas[numero - 1];
            return {
                x: quadro.x + (quadro.largura - pagina.largura) / 2,
                y: quadro.y + topoDaPagina(paginas, numero, apresentacao),
                largura: pagina.largura,
                altura: pagina.altura,
            };
        },
        [indiceDoDocumento, medidas, quadros, apresentacaoDe]
    );

    /**
     * Ir ate uma pagina de um documento: em "pagina" troca a pagina atual do
     * quadro antes de centralizar; em "folhas" so centraliza.
     */
    const irAte = useCallback(
        (documentId: number, pagina: number | null, rectPreferido: Rect | null) => {
            const paginas = medidas[documentId];
            const apresentacao = apresentacaoDe(documentId);
            if (pagina !== null && paginas) {
                const alvoPagina = limitarPagina(pagina, paginas.length);
                const { trocarPagina } = regraIrAte(apresentacao, alvoPagina);
                if (trocarPagina !== null) {
                    mudarApresentacao(documentId, { modo: "pagina", paginaAtual: trocarPagina });
                }
                const rect = (trocarPagina === null ? rectPreferido : null) ?? rectDaPagina(documentId, alvoPagina, true);
                if (rect) {
                    centralizarEm({ ...rect, altura: Math.min(rect.altura, 600) }, 1);
                    return;
                }
            }
            const i = indiceDoDocumento.get(documentId);
            const quadro = rectPreferido ?? (i === undefined ? null : quadros[i]);
            if (quadro) {
                centralizarEm({ ...quadro, altura: Math.min(quadro.altura, 600) }, 1);
            }
        },
        [medidas, apresentacaoDe, mudarApresentacao, rectDaPagina, indiceDoDocumento, quadros, centralizarEm]
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
        }

        window.addEventListener("pointermove", aoMover);
        window.addEventListener("pointerup", aoSoltar);
        return () => {
            window.removeEventListener("pointermove", aoMover);
            window.removeEventListener("pointerup", aoSoltar);
        };
    }, []);

    /**
     * Focar um documento e tambem traze-lo ao centro da tela, no zoom atual:
     * o topo do quadro (cabecalho + primeira pagina) fica enquadrado.
     */
    const focarDocumento = useCallback(
        (documentId: number) => {
            setDocumentoFocado(documentId);
            setConexoes(null);
            setNoSelecionado(null);
            const i = indiceDoDocumento.get(documentId);
            if (i === undefined) {
                return;
            }
            const quadro = quadros[i];
            const alturaVisivel = tamanho.altura / Math.max(viewport.escala, 0.1);
            centralizarEm({ ...quadro, altura: Math.min(quadro.altura, alturaVisivel) }, viewport.escala);
        },
        [indiceDoDocumento, quadros, tamanho, viewport.escala, centralizarEm]
    );

    const limparSelecao = useCallback(() => {
        setDocumentoFocado(null);
        setNoSelecionado(null);
        setConexoes(null);
        setParagrafoSelecionado(null);
    }, []);

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
                setErroConexoes(erro instanceof Error ? erro.message : t("workspace.sidebar.connectionsError"));
            } finally {
                setCarregandoConexoes(false);
            }
        },
        [collectionId, t]
    );

    // Indice da conexao atual no navegador (-1 = nenhuma escolhida ainda).
    const [indiceConexao, setIndiceConexao] = useState(-1);

    const selecionarParagrafo = useCallback(
        (no: ParagraphNode, secao: OutlineNode | null) => {
            setParagrafoSelecionado({ no, secao });
            setDocumentoFocado(no.documentId);
            setNoSelecionado(secao?.nodeId ?? null);
            setConexoes(null);
            setIndiceConexao(-1);
            // Grifo colorido ja e um "quero ver com quem isso conversa": busca na hora.
            if (secao && marcas.has(no.id)) {
                void pedirConexoes(secao.nodeId);
            }
        },
        [marcas, pedirConexoes]
    );

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


    /** Centraliza a pagina em que a secao comeca (ou o quadro do documento). */
    const irParaSecao = useCallback(
        (no: OutlineNode) => {
            const a = ancoraDaSecao(no);
            irAte(no.documentId, a?.pageNumber ?? no.startPage ?? null, a ? rectDoParagrafo(a) : null);
        },
        [ancoraDaSecao, rectDoParagrafo, irAte]
    );

    const documentoAtual = documentos.find((documento) => documento.id === documentoFocado);

    const estadoAtual = useMemo<WorkspaceViewState>(
        () => ({
            viewport,
            quadros: posicoes,
            apresentacao: apresentacoes,
            marcadoresVisiveis,
            documentoFocado: documentoFocado ?? undefined,
        }),
        [viewport, posicoes, apresentacoes, marcadoresVisiveis, documentoFocado]
    );
    const aoFalharSalvar = useCallback(
        (erro: unknown) => setErroDeMarca(erro instanceof Error ? erro.message : t("workspace.state.saveError")),
        [t]
    );
    useSalvarEstado(collectionId, estadoAtual, { ativo: collectionId > 0, salvarLogo: migrado !== null, aoFalhar: aoFalharSalvar });

    const salvarLegenda = useCallback(
        (proxima: Legend) => {
            const anterior = legenda;
            setLegenda(proxima);
            putLegend(collectionId, proxima).catch((falha: unknown) => {
                setLegenda(anterior);
                setErroDeMarca(falha instanceof Error ? falha.message : t("workspace.legend.saveError"));
            });
        },
        [collectionId, legenda, t]
    );

    const nomeDaCor = useCallback((cor: CorDeMarca) => t(`workspace.highlight.colors.${cor}`), [t]);

    // Marcas do documento focado, por cor, para a legenda e o bloco da sidebar.
    const [filtroDeCor, setFiltroDeCor] = useState<CorDeMarca | null>(null);
    const marcasDoFocado = useMemo(() => {
        if (documentoFocado === null) {
            return [];
        }
        return paragrafos
            .filter((no) => no.documentId === documentoFocado && marcas.has(no.id))
            .map((no) => ({ no, cor: marcas.get(no.id)! }));
    }, [paragrafos, marcas, documentoFocado]);
    const contagemDoFocado = useMemo(() => {
        const saida = Object.fromEntries(CORES.map((c) => [c, 0])) as Record<CorDeMarca, number>;
        for (const { cor } of marcasDoFocado) {
            saida[cor] += 1;
        }
        return saida;
    }, [marcasDoFocado]);


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

    const irParaIndice = useCallback(
        (indice: number) => {
            if (!conexoes || indice < 0 || indice >= conexoes.length) {
                return;
            }
            setIndiceConexao(indice);
            const conexao = conexoes[indice];
            setLinhaRealcada(idDaLinha(conexao));
            const no = indiceDoContexto.porNodeId.get(conexao.nodeId);
            const a = no ? ancoraDaSecao(no) : null;
            irAte(conexao.documentId, a?.pageNumber ?? no?.startPage ?? conexao.startPage ?? null, a ? rectDoParagrafo(a) : null);
        },
        [conexoes, indiceDoContexto, ancoraDaSecao, rectDoParagrafo, irAte]
    );

    const navegador = useMemo(
        () =>
            conexoes !== null || carregandoConexoes
                ? { total: conexoes?.length ?? 0, indice: indiceConexao, carregando: carregandoConexoes, onIr: irParaIndice }
                : null,
        [conexoes, carregandoConexoes, indiceConexao, irParaIndice]
    );
    const conexaoAtual = conexoes && indiceConexao >= 0 ? conexoes[indiceConexao] : null;
    const textoDoDestino = useMemo(() => {
        if (!conexaoAtual) {
            return null;
        }
        const no = indiceDoContexto.porNodeId.get(conexaoAtual.nodeId);
        const a = no ? ancoraDaSecao(no) : null;
        return a ? a.content.slice(0, 300) : null;
    }, [conexaoAtual, indiceDoContexto, ancoraDaSecao]);

    const irParaConexao = useCallback(
        (conexao: { documentId: number; nodeId: string; startPage?: number | null }) => {
            const no = indiceDoContexto.porNodeId.get(conexao.nodeId);
            const a = no ? ancoraDaSecao(no) : null;
            const pagina = a?.pageNumber ?? no?.startPage ?? conexao.startPage ?? null;
            irAte(conexao.documentId, pagina, a ? rectDoParagrafo(a) : null);
        },
        [indiceDoContexto, ancoraDaSecao, rectDoParagrafo, irAte]
    );

    return (
        // Empilha em telas estreitas e divide em duas colunas a partir de lg.
        // `min-w-0` e obrigatorio: sem ele o item flex assume a largura do
        // conteudo e o canvas colapsa para dezenas de pixels em 780px (medido).
        <div className="flex h-full w-full flex-col gap-4 lg:flex-row">
            <div
                ref={containerRef}
                className={`relative h-[60vh] w-full min-w-0 flex-1 overflow-hidden rounded-lg border border-border bg-muted/20 touch-none lg:h-[calc(100vh-9rem)] ${
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
                                        subtitulo={t("workspace.frame.chunks", { count: trechos })}
                                        visivel={quadroVisivel(quadros[indice])}
                                        paginasVivas={paginasVivasPorDocumento.get(documento.id) ?? conjuntoVazio}
                                        escalaDeRenderizacao={escalaRender}
                                        modoMiniatura={modoMiniatura}
                                        focado={focado}
                                        onMedidas={aoMedir}
                                        onFocar={focarDocumento}
                                        onPointerDownCabecalho={(event) => aoIniciarArrasto(event, documento.id, indice)}
                                        apresentacao={apresentacaoDe(documento.id)}
                                        onMudarApresentacao={mudarApresentacao}
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
                                                    marcas={marcas}
                                                    legenda={legenda}
                                                    corSecundaria={paragrafoSelecionado ? (marcas.get(paragrafoSelecionado.no.id) ?? null) : null}
                                                    onMarcar={marcar}
                                                    onVerConexoes={(_, secao) => pedirConexoes(secao.nodeId)}
                                                    mostrarMarcadores={marcadoresVisiveis}
                                                    navegador={navegador}
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
                                    onClick={() => focarDocumento(documento.id)}
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
                                        {t("workspace.frame.chunksIndexed", { count: trechos })}
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

                <LegendPanel legenda={legenda} onSalvar={salvarLegenda} contagem={documentoFocado !== null ? contagemDoFocado : null} />

                <div
                    className="absolute bottom-3 right-3 flex items-center gap-1 rounded-md border border-border bg-card/95 p-1 shadow-sm"
                    role="toolbar"
                    aria-label={t("workspace.canvas.toolbar")}
                    onPointerDown={(event) => event.stopPropagation()}
                >
                    <button
                        type="button"
                        onClick={afastar}
                        aria-label={t("workspace.canvas.zoomOut")}
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
                        aria-label={t("workspace.canvas.zoomIn")}
                        className="rounded p-1.5 hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                    >
                        <Plus className="h-4 w-4" aria-hidden />
                    </button>
                    <button
                        type="button"
                        onClick={() => setMarcadoresVisiveis((v) => !v)}
                        aria-label={t("workspace.canvas.markers")}
                        aria-pressed={marcadoresVisiveis}
                        title={t("workspace.canvas.markers")}
                        data-testid="alternar-marcadores"
                        className={`rounded p-1.5 hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary ${marcadoresVisiveis ? "" : "opacity-50"}`}
                    >
                        <Tag className="h-4 w-4" aria-hidden />
                    </button>
                    <button
                        type="button"
                        onClick={() => enquadrar(quadros)}
                        aria-label={t("workspace.canvas.fit")}
                        className="rounded p-1.5 hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
                    >
                        <Maximize2 className="h-4 w-4" aria-hidden />
                    </button>
                </div>
            </div>

            <aside
                className="max-h-[40vh] w-full shrink-0 overflow-y-auto rounded-lg border border-border bg-card p-4 lg:max-h-[calc(100vh-9rem)] lg:w-72"
                aria-label={t("workspace.sidebar.label")}
            >
                {documentoFocado === null ? (
                    <p className="text-sm text-muted-foreground">{t("workspace.sidebar.empty")}</p>
                ) : (
                    <div className="space-y-4">
                        <h2 className="text-sm font-semibold">{documentoAtual?.title}</h2>

                        {(() => {
                            const c = indiceDoContexto.contagem.get(documentoFocado);
                            const semPosicao = c ? c.total - c.comBox : 0;
                            return semPosicao > 0 ? (
                                <p className="text-xs text-muted-foreground" data-testid="sem-posicao">
                                    {t("workspace.sidebar.noPosition", { count: semPosicao, total: c!.total })}
                                </p>
                            ) : null;
                        })()}

                        {erroDeMarca ? (
                            <p className="text-xs text-destructive" role="alert" data-testid="erro-de-marca">
                                {erroDeMarca}
                            </p>
                        ) : null}

                        {marcasDoFocado.length > 0 ? (
                            <div className="rounded-md border border-border p-3" data-testid="bloco-de-marcas">
                                <p className="text-[10px] uppercase tracking-wide text-muted-foreground">{t("workspace.marks.title")}</p>
                                <div className="mt-2 flex flex-wrap gap-1">
                                    {CORES.filter((cor) => contagemDoFocado[cor] > 0).map((cor) => (
                                        <button
                                            key={cor}
                                            type="button"
                                            onClick={() => setFiltroDeCor(filtroDeCor === cor ? null : cor)}
                                            aria-pressed={filtroDeCor === cor}
                                            className={`flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] ${
                                                filtroDeCor === cor ? "border-foreground" : "border-border"
                                            }`}
                                            data-testid={`filtro-${cor}`}
                                        >
                                            <span className={`h-2.5 w-2.5 rounded-full ${classesDaCor(cor).chip}`} aria-hidden />
                                            {rotuloDaCor(legenda, cor, nomeDaCor)}
                                            <span className="tabular-nums text-muted-foreground">{contagemDoFocado[cor]}</span>
                                        </button>
                                    ))}
                                </div>
                                {filtroDeCor ? (
                                    <ul className="mt-2 space-y-1" data-testid="lista-de-marcas">
                                        {marcasDoFocado
                                            .filter(({ cor }) => cor === filtroDeCor)
                                            .map(({ no }) => (
                                                <li key={String(no.id)}>
                                                    <button
                                                        type="button"
                                                        title={t("workspace.marks.goTo")}
                                                        onClick={() => {
                                                            const secao = sectionKeyDe(no)
                                                                ? (indiceDoContexto.secaoPorSlug.get(no.documentId)?.get(sectionKeyDe(no)!) ?? null)
                                                                : null;
                                                            selecionarParagrafo(no, secao);
                                                            irAte(no.documentId, no.pageNumber ?? null, rectDoParagrafo(no));
                                                        }}
                                                        className="w-full rounded px-1 py-0.5 text-left text-xs hover:bg-muted"
                                                    >
                                                        <span className="mr-1 text-muted-foreground">{t("workspace.marks.page", { page: no.pageNumber ?? "?" })}</span>
                                                        {no.content.slice(0, 80)}
                                                    </button>
                                                </li>
                                            ))}
                                    </ul>
                                ) : null}
                            </div>
                        ) : null}

                        {paragrafoSelecionado && paragrafoSelecionado.no.documentId === documentoFocado ? (
                            <div className="rounded-md border border-primary/60 bg-primary/5 p-3" data-testid="trecho-selecionado">
                                <p className="text-[10px] uppercase tracking-wide text-muted-foreground">
                                    {t("workspace.sidebar.selected")} · {t("workspace.sidebar.page", { page: paragrafoSelecionado.no.pageNumber ?? "?" })}
                                    {paragrafoSelecionado.secao ? ` · ${paragrafoSelecionado.secao.title}` : ""}
                                </p>
                                <p className="mt-1 line-clamp-6 text-xs">{paragrafoSelecionado.no.content}</p>
                                <button
                                    type="button"
                                    onClick={() => paragrafoSelecionado.secao && pedirConexoes(paragrafoSelecionado.secao.nodeId)}
                                    disabled={!paragrafoSelecionado.secao}
                                    title={paragrafoSelecionado.secao ? undefined : t("workspace.sidebar.noSectionTooltip")}
                                    className="mt-2 text-xs font-medium text-primary underline-offset-2 hover:underline disabled:cursor-not-allowed disabled:opacity-50"
                                >
                                    {t("workspace.sidebar.viewConnections")}
                                </button>
                            </div>
                        ) : null}

                        {nosDoDocumentoFocado.length === 0 ? (
                            <p className="text-sm text-muted-foreground">{t("workspace.sidebar.noStructure")}</p>
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
                                        <button
                                            type="button"
                                            onClick={() => irParaSecao(no)}
                                            className="w-full text-left text-sm font-medium hover:text-primary"
                                            title={t("workspace.sidebar.goToSection")}
                                        >
                                            {no.title}
                                        </button>
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
                                            {t("workspace.sidebar.viewConnections")}
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        )}

                        {carregandoConexoes ? (
                            <p className="text-xs text-muted-foreground">{t("workspace.sidebar.loadingConnections")}</p>
                        ) : null}

                        {erroConexoes ? (
                            <p className="text-xs text-destructive">{erroConexoes}</p>
                        ) : null}

                        {conexaoAtual ? (
                            <div className="rounded-md border border-primary/60 bg-primary/5 p-3" data-testid="conexao-atual">
                                {/* Os mesmos ◀ ▶ da barra flutuante: a barra fica no paragrafo de
                                    origem, que sai da tela quando o canvas centraliza o destino. */}
                                <div className="flex items-center justify-between gap-2">
                                    <p className="text-[10px] uppercase tracking-wide text-muted-foreground">
                                        {t("workspace.navigator.current")} · {t("workspace.navigator.of", { i: indiceConexao + 1, total: conexoes!.length })}
                                    </p>
                                    <span className="flex items-center gap-0.5">
                                        <button
                                            type="button"
                                            onClick={() => irParaIndice(indiceConexao - 1)}
                                            disabled={indiceConexao <= 0}
                                            aria-label={t("workspace.navigator.prev")}
                                            className="rounded p-0.5 hover:bg-muted disabled:opacity-40"
                                        >
                                            <ChevronLeft className="h-4 w-4" aria-hidden />
                                        </button>
                                        <button
                                            type="button"
                                            onClick={() => irParaIndice(indiceConexao + 1)}
                                            disabled={indiceConexao >= conexoes!.length - 1}
                                            aria-label={t("workspace.navigator.next")}
                                            className="rounded p-0.5 hover:bg-muted disabled:opacity-40"
                                        >
                                            <ChevronRight className="h-4 w-4" aria-hidden />
                                        </button>
                                    </span>
                                </div>
                                <p className="mt-1 text-sm font-medium">{conexaoAtual.title}</p>
                                <p className="mt-1 text-[11px] text-muted-foreground">
                                    {t("workspace.sidebar.page", { page: conexaoAtual.startPage ?? "?" })} · {t("workspace.sidebar.commonTerms")} {conexaoAtual.matchedTerms.join(", ")}
                                </p>
                                {textoDoDestino ? <p className="mt-2 line-clamp-5 text-xs">{textoDoDestino}</p> : null}
                                <button
                                    type="button"
                                    onClick={() => irParaIndice(indiceConexao)}
                                    className="mt-2 text-xs font-medium text-primary underline-offset-2 hover:underline"
                                >
                                    {t("workspace.navigator.goTo")}
                                </button>
                            </div>
                        ) : null}

                        {conexoes !== null && !carregandoConexoes ? (
                            conexoes.length === 0 ? (
                                // Ausencia DECLARADA, nunca area em branco. Uma
                                // regiao vazia seria lida como falha de
                                // carregamento, e o sistema perderia credito no
                                // unico ponto em que sabe dizer "nao encontrei".
                                <p className="text-xs text-muted-foreground">{t("workspace.sidebar.noConnections")}</p>
                            ) : (
                                <ul className="space-y-2 border-t border-border pt-3">
                                    {conexoes.length > LINHAS_MAXIMAS ? (
                                        <li className="text-[11px] text-muted-foreground" data-testid="aviso-linhas-limitadas">
                                            {t("workspace.sidebar.tooManyLines", { total: conexoes.length, max: LINHAS_MAXIMAS })}
                                        </li>
                                    ) : null}
                                    {conexoes.map((conexao) => {
                                        const id = idDaLinha(conexao);
                                        const noDestino = indiceDoContexto.porNodeId.get(conexao.nodeId);
                                        const ancoraDestino = noDestino ? ancoraDaSecao(noDestino) : null;
                                        const paginaDestino = ancoraDestino?.pageNumber ?? noDestino?.startPage ?? conexao.startPage ?? null;
                                        const mostrarIrPara = precisaDoBotaoIrPara(
                                            apresentacaoDe(conexao.documentId),
                                            paginasVivasPorDocumento.get(conexao.documentId) ?? conjuntoVazio,
                                            paginaDestino
                                        );
                                        return (
                                            <li
                                                key={id}
                                                className={`rounded-md p-2 ${linhaRealcada === id ? "bg-primary/15 ring-1 ring-primary/50" : "bg-muted/40"}`}
                                                onMouseEnter={() => setLinhaRealcada(id)}
                                                onMouseLeave={() => setLinhaRealcada(null)}
                                            >
                                                <button
                                                    type="button"
                                                    onClick={() => irParaIndice(conexoes.indexOf(conexao))}
                                                    className="w-full text-left"
                                                    title={t("workspace.sidebar.centerOnTarget")}
                                                >
                                                    <p className="text-xs font-medium">
                                                        {conexao.title}
                                                        <span className="ml-2 rounded bg-primary/10 px-1 text-[10px] text-primary">
                                                            {conexao.score === 1 ? t("workspace.sidebar.term") : t("workspace.sidebar.terms", { count: conexao.score })}
                                                        </span>
                                                    </p>
                                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                                        {paginaDestino !== null ? (
                                                            <span className="mr-2" data-testid="pagina-destino">
                                                                {t("workspace.sidebar.page", { page: paginaDestino })}
                                                            </span>
                                                        ) : null}
                                                        {t("workspace.sidebar.commonTerms")}{" "}
                                                        {conexao.matchedTerms.join(", ")}
                                                    </p>
                                                </button>
                                                {mostrarIrPara && paginaDestino !== null ? (
                                                    <button
                                                        type="button"
                                                        onClick={() => irAte(conexao.documentId, paginaDestino, ancoraDestino ? rectDoParagrafo(ancoraDestino) : null)}
                                                        className="mt-1 rounded border border-primary/40 px-2 py-0.5 text-[11px] font-medium text-primary hover:bg-primary/10"
                                                        data-testid="ir-para-pagina"
                                                    >
                                                        {t("workspace.sidebar.goToPage", { page: paginaDestino })}
                                                    </button>
                                                ) : null}
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
