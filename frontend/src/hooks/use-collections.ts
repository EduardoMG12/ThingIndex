"use client";

import { useEffect, useState } from "react";
import { fetchCollections } from "@/lib/api/collections";
import {
    getCollectionName,
    type CollectionSummary,
} from "@/lib/api/contracts/document-context";

function normalizeCollectionName(value: string) {
    return value.trim().toLowerCase();
}

export function useCollections() {
    const [collections, setCollections] = useState<CollectionSummary[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function loadInitialCollections() {
            try {
                setIsLoading(true);
                setError(null);

                const data = await fetchCollections(controller.signal);
                setCollections(data);
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }

                setError(err instanceof Error ? err.message : "Nao foi possivel carregar as collections.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadInitialCollections();

        return () => controller.abort();
    }, []);

    const mergeCollection = (collection: string | CollectionSummary) => {
        const nextCollection =
            typeof collection === "string"
                ? {
                    id: `draft:${normalizeCollectionName(collection)}`,
                    name: collection.trim(),
                    documentCount: 0,
                    metadata: {},
                }
                : collection;

        const normalizedCollection = normalizeCollectionName(getCollectionName(nextCollection));
        if (!normalizedCollection) {
            return;
        }

        setCollections((current) => {
            const alreadyExists = current.some(
                (item) => normalizeCollectionName(item.name) === normalizedCollection
            );

            if (alreadyExists) {
                return current.map((item) =>
                    normalizeCollectionName(item.name) === normalizedCollection ? nextCollection : item
                );
            }

            return [...current, nextCollection].sort((left, right) =>
                left.name.localeCompare(right.name, "pt-BR")
            );
        });
    };

    const reloadCollections = async () => {
        try {
            setIsLoading(true);
            setError(null);

            const data = await fetchCollections();
            setCollections(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Nao foi possivel carregar as collections.");
        } finally {
            setIsLoading(false);
        }
    };

    return {
        collections,
        isLoading,
        error,
        mergeCollection,
        reloadCollections,
    };
}
