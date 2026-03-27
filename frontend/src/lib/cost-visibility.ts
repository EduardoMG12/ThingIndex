import type { DocumentMonitoringModelAggregate, DocumentMonitoringSummary } from "@/lib/api/documents";
import type { ModelPrice, PricingSource } from "@/lib/api/pricing";
import type { ProviderRuntimeOverview } from "@/lib/api/providers";
import { formatProviderTypeLabel, getProviderRuntimeModel } from "@/lib/provider-runtime";

export type CostVisibilityHealth = "trusted" | "partial" | "setup";
export type PricingSourceHealth = "synced" | "failed" | "pending" | "disabled";

export type CostVisibilitySourceStatus = {
    id: number;
    providerType: string;
    providerLabel: string;
    name: string;
    sourceURL: string;
    health: PricingSourceHealth;
    isActiveRuntime: boolean;
    lastSyncedAt?: string;
    lastPriceCount: number;
    lastSyncError?: string;
};

export type CostVisibilityOverview = {
    health: CostVisibilityHealth;
    activeProviderType: string | null;
    activeProviderLabel: string | null;
    activeModel: string | null;
    activeRuntimeCovered: boolean;
    latestSyncedAt?: string;
    currentPriceCount: number;
    enabledSourcesCount: number;
    syncedSourcesCount: number;
    failedSourcesCount: number;
    pendingSourcesCount: number;
    documentsWithLlm: number;
    pricingAwareDocuments: number;
    totalRequests: number;
    totalTokens: number;
    totalEstimatedCostUsd: number;
    topModels: DocumentMonitoringModelAggregate[];
    sourceStatuses: CostVisibilitySourceStatus[];
};

export function createCostVisibilityOverview({
    summary,
    pricingSources,
    prices,
    runtimeOverview,
}: {
    summary: DocumentMonitoringSummary | null;
    pricingSources: PricingSource[];
    prices: ModelPrice[];
    runtimeOverview: ProviderRuntimeOverview;
}): CostVisibilityOverview {
    const activeProviderType = runtimeOverview.indexing?.provider_type ?? null;
    const activeProviderLabel = activeProviderType ? formatProviderTypeLabel(activeProviderType) : null;
    const activeModel = getProviderRuntimeModel(runtimeOverview.indexing);

    const sourceStatuses = pricingSources
        .map((source) => createSourceStatus(source, activeProviderType))
        .sort(compareSourceStatus);

    const enabledSourcesCount = sourceStatuses.filter((source) => source.health !== "disabled").length;
    const syncedSourcesCount = sourceStatuses.filter((source) => source.health === "synced").length;
    const failedSourcesCount = sourceStatuses.filter((source) => source.health === "failed").length;
    const pendingSourcesCount = sourceStatuses.filter((source) => source.health === "pending").length;
    const currentPriceCount = prices.length;
    const latestSyncedAt = findLatestSyncedAt(sourceStatuses);
    const activeRuntimeCovered = Boolean(
        activeProviderType &&
        sourceStatuses.some((source) => source.providerType === activeProviderType && source.health === "synced") &&
        prices.some((price) => price.provider_type === activeProviderType)
    );

    const topModels = [...(summary?.model_breakdown ?? [])].sort(compareModelsByObservedCost).slice(0, 4);
    const documentsWithLlm = summary?.documents_with_llm ?? 0;
    const pricingAwareDocuments = summary?.pricing_aware_documents ?? 0;
    const totalRequests = summary?.total_llm_requests ?? 0;
    const totalTokens = summary?.total_tokens ?? 0;
    const totalEstimatedCostUsd = summary?.total_estimated_cost_usd ?? 0;

    return {
        health: resolveCostVisibilityHealth({
            activeProviderType,
            activeRuntimeCovered,
            currentPriceCount,
            syncedSourcesCount,
            documentsWithLlm,
            pricingAwareDocuments,
            totalRequests,
        }),
        activeProviderType,
        activeProviderLabel,
        activeModel,
        activeRuntimeCovered,
        latestSyncedAt,
        currentPriceCount,
        enabledSourcesCount,
        syncedSourcesCount,
        failedSourcesCount,
        pendingSourcesCount,
        documentsWithLlm,
        pricingAwareDocuments,
        totalRequests,
        totalTokens,
        totalEstimatedCostUsd,
        topModels,
        sourceStatuses,
    };
}

