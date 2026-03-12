"use client";

import { useEffect, useState } from "react";

import { fetchDocuments, type Document } from "@/lib/api/documents";

export function useDocuments() {
    const [documents, setDocuments] = useState<Document[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function loadDocuments() {
            try {
                setIsLoading(true);
                setError(null);

                const data = await fetchDocuments(controller.signal);
                setDocuments(data);
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : "Nao foi possivel carregar os documentos.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadDocuments();

        return () => controller.abort();
    }, []);

    return { documents, isLoading, error };
}
