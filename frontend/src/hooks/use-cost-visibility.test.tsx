import { renderHook, waitFor, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/documents", () => ({
    fetchDocumentMonitoringSummary: vi.fn(),
}));

vi.mock("@/lib/api/pricing", () => ({
    fetchPricingSources: vi.fn(),
    fetchPrices: vi.fn(),
}));

import { useCostVisibility } from "@/hooks/use-cost-visibility";
import { fetchDocumentMonitoringSummary } from "@/lib/api/documents";
import { fetchPrices, fetchPricingSources } from "@/lib/api/pricing";

describe("useCostVisibility", () => {
    beforeEach(() => {
        vi.mocked(fetchDocumentMonitoringSummary).mockReset();
        vi.mocked(fetchPricingSources).mockReset();
        vi.mocked(fetchPrices).mockReset();
    });

    it("loads monitoring summary, pricing sources, and prices on mount", async () => {
        vi.mocked(fetchDocumentMonitoringSummary).mockResolvedValue({
            total_documents: 1,
            indexed_documents: 1,
            processing_documents: 0,
            queued_documents: 0,
            failed_documents: 0,
            permanent_failed_documents: 0,
            retrying_documents: 0,
            orphaned_documents: 0,
            dlq_documents: 0,
            documents_with_llm: 1,
            pricing_aware_documents: 1,
            total_llm_requests: 1,
            total_input_tokens: 10,
            total_output_tokens: 5,
            total_tokens: 15,
            total_estimated_cost_usd: 0.001,
            average_duration_ms: 0,
            average_quality_score: 0,
            last_completed_at: "2026-03-22T12:00:00Z",
            model_breakdown: [],
        });
        vi.mocked(fetchPricingSources).mockResolvedValue([
            {
                id: 1,
                user_id: 1,
                provider_type: "OPENAI",
                name: "OpenAI Pricing",
                source_url: "https://example.com/openai",
                is_enabled: true,
                last_synced_at: "2026-03-22T11:00:00Z",
                last_sync_status: "SUCCESS",
                last_sync_error: "",
                last_sync_duration_ms: 1000,
                last_price_count: 2,
                created_at: "2026-03-22T10:00:00Z",
                updated_at: "2026-03-22T11:00:00Z",
            },
        ]);
        vi.mocked(fetchPrices).mockResolvedValue([
            {
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
                last_synced_at: "2026-03-22T11:00:00Z",
            },
        ]);

        const { result } = renderHook(() => useCostVisibility());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(fetchDocumentMonitoringSummary).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(fetchPricingSources).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(fetchPrices).toHaveBeenCalledWith(expect.any(AbortSignal));
        expect(result.current.error).toBeNull();
        expect(result.current.summary?.pricing_aware_documents).toBe(1);
        expect(result.current.pricingSources).toHaveLength(1);
        expect(result.current.prices).toHaveLength(1);
    });

    it("preserves partial data when one pricing signal fails and reloads later", async () => {
        vi.mocked(fetchDocumentMonitoringSummary)
            .mockResolvedValueOnce({
                total_documents: 1,
                indexed_documents: 1,
                processing_documents: 0,
                queued_documents: 0,
                failed_documents: 0,
                permanent_failed_documents: 0,
                retrying_documents: 0,
                orphaned_documents: 0,
                dlq_documents: 0,
                documents_with_llm: 1,
                pricing_aware_documents: 0,
                total_llm_requests: 1,
                total_input_tokens: 10,
                total_output_tokens: 0,
                total_tokens: 10,
                total_estimated_cost_usd: 0,
                average_duration_ms: 0,
                average_quality_score: 0,
                last_completed_at: "2026-03-22T12:00:00Z",
                model_breakdown: [],
            })
            .mockResolvedValueOnce({
                total_documents: 1,
                indexed_documents: 1,
                processing_documents: 0,
                queued_documents: 0,
                failed_documents: 0,
                permanent_failed_documents: 0,
                retrying_documents: 0,
                orphaned_documents: 0,
                dlq_documents: 0,
                documents_with_llm: 1,
                pricing_aware_documents: 1,
                total_llm_requests: 2,
                total_input_tokens: 10,
                total_output_tokens: 5,
                total_tokens: 15,
                total_estimated_cost_usd: 0.0012,
                average_duration_ms: 0,
                average_quality_score: 0,
                last_completed_at: "2026-03-22T12:00:00Z",
                model_breakdown: [],
            });
        vi.mocked(fetchPricingSources).mockResolvedValue([]);
        vi.mocked(fetchPrices)
            .mockRejectedValueOnce(new Error("pricing unavailable"))
            .mockResolvedValueOnce([]);

        const { result } = renderHook(() => useCostVisibility());

        await waitFor(() => {
            expect(result.current.isLoading).toBe(false);
        });

        expect(result.current.error).toBe("pricing unavailable");
        expect(result.current.summary?.documents_with_llm).toBe(1);
        expect(result.current.prices).toEqual([]);

        await act(async () => {
            await result.current.reload();
        });

        await waitFor(() => {
            expect(result.current.error).toBeNull();
        });

        expect(fetchDocumentMonitoringSummary).toHaveBeenCalledTimes(2);
        expect(fetchPrices).toHaveBeenCalledTimes(2);
        expect(result.current.summary?.pricing_aware_documents).toBe(1);
    });
});
