"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { fetchDocuments, type Document } from "@/lib/api/documents";

const POLL_INTERVAL_MS = 3000;

const PROCESSING_STATUSES = new Set(["PENDING", "UPLOADING", "QUEUED", "PROCESSING"]);

function hasActiveProcessing(docs: Document[]): boolean {
    return docs.some((doc) => PROCESSING_STATUSES.has(doc.status));
}

// O polling traz a lista completa: quem nao veio foi excluido (em outra aba,
// por exemplo). O merge so mantem a ordem e reaproveita objetos iguais.
function sameDocument(a: Document, b: Document): boolean {
    return a === b || JSON.stringify(a) === JSON.stringify(b);
}

function mergeDocuments(current: Document[], updated: Document[]): Document[] {
    const currentById = new Map(current.map((doc) => [doc.id, doc]));
    const orderedIds = current.map((doc) => doc.id);
    const updatedById = new Map(updated.map((doc) => [doc.id, doc]));

    const merged: Document[] = [];
    for (const id of orderedIds) {
        const fresh = updatedById.get(id);
        if (fresh) {
            const previous = currentById.get(id)!;
            // Mesma referencia para o preview nao refazer fetch a cada rodada.
            merged.push(sameDocument(previous, fresh) ? previous : fresh);
        }
    }
    for (const doc of updated) {
        if (!currentById.has(doc.id)) {
            merged.push(doc);
        }
    }
    const unchanged = merged.length === current.length && merged.every((doc, index) => doc === current[index]);
    return unchanged ? current : merged;
}

export function useDocuments() {
    const [documents, setDocuments] = useState<Document[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const pollControllerRef = useRef<AbortController | null>(null);

    const stopPolling = useCallback(() => {
        if (pollTimerRef.current !== null) {
            clearTimeout(pollTimerRef.current);
            pollTimerRef.current = null;
        }
        if (pollControllerRef.current) {
            pollControllerRef.current.abort();
            pollControllerRef.current = null;
        }
    }, []);

    const startPolling = useCallback(() => {
        stopPolling();

        const controller = new AbortController();
        pollControllerRef.current = controller;

        const poll = async () => {
            if (controller.signal.aborted) {
                return;
            }

            try {
                const data = await fetchDocuments(controller.signal);
                setDocuments((current) => mergeDocuments(current, data));
                setError(null);

                if (hasActiveProcessing(data)) {
                    pollTimerRef.current = setTimeout(poll, POLL_INTERVAL_MS);
                } else {
                    pollControllerRef.current = null;
                }
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                // Loga para a falha de polling nao ficar invisivel.
                console.warn("polling de documentos falhou; nova tentativa com backoff", err);

                pollTimerRef.current = setTimeout(poll, POLL_INTERVAL_MS * 2);
            }
        };

        pollTimerRef.current = setTimeout(poll, POLL_INTERVAL_MS);
    }, [stopPolling]);

    // Estavel: sai como reloadDocuments e entra em deps de efeito.
    const loadDocuments = useCallback(async (signal?: AbortSignal) => {
        try {
            setIsLoading(true);
            setError(null);

            const data = await fetchDocuments(signal);
            setDocuments(data);

            if (hasActiveProcessing(data)) {
                startPolling();
            } else {
                stopPolling();
            }
        } catch (err) {
            if (signal?.aborted) {
                return;
            }

            setError(err instanceof Error ? err.message : "Nao foi possivel carregar os documentos.");
            stopPolling();
        } finally {
            if (!signal?.aborted) {
                setIsLoading(false);
            }
        }
    }, [startPolling, stopPolling]);

    useEffect(() => {
        const controller = new AbortController();

        void loadDocuments(controller.signal);

        return () => {
            controller.abort();
            stopPolling();
        };
    }, [loadDocuments, stopPolling]);

    useEffect(() => {
        return () => {
            stopPolling();
        };
    }, [stopPolling]);

    return { documents, isLoading, error, reloadDocuments: loadDocuments, setDocuments };
}
