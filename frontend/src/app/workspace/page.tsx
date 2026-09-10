"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Loader2, PanelLeft, X } from "lucide-react";
import { toast } from "sonner";
import ChatTab from "@/components/dashboard/ChatTab";
import { DocumentosDaColecao } from "@/components/workspace/DocumentosDaColecao";
import UploadTab from "@/components/dashboard/UploadTab";
import { aindaIndexando } from "@/lib/documents/status";
import { deleteDocument, retryDocumentIndexing } from "@/lib/api/documents";

import AppShell from "@/components/layout/AppShell";
import { useI18n } from "@/lib/i18n/provider";
import { WorkspaceCanvas, type WorkspaceCanvasHandle } from "@/components/workspace/WorkspaceCanvas";
import { fetchCollectionContext, fetchCollections, fetchHighlights, fetchLegend, fetchViewState } from "@/lib/api/collections";
import type { Highlight, Legend } from "@/lib/api/contracts/highlights";
import type { WorkspaceViewState } from "@/lib/api/contracts/view-state";
import type {
    CollectionContext,
    CollectionSummary,
} from "@/lib/api/contracts/document-context";

/**
 * Area de trabalho: painel navegavel com os documentos de uma colecao.
 *
 * A tela le APENAS dado que a indexacao deterministica ja gravou -- titulos,
 * resumos e hierarquia de secoes. Nao ha chamada de LLM em nenhum caminho, e e
 * por isso que ela abre instantaneamente. As conexoes entre trechos sao
 * derivadas sob demanda pelo mesmo scorer relacional das consultas normais.
 */
const PAINEL_LARGURA_PADRAO = 340;
const PAINEL_LARGURA_MIN = 280;
const PAINEL_LARGURA_MAX = 640;
const PAINEL_LARGURA_CHAVE = "thingindex.workspace.painel.largura";

