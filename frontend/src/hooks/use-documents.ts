"use client";

import { useEffect, useState } from "react";

import { fetchDocuments, type Document } from "@/lib/api/documents";

export function useDocuments() {
    const [documents, setDocuments] = useState<Document[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    async function loadDocuments(signal?: AbortSignal) {
        try {
            setIsLoading(true);
            setError(null);

            const data = await fetchDocuments(signal);
            setDocuments(data);
        } catch (err) {
            if (signal?.aborted) {
                return;
            }

            setError(err instanceof Error ? err.message : "Nao foi possivel carregar os documentos.");
        } finally {
            if (!signal?.aborted) {
                setIsLoading(false);
            }
        }
    }

    useEffect(() => {
        const controller = new AbortController();

        void loadDocuments(controller.signal);

        return () => controller.abort();
    }, []);

    return { documents, isLoading, error, reloadDocuments: loadDocuments, setDocuments };
}
