"use client";

import { useEffect, useState } from "react";

import { fetchCollectionContext } from "@/lib/api/collections";
import type { CollectionContext } from "@/lib/api/contracts/document-context";
import { fetchDocumentIndex, type Document, type DocumentIndex } from "@/lib/api/documents";
import type { AppLocale } from "@/lib/i18n/config";
import { translate } from "@/lib/i18n/translate";

const PREVIEW_POLL_INTERVAL_MS = 3000;

// Depende dos campos, nao do objeto: o polling de useDocuments cria objetos
// novos a cada 3 s e o preview refazia os fetches.
export function useDocumentPreviewData(document: Document | null, locale: AppLocale) {
    const [context, setContext] = useState<CollectionContext | null>(null);
    const [documentIndex, setDocumentIndex] = useState<DocumentIndex | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const documentId = document?.id ?? null;
    const documentStatus = document?.status;
    const documentUpdatedAt = document?.updated_at;
    const collectionId = document?.collection_id ?? null;

    useEffect(() => {
        if (documentId === null) {
            setContext(null);
            setDocumentIndex(null);
            setIsLoading(false);
            setError(null);
            return;
        }

        const controller = new AbortController();
        let pollTimer: ReturnType<typeof setTimeout> | null = null;

        async function loadContext(id: number) {
            try {
                setIsLoading(true);
                setError(null);
                const [indexData, contextData] = await Promise.all([
                    fetchDocumentIndex(id, controller.signal),
                    collectionId
                        ? fetchCollectionContext(collectionId, controller.signal)
                        : Promise.resolve(null),
                ]);
                setDocumentIndex(indexData);
                setContext(contextData);

                const currentStatus = indexData.document?.status ?? documentStatus;
                if (!controller.signal.aborted && (currentStatus === "QUEUED" || currentStatus === "PROCESSING")) {
                    pollTimer = setTimeout(() => {
                        void loadContext(id);
                    }, PREVIEW_POLL_INTERVAL_MS);
                }
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : translate(locale, "documentsPage.errors.loadIndexStructure"));
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadContext(documentId);

        return () => {
            controller.abort();
            if (pollTimer) {
                clearTimeout(pollTimer);
            }
        };
    }, [documentId, documentStatus, documentUpdatedAt, collectionId, locale]);

    return { context, documentIndex, isLoading, error };
}
