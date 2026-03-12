"use client";

import { useEffect, useState } from "react";
import { fetchCollections } from "@/lib/api/collections";

export function useCollections() {
    const [collections, setCollections] = useState<string[]>([]);
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

    const mergeCollection = (collection: string) => {
        const normalizedCollection = collection.trim();
        if (!normalizedCollection) {
            return;
        }

        setCollections((current) => {
            const alreadyExists = current.some(
                (item) => item.trim().toLowerCase() === normalizedCollection.toLowerCase()
            );

            if (alreadyExists) {
                return current;
            }

            return [...current, normalizedCollection].sort((left, right) => left.localeCompare(right, "pt-BR"));
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
