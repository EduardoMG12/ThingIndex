"use client";

import { useEffect, useState } from "react";
import type React from "react";

import { fetchDocumentMonitoringSummary, type DocumentMonitoringSummary } from "@/lib/api/documents";
import { fetchPrices, fetchPricingSources, type ModelPrice, type PricingSource } from "@/lib/api/pricing";

type CostVisibilityState = {
    summary: DocumentMonitoringSummary | null;
    pricingSources: PricingSource[];
    prices: ModelPrice[];
    isLoading: boolean;
    error: string | null;
};

const INITIAL_STATE: CostVisibilityState = {
    summary: null,
    pricingSources: [],
    prices: [],
    isLoading: true,
    error: null,
};

export function useCostVisibility() {
    const [state, setState] = useState<CostVisibilityState>(INITIAL_STATE);

    useEffect(() => {
        const controller = new AbortController();
        void loadCostVisibility(setState, controller.signal);
        return () => controller.abort();
    }, []);

    const reload = async () => {
        await loadCostVisibility(setState);
    };

    return {
        summary: state.summary,
        pricingSources: state.pricingSources,
        prices: state.prices,
        isLoading: state.isLoading,
        error: state.error,
        reload,
    };
}

async function loadCostVisibility(
    setState: React.Dispatch<React.SetStateAction<CostVisibilityState>>,
    signal?: AbortSignal
) {
    setState((current) => ({
        ...current,
        isLoading: true,
        error: null,
    }));

    const [summaryResult, sourcesResult, pricesResult] = await Promise.allSettled([
        fetchDocumentMonitoringSummary(signal),
        fetchPricingSources(signal),
        fetchPrices(signal),
    ]);

    if (signal?.aborted) {
        return;
    }

    const errors = [summaryResult, sourcesResult, pricesResult]
        .filter((result): result is PromiseRejectedResult => result.status === "rejected")
        .map((result) => result.reason)
        .filter((reason): reason is Error => reason instanceof Error);

    setState({
        summary: summaryResult.status === "fulfilled" ? summaryResult.value : null,
        pricingSources: sourcesResult.status === "fulfilled" ? sourcesResult.value : [],
        prices: pricesResult.status === "fulfilled" ? pricesResult.value : [],
        isLoading: false,
        error: errors[0]?.message ?? null,
    });
}
