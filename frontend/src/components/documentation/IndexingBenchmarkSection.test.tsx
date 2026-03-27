import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createTestWrapper } from "@/test/test-utils";

const { fetchIndexingBenchmarkMock } = vi.hoisted(() => ({
    fetchIndexingBenchmarkMock: vi.fn(),
}));

vi.mock("@/lib/api/indexing-benchmark", () => ({
    fetchIndexingBenchmark: fetchIndexingBenchmarkMock,
}));

import { IndexingBenchmarkSection } from "@/components/documentation/IndexingBenchmarkSection";

describe("IndexingBenchmarkSection", () => {
    it("renders the public indexing benchmark payload", async () => {
        fetchIndexingBenchmarkMock.mockResolvedValue({
            dataset_version: "2026-03-relational-baseline-v1",
            corpus_source_path: "internal/usecase/testdata/indexing-benchmark/corpus.json",
            last_reviewed_on: "2026-03-22",
            summary: "Controlled benchmark.",
            methodology: ["Run the deterministic indexer."],
            pass_criteria: ["Top-1 exact match."],
            limitations: ["Small corpus."],
            heuristic_quality_note: "Heuristic is operational only.",
            corpus: [
                {
                    id: "service-agreement",
                    title: "Master Service Agreement",
                    mime_type: "text/plain",
                    focus: "Payment deadlines.",
                    query_count: 1,
                    expected_outline_titles: ["2 Payment Terms"],
                    expected_table_count: 0,
                    outline_node_count: 4,
                    table_count: 0,
                    structure_passed: true,
                    retrieval_ready: true,
                    heuristic_quality_score: 98,
                },
            ],
            metrics: {
                corpus_documents: 3,
                document_queries: 4,
                collection_queries: 3,
                structure_pass_rate: 33.33333333333333,
                tree_node_top1_accuracy: 0,
                tree_node_recall_at_3: 25,
                tree_page_top1_accuracy: 75,
                tree_page_recall_at_3: 75,
                collection_document_top1_accuracy: 66.66666666666666,
                collection_document_recall_at_3: 66.66666666666666,
                exact_match_accuracy: 28.57142857142857,
                average_heuristic_score: 98,
                minimum_heuristic_score: 98,
                maximum_heuristic_score: 98,
            },
            current_snapshot: {
                id: "2026-03-26-relational-baseline-v1",
                label: "Pre-quality backlog baseline",
                recorded_on: "2026-03-26",
                summary: "Before the indexing quality backlog.",
                dataset_version: "2026-03-relational-baseline-v1",
                pipeline_version: "page-tree-v3",
                prompt_profile_version: "llm-index-v5",
                chunk_strategy_version: "section-window-v2",
                metrics: {
                    corpus_documents: 3,
                    document_queries: 4,
                    collection_queries: 3,
                    structure_pass_rate: 33.33333333333333,
                    tree_node_top1_accuracy: 0,
                    tree_node_recall_at_3: 25,
                    tree_page_top1_accuracy: 75,
                    tree_page_recall_at_3: 75,
                    collection_document_top1_accuracy: 66.66666666666666,
                    collection_document_recall_at_3: 66.66666666666666,
                    exact_match_accuracy: 28.57142857142857,
                    average_heuristic_score: 98,
                    minimum_heuristic_score: 98,
                    maximum_heuristic_score: 98,
                },
            },
            latest_comparison: {
                id: "2026-03-20-relational-baseline-v0__to__2026-03-26-relational-baseline-v1",
                title: "Before -> After",
                recorded_on: "2026-03-26",
                summary: "Recorded pipeline delta.",
                before: {
                    id: "2026-03-20-relational-baseline-v0",
                    label: "Before",
                    recorded_on: "2026-03-20",
                    summary: "",
                    dataset_version: "2026-03-relational-baseline-v0",
                    pipeline_version: "",
                    prompt_profile_version: "",
                    chunk_strategy_version: "",
                    metrics: {
                        corpus_documents: 1,
                        document_queries: 1,
                        collection_queries: 1,
                        structure_pass_rate: 0,
                        tree_node_top1_accuracy: 0,
                        tree_node_recall_at_3: 0,
                        tree_page_top1_accuracy: 0,
                        tree_page_recall_at_3: 0,
                        collection_document_top1_accuracy: 0,
                        collection_document_recall_at_3: 0,
                        exact_match_accuracy: 0,
                        average_heuristic_score: 98,
                        minimum_heuristic_score: 98,
                        maximum_heuristic_score: 98,
                    },
                },
                after: {
                    id: "2026-03-26-relational-baseline-v1",
                    label: "After",
                    recorded_on: "2026-03-26",
                    summary: "",
                    dataset_version: "2026-03-relational-baseline-v1",
                    pipeline_version: "page-tree-v3",
                    prompt_profile_version: "llm-index-v5",
                    chunk_strategy_version: "section-window-v2",
                    metrics: {
                        corpus_documents: 3,
                        document_queries: 4,
                        collection_queries: 3,
                        structure_pass_rate: 33.33333333333333,
                        tree_node_top1_accuracy: 0,
                        tree_node_recall_at_3: 25,
                        tree_page_top1_accuracy: 75,
                        tree_page_recall_at_3: 75,
                        collection_document_top1_accuracy: 66.66666666666666,
                        collection_document_recall_at_3: 66.66666666666666,
                        exact_match_accuracy: 28.57142857142857,
                        average_heuristic_score: 98,
                        minimum_heuristic_score: 98,
                        maximum_heuristic_score: 98,
                    },
                },
                key_metric_deltas: [
                    {
                        metric: "exact_match_accuracy",
                        previous_value: 0,
                        current_value: 28.57142857142857,
                        delta: 28.57142857142857,
                    },
                ],
            },
            comparison_history: [],
            cases: [
                {
                    id: "service-payment-deadline",
                    scope: "DOCUMENT_TREE",
                    query: "payment deadline invoice receipt",
                    document_title: "Master Service Agreement",
                    expected_document_title: "",
                    actual_document_title: "",
                    document_matched: false,
                    document_in_top_3: false,
                    expected_node_title: "2 Payment Terms",
                    actual_node_title: "2 Payment Terms",
                    node_matched: true,
                    node_in_top_3: true,
                    expected_page_number: 1,
                    actual_page_number: 1,
                    page_matched: true,
                    page_in_top_3: true,
                    passed: true,
                },
            ],
        });

        render(<IndexingBenchmarkSection />, {
            wrapper: createTestWrapper(),
        });

        await waitFor(() => {
            expect(screen.getByText("7 controlled cases")).toBeInTheDocument();
        });

        expect(screen.getByText("/api/v1/meta/indexing-benchmark")).toBeInTheDocument();
        expect(screen.getByText("internal/usecase/testdata/indexing-benchmark/corpus.json")).toBeInTheDocument();
        expect(screen.getAllByText("Master Service Agreement")).toHaveLength(2);
        expect(screen.getByText("payment deadline invoice receipt")).toBeInTheDocument();
        expect(screen.getAllByText("28.6%")).toHaveLength(2);
        expect(screen.getByText("Assessed cases")).toBeInTheDocument();
        expect(screen.getByText("Latest recorded before/after")).toBeInTheDocument();
        expect(screen.getAllByText(/Current pipeline signature/)).toHaveLength(3);
        expect(screen.getByText("+28.6 pts")).toBeInTheDocument();
        expect(screen.getByText("Heuristic is operational only.")).toBeInTheDocument();
    });
});
