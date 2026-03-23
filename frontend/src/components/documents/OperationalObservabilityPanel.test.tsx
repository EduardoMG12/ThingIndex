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
        processing_attempts: [],
        ...overrides,
    };
}

describe("OperationalObservabilityPanel", () => {
    it("renders the executive overview, anomaly map, and watchlist hints", () => {
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
        expect(screen.getByText("Executive brief")).toBeInTheDocument();
        expect(screen.getByText("Next operational actions")).toBeInTheDocument();
        expect(screen.getByText("Anomaly map")).toBeInTheDocument();
        expect(screen.getByText("Missing originals")).toBeInTheDocument();
        expect(screen.getByText("Open priority document")).toBeInTheDocument();
        expect(
            screen.getAllByText("Reupload the file to restore the original, or delete the record if it should no longer exist.")
        ).toHaveLength(2);
    });

    it("lets the operator focus a slice, clear it, and open the priority document", async () => {
        const user = userEvent.setup();
        const onFilterChange = vi.fn();
        const onSearchTermChange = vi.fn();
        const onOpenPreview = vi.fn();

        const orphanedDocument = createDocument({
            id: 10,
            title: "Orphaned.pdf",
            status: "ORPHANED",
            updated_at: "2026-03-20T11:45:00Z",
        });

        render(
            <OperationalObservabilityPanel
                documents={[orphanedDocument, createDocument({ id: 11, title: "Healthy.pdf" })]}
                summary={null}
                activeFilter="manual"
                searchTerm="ops"
                onFilterChange={onFilterChange}
                onSearchTermChange={onSearchTermChange}
                onOpenPreview={onOpenPreview}
                onRetry={vi.fn()}
                retryingDocumentId={null}
            />,
            { wrapper: createTestWrapper("en") }
        );

        await user.click(screen.getAllByRole("button", { name: "Focus: Resolve manual blockers" })[0]);
        expect(onFilterChange).toHaveBeenCalledWith("manual");
        expect(onSearchTermChange).toHaveBeenCalledWith("");

        await user.click(screen.getAllByRole("button", { name: "Clear operational slice" })[0]);
        expect(onFilterChange).toHaveBeenCalledWith("all");

        await user.click(screen.getByRole("button", { name: "Open priority document" }));
        expect(onOpenPreview).toHaveBeenCalledWith(orphanedDocument);
    });

    it("forwards manual search and filter interactions", async () => {
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
