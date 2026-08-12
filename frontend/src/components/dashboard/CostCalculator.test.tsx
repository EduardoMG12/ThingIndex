import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/use-cost-visibility", () => ({
    useCostVisibility: vi.fn(),
}));

vi.mock("@/hooks/use-provider-runtime-overview", () => ({
    useProviderRuntimeOverview: vi.fn(),
}));

import CostCalculator from "@/components/dashboard/CostCalculator";
import { useCostVisibility } from "@/hooks/use-cost-visibility";
import { useProviderRuntimeOverview } from "@/hooks/use-provider-runtime-overview";
import { createTestWrapper } from "@/test/test-utils";

function createCostVisibilityReturn() {
    return {
        summary: {
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
            pricing_aware_documents: 3,
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
                    document_count: 3,
                    request_count: 8,
                    total_tokens: 1800,
                    total_cost_usd: 0.0123,
                },
            ],
        },
        pricingSources: [
                {
                    id: 1,
                    user_id: 1,
                    provider_type: "OPENAI" as const,
                name: "OpenAI Pricing",
                source_url: "https://example.com/openai",
                is_enabled: true,
                last_synced_at: "2026-03-22T11:30:00Z",
                last_sync_status: "SUCCESS",
                last_sync_error: "",
                last_sync_duration_ms: 1000,
                last_price_count: 3,
                created_at: "2026-03-22T10:00:00Z",
                updated_at: "2026-03-22T11:30:00Z",
            },
        ],
        prices: [
                {
                    id: 11,
                    pricing_source_id: 1,
                    provider_type: "OPENAI" as const,
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
            },
        ],
        isLoading: false,
        error: null,
        reload: vi.fn(),
    };
}

describe("CostCalculator", () => {
    beforeEach(() => {
        vi.mocked(useCostVisibility).mockReturnValue(createCostVisibilityReturn());

        vi.mocked(useProviderRuntimeOverview).mockReturnValue({
            overview: {
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
            },
            isLoading: false,
            error: null,
        });
    });

    it("renders cost visibility from runtime data without fake fixed placeholders", () => {
        render(<CostCalculator />, {
            wrapper: createTestWrapper("en"),
        });

        expect(screen.getByText("Operational cost estimate")).toBeInTheDocument();
        expect(screen.getByText("Trusted estimate")).toBeInTheDocument();
        expect(screen.getByText("Observed indexing cost")).toBeInTheDocument();
        expect(screen.getByText("Estimate trust boundary")).toBeInTheDocument();
        expect(screen.getByText("Pricing source health")).toBeInTheDocument();
        expect(screen.getByText("Observed cost drivers")).toBeInTheDocument();
        expect(screen.getAllByText("gpt-4.1-mini", { exact: true })).toHaveLength(2);
        expect(screen.queryByText(/Vertex AI/i)).not.toBeInTheDocument();
        expect(screen.queryByText(/1\.2k \/ 5k tokens/i)).not.toBeInTheDocument();
        expect(screen.queryByText("No fixed amount shown")).not.toBeInTheDocument();
    });

    it("surfaces partial-data warnings without collapsing the panel", () => {
        vi.mocked(useCostVisibility).mockReturnValue({
            ...createCostVisibilityReturn(),
            summary: null,
            pricingSources: [],
            prices: [],
            error: "pricing unavailable",
            isLoading: false,
            reload: vi.fn(),
        });

        vi.mocked(useProviderRuntimeOverview).mockReturnValue({
            overview: {
                storage: null,
                indexing: null,
                chat: null,
            },
            isLoading: false,
            error: null,
        });

        render(<CostCalculator />, {
            wrapper: createTestWrapper("en"),
        });

        expect(screen.getByText("Pricing setup needed")).toBeInTheDocument();
        expect(screen.getByText("Some pricing signals did not load")).toBeInTheDocument();
        expect(screen.getByText("pricing unavailable")).toBeInTheDocument();
        expect(screen.getByText("No pricing source is registered yet. Save at least one official URL in settings before trusting cost estimates.")).toBeInTheDocument();
    });
});