function createSourceStatus(source: PricingSource, activeProviderType: string | null): CostVisibilitySourceStatus {
    return {
        id: source.id,
        providerType: source.provider_type,
        providerLabel: formatProviderTypeLabel(source.provider_type),
        name: source.name,
        sourceURL: source.source_url,
        health: resolvePricingSourceHealth(source),
        isActiveRuntime: source.provider_type === activeProviderType,
        lastSyncedAt: source.last_synced_at ?? undefined,
        lastPriceCount: source.last_price_count ?? 0,
        lastSyncError: source.last_sync_error ?? undefined,
    };
}

function resolvePricingSourceHealth(source: PricingSource): PricingSourceHealth {
    if (!source.is_enabled) {
        return "disabled";
    }

    const normalizedStatus = source.last_sync_status?.trim().toUpperCase();
    if (normalizedStatus === "SUCCESS" || normalizedStatus === "SYNCED") {
        return "synced";
    }
    if (normalizedStatus === "FAILED") {
        return "failed";
    }

    return "pending";
}

function resolveCostVisibilityHealth({
    activeProviderType,
    activeRuntimeCovered,
    currentPriceCount,
    syncedSourcesCount,
    documentsWithLlm,
    pricingAwareDocuments,
    totalRequests,
}: {
    activeProviderType: string | null;
    activeRuntimeCovered: boolean;
    currentPriceCount: number;
    syncedSourcesCount: number;
    documentsWithLlm: number;
    pricingAwareDocuments: number;
    totalRequests: number;
}) {
    if (
        pricingAwareDocuments > 0 &&
        currentPriceCount > 0 &&
        syncedSourcesCount > 0 &&
        (!activeProviderType || activeRuntimeCovered)
    ) {
        return "trusted";
    }

    if (
        activeProviderType ||
        documentsWithLlm > 0 ||
        pricingAwareDocuments > 0 ||
        totalRequests > 0 ||
        syncedSourcesCount > 0 ||
        currentPriceCount > 0
    ) {
        return "partial";
    }

    return "setup";
}

function compareSourceStatus(left: CostVisibilitySourceStatus, right: CostVisibilitySourceStatus) {
    const activeDiff = Number(right.isActiveRuntime) - Number(left.isActiveRuntime);
    if (activeDiff !== 0) {
        return activeDiff;
    }

    const healthDiff = getSourceHealthWeight(left.health) - getSourceHealthWeight(right.health);
    if (healthDiff !== 0) {
        return healthDiff;
    }

    return left.providerLabel.localeCompare(right.providerLabel);
}

function getSourceHealthWeight(health: PricingSourceHealth) {
    switch (health) {
        case "failed":
            return 0;
        case "pending":
            return 1;
        case "synced":
            return 2;
        default:
            return 3;
    }
}

function compareModelsByObservedCost(left: DocumentMonitoringModelAggregate, right: DocumentMonitoringModelAggregate) {
    if (left.total_cost_usd !== right.total_cost_usd) {
        return right.total_cost_usd - left.total_cost_usd;
    }

    if (left.total_tokens !== right.total_tokens) {
        return right.total_tokens - left.total_tokens;
    }

    return right.request_count - left.request_count;
}

function findLatestSyncedAt(sourceStatuses: CostVisibilitySourceStatus[]) {
    const timestamps = sourceStatuses
        .filter((source) => source.health === "synced" && source.lastSyncedAt)
        .map((source) => new Date(source.lastSyncedAt as string).getTime())
        .filter((value) => Number.isFinite(value));

    if (timestamps.length === 0) {
        return undefined;
    }

    return new Date(Math.max(...timestamps)).toISOString();
}
