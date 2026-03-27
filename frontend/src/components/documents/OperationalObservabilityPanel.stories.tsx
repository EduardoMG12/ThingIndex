import { useState, type ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { expect, userEvent, within } from "storybook/test";

import OperationalObservabilityPanel from "@/components/documents/OperationalObservabilityPanel";
import type { Document, DocumentMonitoringSummary } from "@/lib/api/documents";
import type { DocumentOperationalFilter } from "@/lib/document-observability";

function createDocument(overrides: Partial<Document> = {}): Document {
    return {
        id: 1,
        user_id: 1,
        title: "Operations Manual.pdf",
        file_url: "/documents/1",
        file_path: "documents/operations-manual.pdf",
        size: 2048,
        mime_type: "application/pdf",
        status: "INDEXED",
        collection: "Operations",
        category: {
            id: 3,
            user_id: 1,
            name: "Playbooks",
            description: "Operational playbooks",
            created_at: "2026-03-20T12:00:00Z",
            updated_at: "2026-03-20T12:00:00Z",
        },
        created_at: "2026-03-20T12:00:00Z",
        updated_at: "2026-03-20T12:00:00Z",
        processing_attempts: [],
        ...overrides,
    };
}

const monitoringSummary: DocumentMonitoringSummary = {
    total_documents: 4,
    indexed_documents: 1,
    processing_documents: 1,
    queued_documents: 1,
    failed_documents: 1,
    permanent_failed_documents: 1,
    retrying_documents: 0,
    orphaned_documents: 1,
    dlq_documents: 0,
    documents_with_llm: 2,
    pricing_aware_documents: 2,
    total_llm_requests: 4,
    total_input_tokens: 1800,
    total_output_tokens: 650,
    total_tokens: 2450,
    total_estimated_cost_usd: 0.0134,
    average_duration_ms: 4200,
    average_quality_score: 84.1,
    last_completed_at: "2026-03-22T12:30:00Z",
    model_breakdown: [
        {
            provider_type: "OPENAI",
            model: "gpt-4.1-mini",
            document_count: 2,
            request_count: 4,
            total_tokens: 2450,
            total_cost_usd: 0.0134,
        },
    ],
};

const documents: Document[] = [
    createDocument({
        id: 11,
        title: "Orphaned invoice.pdf",
        status: "ORPHANED",
        updated_at: "2026-03-22T12:10:00Z",
    }),
    createDocument({
        id: 12,
        title: "Stalled parser.pdf",
        status: "PROCESSING",
        processing_stage: "PROCESSING",
        index_started_at: "2026-03-22T10:45:00Z",
        updated_at: "2026-03-22T10:45:00Z",
    }),
    createDocument({
        id: 13,
        title: "Failed import.pdf",
        status: "FAILED",
        processing_stage: "FAILED_PERMANENT",
        last_processing_error: "HTML parser crashed while reading the source.",
        updated_at: "2026-03-22T12:15:00Z",
    }),
    createDocument({
        id: 14,
        title: "Healthy guide.pdf",
        status: "INDEXED",
        collection: "Guides",
        updated_at: "2026-03-22T12:25:00Z",
    }),
];

type OperationalObservabilityPanelStoryProps = ComponentProps<typeof OperationalObservabilityPanel>;

const meta = {
    title: "Documents/OperationalObservabilityPanel",
    component: OperationalObservabilityPanel,
    tags: ["autodocs"],
    parameters: {
        layout: "fullscreen",
    },
    args: {
        documents,
        summary: monitoringSummary,
        activeFilter: "all",
        searchTerm: "",
        onFilterChange: () => undefined,
        onSearchTermChange: () => undefined,
        retryingDocumentId: null,
        onOpenPreview: () => undefined,
        onRetry: () => undefined,
    },
} satisfies Meta<typeof OperationalObservabilityPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

function ControlledOperationalObservabilityPanel(args: OperationalObservabilityPanelStoryProps) {
    const [activeFilter, setActiveFilter] = useState<DocumentOperationalFilter>("all");
    const [searchTerm, setSearchTerm] = useState("");

    return (
        <OperationalObservabilityPanel
            {...args}
            activeFilter={activeFilter}
            searchTerm={searchTerm}
            onFilterChange={setActiveFilter}
            onSearchTermChange={setSearchTerm}
        />
    );
}

export const Default: Story = {
    args: {
        ...meta.args,
    },
    render: (args) => <ControlledOperationalObservabilityPanel {...args} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement);

        await expect(canvas.getByText("Actionable monitoring")).toBeVisible();
        await expect(canvas.getByText("Executive brief")).toBeVisible();
        await expect(canvas.getByText("Anomaly map")).toBeVisible();
        await userEvent.click(canvas.getByRole("button", { name: /Needs attention/i }));
        await expect(canvas.getByRole("button", { name: /Needs attention/i })).toHaveAttribute("aria-pressed", "true");
        await userEvent.type(canvas.getByLabelText("Operational search and slice"), "healthy");
        await expect(canvas.getByDisplayValue("healthy")).toBeVisible();
    },
};

export const NominalFlow: Story = {
    args: {
        ...meta.args,
        documents: [documents[3]],
        summary: {
            ...monitoringSummary,
            total_documents: 1,
            indexed_documents: 1,
            processing_documents: 0,
            queued_documents: 0,
            failed_documents: 0,
            permanent_failed_documents: 0,
            retrying_documents: 0,
            orphaned_documents: 0,
            documents_with_llm: 1,
        },
    },
    render: (args) => <ControlledOperationalObservabilityPanel {...args} />,
};
