import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchIndexingBenchmark } from "@/lib/api/indexing-benchmark";

describe("indexing benchmark api", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
    });

    it("loads and normalizes the public indexing benchmark", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        dataset_version: "2026-03-relational-baseline-v1",
                        corpus_source_path: "internal/usecase/testdata/indexing-benchmark/corpus.json",
                        last_reviewed_on: "2026-03-22",
                        summary: "Controlled benchmark.",
                        methodology: ["Run the deterministic indexer.", "", null],
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
                                expected_outline_titles: ["2 Payment Terms", "", null],
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
                        cases: [
                            {
                                id: "service-payment-deadline",
                                scope: "DOCUMENT_TREE",
                                query: "payment deadline invoice receipt",
                                document_title: "Master Service Agreement",
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
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const report = await fetchIndexingBenchmark();

        expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe("/api/runtime/indexing-benchmark");
        expect(report).toEqual({
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
    });
});
