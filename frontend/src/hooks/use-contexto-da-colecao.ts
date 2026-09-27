"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { fetchCollectionContext, fetchHighlights, fetchLegend, fetchViewState } from "@/lib/api/collections";
import type { CollectionContext } from "@/lib/api/contracts/document-context";
import type { Highlight, Legend } from "@/lib/api/contracts/highlights";
import type { WorkspaceViewState } from "@/lib/api/contracts/view-state";

export type AnexosDoContexto = { marcas: Highlight[]; legenda: Legend; estado: WorkspaceViewState };

// Carrega contexto, marcas, legenda e estado do canvas da colecao selecionada.
//
// Toda carga (a da troca de colecao e as recargas silenciosas do polling ou de
// apagar/reindexar) passa pelo mesmo AbortController: a nova aborta a anterior,
// e a troca de colecao e o unmount abortam a pendente. Antes a recarga usava um
// controller descartavel, e uma resposta atrasada da colecao A podia chegar
// depois da B e mostrar documentos da colecao errada.
export function useContextoDaColecao(selecionada: number | null, mensagemDeErro: string) {
    const [context, setContext] = useState<CollectionContext | null>(null);
    const [anexos, setAnexos] = useState<AnexosDoContexto>({ marcas: [], legenda: {}, estado: {} });
    // Qual colecao ja terminou de carregar (com sucesso ou erro). "carregando" e
    // derivado disso, sem setState sincrono dentro do efeito; a recarga
    // silenciosa nao mexe aqui, entao nao mostra o loader de tela cheia.
    const [concluidaPara, setConcluidaPara] = useState<number | null>(null);
    const [falha, setFalha] = useState<{ collectionId: number; mensagem: string } | null>(null);

    const controllerRef = useRef<AbortController | null>(null);
    const selecionadaRef = useRef(selecionada);

    const carregarContexto = useCallback((collectionId: number) => {
        controllerRef.current?.abort();
        const controller = new AbortController();
        controllerRef.current = controller;
        const { signal } = controller;
        const valeAinda = () => !signal.aborted && selecionadaRef.current === collectionId;

        Promise.all([
            fetchCollectionContext(collectionId, signal),
            fetchHighlights(collectionId, signal).catch(() => [] as Highlight[]),
            fetchLegend(collectionId, signal).catch(() => ({}) as Legend),
            fetchViewState(collectionId, signal).catch(() => ({}) as WorkspaceViewState),
        ])
            .then(([resultado, marcas, legenda, estado]) => {
                if (valeAinda()) {
                    setAnexos({ marcas, legenda, estado });
                    setContext(resultado);
                    setFalha(null);
                }
            })
            .catch((motivo: unknown) => {
                if (valeAinda()) {
                    setFalha({ collectionId, mensagem: motivo instanceof Error ? motivo.message : mensagemDeErro });
                }
            })
            .finally(() => {
                if (valeAinda()) {
                    setConcluidaPara(collectionId);
                }
            });
    }, [mensagemDeErro]);

    useEffect(() => {
        selecionadaRef.current = selecionada;
        if (selecionada === null) {
            return;
        }
        carregarContexto(selecionada);
        return () => controllerRef.current?.abort();
    }, [selecionada, carregarContexto]);

    const recarregar = useCallback(() => {
        if (selecionada !== null) {
            carregarContexto(selecionada);
        }
    }, [selecionada, carregarContexto]);

    const carregando = selecionada !== null && concluidaPara !== selecionada;
    const erro = falha && falha.collectionId === selecionada ? falha.mensagem : null;

    return { context, anexos, carregando, erro, recarregar };
}
