"use client";

import { useEffect, useState } from "react";

import {
    fetchProviderRuntimeOverview,
    type ProviderRuntimeOverview,
} from "@/lib/api/providers";

const EMPTY_OVERVIEW: ProviderRuntimeOverview = {
    storage: null,
    indexing: null,
    chat: null,
};

export function useProviderRuntimeOverview() {
    const [overview, setOverview] = useState<ProviderRuntimeOverview>(EMPTY_OVERVIEW);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function loadOverview(signal?: AbortSignal) {
            try {
                setIsLoading(true);
                setError(null);
                setOverview(await fetchProviderRuntimeOverview(signal));
            } catch (err) {
                if (signal?.aborted) {
                    return;
                }

                setOverview(EMPTY_OVERVIEW);
                setError(err instanceof Error ? err.message : "Nao foi possivel carregar o runtime ativo dos providers.");
            } finally {
                if (!signal?.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadOverview(controller.signal);

        return () => controller.abort();
    }, []);

    return {
        overview,
        isLoading,
        error,
    };
}
