"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { fetchDocuments, type Document } from "@/lib/api/documents";

const POLL_INTERVAL_MS = 3000;

const PROCESSING_STATUSES = new Set(["PENDING", "UPLOADING", "QUEUED", "PROCESSING"]);

function hasActiveProcessing(docs: Document[]): boolean {
    return docs.some((doc) => PROCESSING_STATUSES.has(doc.status));
}

function mergeDocuments(current: Document[], updated: Document[]): Document[] {
    const byId = new Map(current.map((doc) => [doc.id, doc]));
    for (const doc of updated) {
        byId.set(doc.id, doc);
    }
    return Array.from(byId.values());
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
                pollTimerRef.current = setTimeout(poll, POLL_INTERVAL_MS * 2);
            }
        };

        pollTimerRef.current = setTimeout(poll, POLL_INTERVAL_MS);
    }, [stopPolling]);

    async function loadDocuments(signal?: AbortSignal) {
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
    }

    useEffect(() => {
        const controller = new AbortController();

        void loadDocuments(controller.signal);

        return () => {
            controller.abort();
            stopPolling();
        };
    }, []);

    useEffect(() => {
        return () => {
            stopPolling();
        };
    }, [stopPolling]);

    return { documents, isLoading, error, reloadDocuments: loadDocuments, setDocuments };
}
