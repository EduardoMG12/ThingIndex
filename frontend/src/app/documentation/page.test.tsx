import type { PropsWithChildren } from "react";

import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const {
    fetchRuntimeProfileMock,
    fetchIndexingBenchmarkMock,
    fetchProviderSupportMatrixMock,
} = vi.hoisted(() => ({
    fetchRuntimeProfileMock: vi.fn(),
    fetchIndexingBenchmarkMock: vi.fn(),
    fetchProviderSupportMatrixMock: vi.fn(),
}));

vi.mock("@/components/layout/AppShell", () => ({
    __esModule: true,
    default: ({ children }: PropsWithChildren) => <div data-testid="app-shell">{children}</div>,
}));

vi.mock("@/lib/api/runtime-profile", () => ({
    fetchRuntimeProfile: fetchRuntimeProfileMock,
}));

vi.mock("@/lib/api/indexing-benchmark", () => ({
    fetchIndexingBenchmark: fetchIndexingBenchmarkMock,
}));

vi.mock("@/lib/api/providers", async () => {
    const actual = await vi.importActual<typeof import("@/lib/api/providers")>("@/lib/api/providers");

    return {
        ...actual,
        fetchProviderSupportMatrix: fetchProviderSupportMatrixMock,
    };
});

import DocumentationPage from "@/app/documentation/page";

describe("DocumentationPage", () => {
    it("renders the streamlined documentation page with the current roadmap state", async () => {
        fetchRuntimeProfileMock.mockResolvedValue({
            product_name: "ThingIndex",
            tagline: "Relational document intelligence.",
            primary_use_case: "Organize operational document collections.",
            core_problem: "Disconnected tools break provenance.",
            value_proposition: "Relational-first context recovery.",
            target_users: [{ name: "Operations teams", description: "Need provenance." }],
            current_scope: ["Document workspace"],
            differentiators: ["Relational-first"],
            non_goals: ["No generic copilot"],
            research_tracks: ["Vector benchmark"],
            artifact_refs: [
                {
                    label: "Product focus RFC",
                    path: "rfc/0001-product-focus.md",
                    kind: "RFC",
                    summary: "Backlog alignment.",
                },
            ],
            last_reviewed_on: "2026-03-23",
        });
        fetchIndexingBenchmarkMock.mockResolvedValue({
            dataset_version: "2026-03-relational-baseline-v1",
            corpus_source_path: "internal/usecase/testdata/indexing-benchmark/corpus.json",
            last_reviewed_on: "2026-03-23",
            summary: "Controlled benchmark.",
            methodology: ["Run the deterministic indexer."],
            pass_criteria: ["Top-1 exact match."],
            limitations: ["Small corpus."],
            heuristic_quality_note: "Heuristic is operational only.",
            corpus: [
                {
                    id: "msa",
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
                corpus_documents: 1,
                document_queries: 4,
                collection_queries: 3,
                structure_pass_rate: 100,
                tree_node_top1_accuracy: 75,
                tree_node_recall_at_3: 100,
                tree_page_top1_accuracy: 75,
                tree_page_recall_at_3: 100,
                collection_document_top1_accuracy: 100,
                collection_document_recall_at_3: 100,
                exact_match_accuracy: 85.71428571428571,
                average_heuristic_score: 98,
                minimum_heuristic_score: 98,
                maximum_heuristic_score: 98,
            },
            cases: [
                {
                    id: "service-payment",
                    scope: "DOCUMENT_TREE",
                    query: "payment deadline",
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
        fetchProviderSupportMatrixMock
            .mockResolvedValueOnce([
                {
                    family: "STORAGE",
                    provider_type: "MINIO",
                    display_name: "MinIO",
                    support_level: "READY",
                    transport: "S3_COMPATIBLE",
                    summary: "Ready for the main runtime.",
                    suggested_setup: {},
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: false,
                        supports_streaming: false,
                        supports_indexing: true,
                        supports_chat: false,
                        supports_optional_live_tests: true,
                        requires_explicit_model: false,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Mainline storage path."],
                },
            ])
            .mockResolvedValueOnce([
                {
                    family: "LLM",
                    provider_type: "OPENAI",
                    display_name: "OpenAI",
                    support_level: "READY",
                    transport: "NATIVE_OPENAI",
                    summary: "Ready for chat and indexing.",
                    suggested_setup: {},
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: true,
                        supports_streaming: true,
                        supports_indexing: true,
                        supports_chat: true,
                        supports_optional_live_tests: true,
                        requires_explicit_model: false,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Primary path."],
                },
            ]);

        render(<DocumentationPage />);

        await waitFor(() => {
            expect(screen.getByRole("heading", { level: 2, name: "Foco do Produto" })).toBeInTheDocument();
        });

        expect(screen.getByTestId("app-shell")).toBeInTheDocument();
        expect(screen.getByText("Catálogo documental relacional")).toBeInTheDocument();
        expect(screen.getByText("Thin HTTP boundary")).toBeInTheDocument();
        expect(screen.getByText("Nenhuma task aberta no backlog principal neste momento.")).toBeInTheDocument();
        expect(
            screen.getByText("Novos polimentos visuais entram como manutenção incremental, não como uma fase estrutural do produto.")
        ).toBeInTheDocument();
        expect(screen.getByText("docs/proposta-continuacao-tcc.md")).toBeInTheDocument();
        expect(screen.getByText("docs/README.md")).toBeInTheDocument();
    });
});
