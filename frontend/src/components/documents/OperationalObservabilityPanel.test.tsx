import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import OperationalObservabilityPanel from "@/components/documents/OperationalObservabilityPanel";
import { createTestWrapper } from "@/test/test-utils";

function createDocument(overrides: Record<string, unknown> = {}) {
    return {
        id: 1,
        user_id: 7,
        title: "Case.pdf",
        file_url: "/documents/1",
        file_path: "documents/case.pdf",
        size: 1024,
        mime_type: "application/pdf",
        status: "INDEXED",
        created_at: "2026-03-19T00:00:00Z",
        updated_at: "2026-03-19T00:00:00Z",
        ...overrides,
    };
}

describe("OperationalObservabilityPanel", () => {
    it("renders a critical operational summary and the watchlist", () => {
        render(
            <OperationalObservabilityPanel
                documents={[
                    createDocument({
                        id: 1,
                        title: "Orphaned.pdf",
                        status: "ORPHANED",
                        updated_at: "2026-03-20T11:45:00Z",
                    }),
                    createDocument({
                        id: 2,
                        title: "Healthy.pdf",
                        status: "INDEXED",
                        updated_at: "2026-03-20T11:55:00Z",
                    }),
                ]}
                summary={{
                    total_documents: 2,
                    indexed_documents: 1,
                    processing_documents: 0,
                    queued_documents: 0,
                    failed_documents: 0,
                    permanent_failed_documents: 0,
                    retrying_documents: 0,
                    orphaned_documents: 1,
                    dlq_documents: 0,
                    documents_with_llm: 0,
                    pricing_aware_documents: 0,
                    total_llm_requests: 0,
                    total_input_tokens: 0,
                    total_output_tokens: 0,
                    total_tokens: 0,
                    total_estimated_cost_usd: 0,
                    average_duration_ms: 0,
                    average_quality_score: 0,
                    last_completed_at: "2026-03-20T11:30:00Z",
                    model_breakdown: [],
                }}
                activeFilter="all"
                searchTerm=""
                onFilterChange={vi.fn()}
                onSearchTermChange={vi.fn()}
                onOpenPreview={vi.fn()}
                onRetry={vi.fn()}
                retryingDocumentId={null}
            />,
            { wrapper: createTestWrapper("en") }
        );

        expect(screen.getByRole("heading", { name: "Actionable monitoring" })).toBeInTheDocument();
        expect(screen.getByText("Critical attention")).toBeInTheDocument();
        expect(screen.getByText("Orphaned.pdf")).toBeInTheDocument();
        expect(screen.getByText("Original file missing")).toBeInTheDocument();
    });

    it("forwards search and filter interactions", async () => {
        const user = userEvent.setup();
        const onFilterChange = vi.fn();
        const onSearchTermChange = vi.fn();

        render(
            <OperationalObservabilityPanel
                documents={[createDocument()]}
                summary={null}
                activeFilter="all"
                searchTerm=""
                onFilterChange={onFilterChange}
                onSearchTermChange={onSearchTermChange}
                onOpenPreview={vi.fn()}
                onRetry={vi.fn()}
                retryingDocumentId={null}
            />,
            { wrapper: createTestWrapper("en") }
        );

        await user.type(screen.getByLabelText("Operational search and slice"), "ops");
        await user.click(screen.getByRole("button", { name: /Needs attention/i }));

        expect(onSearchTermChange).toHaveBeenCalled();
        expect(onFilterChange).toHaveBeenCalledWith("attention");
    });
});
