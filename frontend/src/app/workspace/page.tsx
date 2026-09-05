"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";

import AppShell from "@/components/layout/AppShell";
import { WorkspaceCanvas } from "@/components/workspace/WorkspaceCanvas";
import { fetchCollectionContext, fetchCollections } from "@/lib/api/collections";
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
export default function WorkspacePage() {
    const [collections, setCollections] = useState<CollectionSummary[]>([]);
    const [selecionada, setSelecionada] = useState<number | null>(null);
    const [context, setContext] = useState<CollectionContext | null>(null);
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
                    const primeiro = lista[0].id;
                    setSelecionada(typeof primeiro === "number" ? primeiro : Number(primeiro));
                }
            } catch (falha) {
                if (!controller.signal.aborted) {
                    setErro(falha instanceof Error ? falha.message : "Falha ao carregar coleções.");
                }
            } finally {
                if (!controller.signal.aborted) {
                    setCarregando(false);
                }
            }
        }

        void carregarColecoes();
        return () => controller.abort();
    }, []);

    const carregarContexto = useCallback((collectionId: number, signal: AbortSignal) => {
        setCarregando(true);
        setErro(null);

        fetchCollectionContext(collectionId, signal)
            .then((resultado) => {
                if (!signal.aborted) {
                    setContext(resultado);
                }
            })
            .catch((falha: unknown) => {
                if (!signal.aborted) {
                    setErro(
                        falha instanceof Error ? falha.message : "Falha ao carregar a coleção."
                    );
                }
            })
            .finally(() => {
                if (!signal.aborted) {
                    setCarregando(false);
                }
            });
    }, []);

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
                        <h1 className="text-xl font-semibold">Área de trabalho</h1>
                        <p className="text-sm text-muted-foreground">
                            Explore os documentos da coleção e as conexões entre seus trechos.
                        </p>
                    </div>

                    {collections.length > 0 ? (
                        <label className="flex items-center gap-2 text-sm">
                            <span className="text-muted-foreground">Coleção</span>
                            <select
                                value={selecionada ?? ""}
                                onChange={(event) => setSelecionada(Number(event.target.value))}
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
                    <p className="text-sm text-muted-foreground">
                        Nenhuma coleção ainda. Envie um documento para começar.
                    </p>
                ) : context ? (
                    <div className="flex-1">
                        <WorkspaceCanvas context={context} />
                    </div>
                ) : null}
            </div>
        </AppShell>
    );
}
