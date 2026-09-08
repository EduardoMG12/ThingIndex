"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Loader2, MessageSquare } from "lucide-react";
import ChatTab from "@/components/dashboard/ChatTab";

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

    const carregarContexto = useCallback((collectionId: number, signal: AbortSignal) => {
        setCarregando(true);
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

    return (
        <AppShell>
            <div className="flex h-full flex-col gap-4 p-6">
                <header className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <h1 className="text-xl font-semibold">{t("workspace.title")}</h1>
                        <p className="text-sm text-muted-foreground">{t("workspace.description")}</p>
                    </div>

                    <div className="flex flex-wrap items-center gap-3">
                    <button
                        type="button"
                        onClick={() => setChatAberto((atual) => !atual)}
                        aria-pressed={chatAberto}
                        data-testid="alternar-chat"
                        className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm ${chatAberto ? "border-primary/40 bg-primary/10 text-primary" : "border-border bg-card text-muted-foreground hover:text-foreground"}`}
                    >
                        <MessageSquare className="h-4 w-4" aria-hidden />
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
                    <div className="flex min-h-0 flex-1 flex-col gap-4 lg:flex-row">
                        {chatAberto ? (
                            <aside
                                data-testid="painel-chat"
                                aria-label={t("workspace.chat.toggle")}
                                className="max-h-[40vh] w-full shrink-0 overflow-hidden rounded-lg border border-border lg:h-[calc(100vh-9rem)] lg:max-h-none lg:w-[380px]"
                            >
                                <ChatTab
                                    className="h-full"
                                    collectionId={Number(context.collection.id)}
                                    compacto
                                    onVerNoDocumento={(source) => {
                                        if (source.documentId) {
                                            canvasRef.current?.irAte(source.documentId, source.page, source.nodeId);
                                        }
                                    }}
                                />
                            </aside>
                        ) : null}
                        <div className="min-w-0 flex-1">
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
