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
            structural_signals: [
                {
                    id: "expected_query_node_match",
                    direction: "higher_is_better",
                    status: "pass",
                    rate: 100,
                    count: 4,
                    total: 4,
                },
                {
                    id: "useful_depth",
                    direction: "higher_is_better",
                    status: "watch",
                    rate: 75,
                    count: 3,
                    total: 4,
                },
                {
                    id: "outline_coverage",
                    direction: "higher_is_better",
                    status: "pass",
                    rate: 100,
                    count: 10,
                    total: 10,
                },
                {
                    id: "heading_false_positive",
                    direction: "lower_is_better",
                    status: "watch",
                    rate: 5,
                    count: 1,
                    total: 20,
                },
            ],
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
            reference_comparison: {
                source_path: "pageindex/results",
                last_reviewed_on: "2026-03-27",
                summary: "ThingIndex now matches the workspace references on coherence, coverage, and legibility, but still trails them on structural depth.",
                artifacts: [
                    {
                        id: "earthmover",
                        label: "Research paper reference",
                        source_path: "pageindex/results/earthmover_structure.json",
                        document_name: "earthmover.pdf",
                        total_nodes: 21,
                        max_depth: 3,
                        summary_coverage_ratio: 1,
                        invalid_range_count: 1,
                        average_title_length: 23.52,
                        average_summary_length: 1414.67,
                        notes: ["Carries the deepest hierarchy in the reference set."],
                    },
                ],
                checklist: [
                    {
                        id: "depth",
                        dimension: "depth",
                        label: "Useful structural depth",
                        status: "fail",
                        current_value: "max_depth=1; nested_nodes=0.0%",
                        reference_value: "max_depth=3; nested_nodes=75.4%",
                        summary: "The public ThingIndex corpus remains structurally flatter than the workspace references.",
                    },
                ],
            },
            retrieval_comparison: {
                summary: "The relational baseline remains ahead of the strongest experimental control (Flat hybrid control), especially on exact node and page grounding.",
                baseline_id: "relational-tree-baseline",
                notes: [
                    "This repository does not ship a native vector index in the current runtime.",
                    "The variants below are explicit experimental controls over the same benchmark corpus, not hidden production paths.",
                ],
                variants: [
                    {
                        id: "relational-tree-baseline",
                        label: "Current relational baseline",
                        category: "baseline",
                        status: "pass",
                        summary: "Current production path combining outline-aware retrieval, page grounding, and explicit relational provenance.",
                        metrics: {
                            document_queries: 4,
                            collection_queries: 3,
                            tree_node_top1_accuracy: 100,
                            tree_page_top1_accuracy: 100,
                            collection_document_top1_accuracy: 100,
                            exact_match_accuracy: 100,
                        },
                        key_metric_deltas: [
                            {
                                metric: "exact_match_accuracy",
                                previous_value: 100,
                                current_value: 100,
                                delta: 0,
                            },
                        ],
                    },
                    {
                        id: "flat-hybrid-control",
                        label: "Flat hybrid control",
                        category: "hybrid_control",
                        status: "watch",
                        summary: "Experimental control that ranks nodes and pages independently without iterative tree traversal or path reinforcement.",
                        metrics: {
                            document_queries: 4,
                            collection_queries: 3,
                            tree_node_top1_accuracy: 100,
                            tree_page_top1_accuracy: 75,
                            collection_document_top1_accuracy: 100,
                            exact_match_accuracy: 90.9,
                        },
                        key_metric_deltas: [
                            {
                                metric: "exact_match_accuracy",
                                previous_value: 100,
                                current_value: 90.9,
                                delta: -9.1,
                            },
                        ],
                    },
                ],
            },
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
        expect(screen.getByText("Structural signals")).toBeInTheDocument();
        expect(screen.getByText("Expected node match")).toBeInTheDocument();
        expect(screen.getByText("Heading false positives")).toBeInTheDocument();
        expect(screen.getByText("1/20 indexed headings were unexpected")).toBeInTheDocument();
        expect(screen.getByText("Retrieval comparison")).toBeInTheDocument();
        expect(screen.getByText("Current relational baseline")).toBeInTheDocument();
        expect(screen.getByText("Flat hybrid control")).toBeInTheDocument();
        expect(screen.getByText("This repository does not ship a native vector index in the current runtime.")).toBeInTheDocument();
        expect(screen.getByText("Latest recorded before/after")).toBeInTheDocument();
        expect(screen.getByText("Workspace reference comparison")).toBeInTheDocument();
        expect(screen.getByText("Objective checklist")).toBeInTheDocument();
        expect(screen.getByText("Useful structural depth")).toBeInTheDocument();
        expect(screen.getByText("pageindex/results/earthmover_structure.json")).toBeInTheDocument();
        expect(screen.getAllByText(/Current pipeline signature/)).toHaveLength(3);
        expect(screen.getByText("+28.6 pts")).toBeInTheDocument();
        expect(screen.getByText("Heuristic is operational only.")).toBeInTheDocument();
    });
});
