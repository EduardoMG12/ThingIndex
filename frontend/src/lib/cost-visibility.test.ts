import { describe, expect, it } from "vitest";

import type { DocumentMonitoringSummary } from "@/lib/api/documents";
import type { ModelPrice, PricingSource } from "@/lib/api/pricing";
import type { ProviderRuntimeOverview } from "@/lib/api/providers";
import { createCostVisibilityOverview } from "@/lib/cost-visibility";

function createSummary(overrides: Partial<DocumentMonitoringSummary> = {}): DocumentMonitoringSummary {
    return {
        total_documents: 4,
        indexed_documents: 4,
        processing_documents: 0,
        queued_documents: 0,
        failed_documents: 0,
        permanent_failed_documents: 0,
        retrying_documents: 0,
        orphaned_documents: 0,
        dlq_documents: 0,
        documents_with_llm: 3,
        pricing_aware_documents: 2,
        total_llm_requests: 8,
        total_input_tokens: 1400,
        total_output_tokens: 400,
        total_tokens: 1800,
        total_estimated_cost_usd: 0.0123,
        average_duration_ms: 0,
        average_quality_score: 0,
        last_completed_at: "2026-03-22T12:00:00Z",
        model_breakdown: [
            {
                provider_type: "OPENAI",
                model: "gpt-4.1-mini",
                document_count: 2,
                request_count: 6,
                total_tokens: 1500,
                total_cost_usd: 0.0102,
            },
            {
                provider_type: "OPENAI",
                model: "gpt-4o-mini",
                document_count: 1,
                request_count: 2,
                total_tokens: 300,
                total_cost_usd: 0.0021,
            },
        ],
        ...overrides,
    };
}

function createSource(overrides: Partial<PricingSource> = {}): PricingSource {
    return {
        id: 1,
        user_id: 1,
        provider_type: "OPENAI",
        name: "OpenAI Pricing",
        source_url: "https://example.com/openai",
        is_enabled: true,
        last_synced_at: "2026-03-22T11:30:00Z",
        last_sync_status: "SUCCESS",
        last_sync_error: "",
        last_sync_duration_ms: 1400,
        last_price_count: 4,
        created_at: "2026-03-22T10:00:00Z",
        updated_at: "2026-03-22T11:30:00Z",
        ...overrides,
    };
}

function createPrice(overrides: Partial<ModelPrice> = {}): ModelPrice {
    return {
        id: 11,
        pricing_source_id: 1,
        provider_type: "OPENAI",
        model_key: "gpt-4.1-mini",
        model_name: "gpt-4.1-mini",
        unit: "per 1M tokens",
        currency: "USD",
        flat_price_usd: 0,
        input_cost_per_million_usd: 0.2,
        output_cost_per_million_usd: 0.8,
        cached_input_cost_per_million_usd: 0.1,
        context_cost_per_million_usd: 0,
        price_notes: "",
        last_synced_at: "2026-03-22T11:30:00Z",
        ...overrides,
    };
}

const runtimeOverview: ProviderRuntimeOverview = {
    storage: null,
    indexing: {
        family: "LLM",
        activation_target: "INDEXING",
        provider_type: "OPENAI",
        public_metadata: {
            model: "gpt-4.1-mini",
        },
        updated_at: "2026-03-22T12:00:00Z",
    },
    chat: null,
};

describe("cost-visibility", () => {
    it("marks the view as trusted when synced prices and priced usage exist", () => {
        const overview = createCostVisibilityOverview({
            summary: createSummary(),
            pricingSources: [createSource()],
            prices: [createPrice()],
            runtimeOverview,
        });

        expect(overview.health).toBe("trusted");
        expect(overview.activeRuntimeCovered).toBe(true);
        expect(overview.syncedSourcesCount).toBe(1);
        expect(overview.documentsWithLlm).toBe(3);
        expect(overview.pricingAwareDocuments).toBe(2);
        expect(overview.topModels[0]?.model).toBe("gpt-4.1-mini");
        expect(overview.sourceStatuses[0]).toMatchObject({
            providerType: "OPENAI",
            health: "synced",
            isActiveRuntime: true,
        });
    });

    it("falls back to partial coverage when runtime exists without synced pricing", () => {
        const overview = createCostVisibilityOverview({
            summary: createSummary({
                pricing_aware_documents: 0,
                total_estimated_cost_usd: 0,
            }),
            pricingSources: [
                createSource({
                    last_sync_status: "FAILED",
                    last_sync_error: "pricing sync failed",
                    last_synced_at: undefined,
                    last_price_count: 0,
                }),
            ],
            prices: [],
            runtimeOverview,
        });

        expect(overview.health).toBe("partial");
        expect(overview.activeRuntimeCovered).toBe(false);
        expect(overview.failedSourcesCount).toBe(1);
        expect(overview.currentPriceCount).toBe(0);
        expect(overview.sourceStatuses[0]?.health).toBe("failed");
    });
});