function WorkspacePageInner() {
    const { t } = useI18n();
    const router = useRouter();
    const pathname = usePathname();
    const searchParams = useSearchParams();
    const canvasRef = useRef<WorkspaceCanvasHandle>(null);
    const [collections, setCollections] = useState<CollectionSummary[]>([]);
    const [selecionada, setSelecionada] = useState<number | null>(null);
    // Chat a esquerda do canvas: aberto por padrao em telas largas, fechado
    // nas estreitas (o painel empilha em cima do canvas). Nao persiste.
    const [chatAberto, setChatAberto] = useState(false);
    // Largura do painel: arrastavel pela divisa, lembrada no navegador.
    const [larguraPainel, setLarguraPainel] = useState(PAINEL_LARGURA_PADRAO);
    useEffect(() => {
        try {
            const salva = Number(window.localStorage.getItem(PAINEL_LARGURA_CHAVE));
            if (Number.isFinite(salva) && salva >= PAINEL_LARGURA_MIN && salva <= PAINEL_LARGURA_MAX) {
                // eslint-disable-next-line react-hooks/set-state-in-effect -- so na montagem, le a preferencia
                setLarguraPainel(salva);
            }
        } catch {
            // sem localStorage, fica o padrao
        }
    }, []);
    const arrastoDaDivisa = useRef<{ x0: number; largura0: number } | null>(null);
    const aoArrastarDivisa = (event: React.PointerEvent<HTMLDivElement>) => {
        const inicio = arrastoDaDivisa.current;
        if (!inicio) {
            return;
        }
        const proxima = Math.min(PAINEL_LARGURA_MAX, Math.max(PAINEL_LARGURA_MIN, inicio.largura0 + event.clientX - inicio.x0));
        setLarguraPainel(proxima);
    };
    const aoSoltarDivisa = (event: React.PointerEvent<HTMLDivElement>) => {
        arrastoDaDivisa.current = null;
        event.currentTarget.releasePointerCapture(event.pointerId);
        try {
            window.localStorage.setItem(PAINEL_LARGURA_CHAVE, String(larguraPainel));
        } catch {
            // preferencia nao salva; nada a fazer
        }
    };
    const [aba, setAba] = useState<"chat" | "documentos">("chat");
    const [uploadAberto, setUploadAberto] = useState(false);
    // Depois de um envio, recarrega o contexto a cada 5 s ate nenhum documento
    // estar mais em indexacao; o quadro novo aparece sem recarregar a pagina.
    const [acompanhando, setAcompanhando] = useState(false);
    useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect -- so na montagem, para ler a largura da tela
        setChatAberto(window.matchMedia("(min-width: 1024px)").matches);
    }, []);
    const colecaoDaUrl = Number(searchParams.get("collection"));
    const [context, setContext] = useState<CollectionContext | null>(null);
    // Marcas, legenda e estado chegam junto com o contexto para o canvas abrir
    // ja no lugar certo, sem pulo; falha neles nao impede a tela.
    const [anexos, setAnexos] = useState<{ marcas: Highlight[]; legenda: Legend; estado: WorkspaceViewState }>({ marcas: [], legenda: {}, estado: {} });
    const [carregando, setCarregando] = useState(true);
    const [erro, setErro] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function carregarColecoes() {
            try {
                const lista = await fetchCollections(controller.signal);
                if (controller.signal.aborted) {
                    return;
                }
                setCollections(lista);
                if (lista.length > 0) {
                    // ?collection= vale quando existe na lista; senao a primeira.
                    const pedida = lista.find((c) => Number(c.id) === colecaoDaUrl);
                    const primeiro = (pedida ?? lista[0]).id;
                    setSelecionada(typeof primeiro === "number" ? primeiro : Number(primeiro));
                }
            } catch (falha) {
                if (!controller.signal.aborted) {
                    setErro(falha instanceof Error ? falha.message : t("workspace.loadCollectionsError"));
                }
            } finally {
                if (!controller.signal.aborted) {
                    setCarregando(false);
                }
            }
        }

        void carregarColecoes();
        return () => controller.abort();
        // eslint-disable-next-line react-hooks/exhaustive-deps -- so na montagem; depois a URL e escrita por nos
    }, [t]);

    const trocarColecao = useCallback(
        (id: number) => {
            setSelecionada(id);
            const params = new URLSearchParams(searchParams.toString());
            params.set("collection", String(id));
            params.delete("conversation");
            params.delete("doc");
            router.replace(`${pathname}?${params.toString()}`, { scroll: false });
        },
        [pathname, router, searchParams]
    );

    // `silencioso`: recarrega sem o loader de tela cheia, para o canvas nao
    // desmontar (posicoes, zoom e selecao ficam) depois de apagar/reindexar.
    const carregarContexto = useCallback((collectionId: number, signal: AbortSignal, silencioso = false) => {
        if (!silencioso) {
            setCarregando(true);
        }
        setErro(null);

        Promise.all([
            fetchCollectionContext(collectionId, signal),
            fetchHighlights(collectionId, signal).catch(() => [] as Highlight[]),
            fetchLegend(collectionId, signal).catch(() => ({}) as Legend),
            fetchViewState(collectionId, signal).catch(() => ({}) as WorkspaceViewState),
        ])
            .then(([resultado, marcas, legenda, estado]) => {
                if (!signal.aborted) {
                    setAnexos({ marcas, legenda, estado });
                    setContext(resultado);
                }
            })
            .catch((falha: unknown) => {
                if (!signal.aborted) {
                    setErro(falha instanceof Error ? falha.message : t("workspace.loadContextError"));
                }
            })
            .finally(() => {
                if (!signal.aborted) {
                    setCarregando(false);
                }
            });
    }, [t]);

    useEffect(() => {
        if (selecionada === null) {
            return;
        }
        const controller = new AbortController();
        carregarContexto(selecionada, controller.signal);
        return () => controller.abort();
    }, [selecionada, carregarContexto]);

    const recarregarContexto = useCallback(() => {
        if (selecionada !== null) {
            carregarContexto(selecionada, new AbortController().signal, true);
        }
    }, [selecionada, carregarContexto]);

    const contextRef = useRef(context);
    useEffect(() => {
        contextRef.current = context;
    }, [context]);
    useEffect(() => {
        if (!acompanhando) {
            return;
        }
        // A decisao de parar e tomada no tick, com o contexto mais recente:
        // num efeito ela rodaria com o contexto velho (sem o documento novo)
        // e pararia antes da primeira recarga.
        const timer = window.setInterval(() => {
            const atual = contextRef.current;
            if (atual && !atual.documents.some((d) => aindaIndexando(d.status))) {
                setAcompanhando(false);
                return;
            }
            recarregarContexto();
        }, 5000);
        return () => window.clearInterval(timer);
    }, [acompanhando, recarregarContexto]);

    const acaoNoDocumento = useCallback(
        async (acao: () => Promise<unknown>) => {
            try {
                await acao();
            } catch (falha) {
                toast.error(falha instanceof Error ? falha.message : t("workspace.documents.actionError"));
                return;
            }
            recarregarContexto();
        },
        [recarregarContexto, t]
    );

    return (
        <AppShell contentClassName="max-w-none p-0">
            {/* Canvas dominante, como Figma/Miro: cabecalho de uma linha e o
                resto da altura para o canvas. A descricao vira tooltip. */}
            <div className="flex h-[calc(100vh-4.5rem)] min-h-[560px] flex-col gap-2 p-3">
                <header className="flex flex-wrap items-center justify-between gap-3 px-1">
                    <h1 className="text-base font-semibold" title={t("workspace.description")}>{t("workspace.title")}</h1>

                    <div className="flex flex-wrap items-center gap-3">
                    <button
                        type="button"
                        onClick={() => setChatAberto((atual) => !atual)}
                        aria-pressed={chatAberto}
                        data-testid="alternar-chat"
                        className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm ${chatAberto ? "border-primary/40 bg-primary/10 text-primary" : "border-border bg-card text-muted-foreground hover:text-foreground"}`}
                    >
                        <PanelLeft className="h-4 w-4" aria-hidden />
                        {t("workspace.chat.toggle")}
                    </button>
                    {collections.length > 0 ? (
                        <label className="flex items-center gap-2 text-sm">
                            <span className="text-muted-foreground">{t("workspace.collectionLabel")}</span>
                            <select
                                value={selecionada ?? ""}
                                onChange={(event) => trocarColecao(Number(event.target.value))}
                                className="rounded-md border border-border bg-card px-3 py-1.5 text-sm"
                            >
                                {collections.map((collection) => (
                                    <option key={collection.id} value={collection.id}>
                                        {collection.name}
                                    </option>
                                ))}
                            </select>
                        </label>
                    ) : null}
                    </div>
                </header>

                {erro ? (
                    <p className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                        {erro}
                    </p>
                ) : null}

                {carregando ? (
                    <div className="flex flex-1 items-center justify-center">
                        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                ) : collections.length === 0 ? (
                    <p className="text-sm text-muted-foreground">{t("workspace.noCollections")}</p>
                ) : context ? (
                    <div className="flex min-h-0 flex-1 flex-col gap-2 lg:flex-row">
                        {chatAberto ? (
                            <aside
                                data-testid="painel-chat"
                                aria-label={t("workspace.chat.toggle")}
                                className="flex max-h-[40vh] w-full shrink-0 flex-col overflow-hidden rounded-lg border border-border lg:h-full lg:max-h-none lg:w-[var(--painel)]"
                                style={{ "--painel": `${larguraPainel}px` } as React.CSSProperties}
                            >
                                <div role="tablist" className="flex shrink-0 items-stretch border-b border-border text-sm">
                                    {(["chat", "documentos"] as const).map((nome) => (
                                        <button
                                            key={nome}
                                            type="button"
                                            role="tab"
                                            aria-selected={aba === nome}
                                            data-testid={`aba-${nome}`}
                                            onClick={() => setAba(nome)}
                                            className={`flex-1 px-3 py-2 ${aba === nome ? "border-b-2 border-primary font-medium text-foreground" : "text-muted-foreground hover:text-foreground"}`}
                                        >
                                            {t(nome === "chat" ? "workspace.chat.tab" : "workspace.documents.tab")}
                                        </button>
                                    ))}
                                    <button
                                        type="button"
                                        onClick={() => setChatAberto(false)}
                                        aria-label={t("workspace.chat.close")}
                                        title={t("workspace.chat.close")}
                                        data-testid="fechar-painel"
                                        className="px-2 text-muted-foreground hover:text-foreground"
                                    >
                                        <X className="h-4 w-4" aria-hidden />
                                    </button>
                                </div>
                                {/* O chat fica montado escondido para nao perder a conversa ao trocar de aba. */}
                                <div className="min-h-0 flex-1" hidden={aba !== "chat"}>
                                    <ChatTab
                                        className="h-full"
                                        collectionId={Number(context.collection.id)}
                                        compacto
                                        indexadoEm={new Map(context.documents.filter((d) => d.indexedAt).map((d) => [d.id, d.indexedAt as string]))}
                                        onVerNoDocumento={(source) => {
                                            if (!source.documentId) {
                                                return;
                                            }
                                            // Fonte de linha de tabela: o id carrega "table-<id>-row-<n>";
                                            // realcar a tabela e mais util do que so abrir a pagina.
                                            const tabela = /table-(\d+)-row-/.exec(source.id ?? "");
                                            if (tabela) {
                                                canvasRef.current?.realcarTabela(Number(tabela[1]));
                                                return;
                                            }
                                            canvasRef.current?.irAte(source.documentId, source.page, source.nodeId, source.paragraphId, source.lines, source.box ? { box: source.box, lineBoxes: source.lineBoxes ?? [] } : undefined);
                                        }}
                                    />
                                </div>
                                <div className="min-h-0 flex-1" hidden={aba !== "documentos"}>
                                    <DocumentosDaColecao
                                        documentos={context.documents}
                                        onFocar={(id) => canvasRef.current?.focarDocumento(id)}
                                        onReindexar={(id) => acaoNoDocumento(() => retryDocumentIndexing(id))}
                                        onApagar={(id) => acaoNoDocumento(() => deleteDocument(id))}
                                        onEnviar={() => setUploadAberto(true)}
                                    />
                                </div>
                            </aside>
                        ) : null}
                        {chatAberto ? (
                            <div
                                role="separator"
                                aria-orientation="vertical"
                                aria-label={t("workspace.chat.resize")}
                                title={t("workspace.chat.resize")}
                                data-testid="redimensionar-painel"
                                className="hidden w-2 shrink-0 cursor-col-resize rounded hover:bg-primary/20 lg:block"
                                onPointerDown={(event) => {
                                    arrastoDaDivisa.current = { x0: event.clientX, largura0: larguraPainel };
                                    event.currentTarget.setPointerCapture(event.pointerId);
                                }}
                                onPointerMove={aoArrastarDivisa}
                                onPointerUp={aoSoltarDivisa}
                                onPointerCancel={aoSoltarDivisa}
                            />
                        ) : null}
                        <div className="min-h-0 min-w-0 flex-1">
                            <WorkspaceCanvas
                                ref={canvasRef}
                                key={String(context.collection.id)}
                                context={context}
                                marcasIniciais={anexos.marcas}
                                legendaInicial={anexos.legenda}
                                estadoInicial={anexos.estado}
                            />
                        </div>
                    </div>
                ) : null}
            </div>
            {uploadAberto && context ? (
                <div
                    className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
                    onClick={() => setUploadAberto(false)}
                    onKeyDown={(event) => {
                        if (event.key === "Escape") {
                            setUploadAberto(false);
                        }
                    }}
                >
                    <div
                        role="dialog"
                        aria-modal="true"
                        aria-label={t("workspace.documents.uploadTitle", { collection: context.collection.name })}
                        data-testid="modal-de-upload"
                        className="w-full max-w-2xl rounded-lg bg-background shadow-xl"
                        onClick={(event) => event.stopPropagation()}
                    >
                        <div className="flex items-center justify-between border-b border-border px-4 py-3">
                            <h2 className="text-sm font-semibold">{t("workspace.documents.uploadTitle", { collection: context.collection.name })}</h2>
                            <button type="button" onClick={() => setUploadAberto(false)} aria-label={t("common.dangerConfirm.closeLabel")} className="rounded p-1 hover:bg-muted">
                                <X className="h-4 w-4" aria-hidden />
                            </button>
                        </div>
                        <div className="p-4">
                            <UploadTab
                                colecaoFixa={context.collection.name}
                                aoConcluir={() => {
                                    setAcompanhando(true);
                                    recarregarContexto();
                                }}
                            />
                        </div>
                    </div>
                </div>
            ) : null}
        </AppShell>
    );
}

// useSearchParams exige um limite de Suspense na pre-renderizacao.
export default function WorkspacePage() {
    return (
        <Suspense fallback={null}>
            <WorkspacePageInner />
        </Suspense>
    );
}
