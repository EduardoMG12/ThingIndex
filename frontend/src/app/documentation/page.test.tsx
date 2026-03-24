import type { PropsWithChildren } from "react";

import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createTestWrapper } from "@/test/test-utils";

const {
    pathnameState,
    fetchRuntimeProfileMock,
    fetchIndexingBenchmarkMock,
    fetchProviderSupportMatrixMock,
} = vi.hoisted(() => ({
    pathnameState: { value: "/en" },
    fetchRuntimeProfileMock: vi.fn(),
    fetchIndexingBenchmarkMock: vi.fn(),
    fetchProviderSupportMatrixMock: vi.fn(),
}));

vi.mock("next/navigation", () => ({
    usePathname: () => pathnameState.value,
    useRouter: () => ({
        push: vi.fn(),
        replace: vi.fn(),
        refresh: vi.fn(),
        prefetch: vi.fn(),
        back: vi.fn(),
    }),
    redirect: (href: string) => {
        throw new Error(`NEXT_REDIRECT:${href}`);
    },
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

function mockDocumentationPayloads() {
    fetchRuntimeProfileMock.mockResolvedValue({
        product_name: "ThingIndex",
        tagline: "Relational document intelligence for auditable operational knowledge.",
        primary_use_case: "Organize collections of operational, compliance, legal, and research documents with typed relational indexing, recoverable context, and grounded follow-up workflows.",
        core_problem: "Document-heavy teams often have storage, metadata, and AI answers split across disconnected tools, which makes context recovery, provenance, and operational auditing difficult.",
        value_proposition: "ThingIndex keeps document organization, structured extraction, context recovery, and provider portability in a single relational runtime that can be self-hosted or consumed as an API without hiding provenance behind a vector-first stack.",
        target_users: [{ name: "Operations and compliance teams", description: "Need searchable workspaces, explicit provenance, and repeatable document handling across collections." }],
        current_scope: ["JWT-authenticated document workspace with categories, collections, and uploads"],
        differentiators: ["Relational-first context reconstruction instead of vector-first navigation"],
        non_goals: ["Do not present the main runtime as a generic AI copilot detached from document provenance"],
        research_tracks: ["Expand the controlled indexing benchmark to compare the relational baseline against vector retrieval"],
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
        summary: "Controlled exact-match benchmark for the current relational indexing baseline. The corpus is embedded, reproducible, and evaluated with the real deterministic indexer used by the main runtime.",
        methodology: ["Run the real deterministic indexer over the embedded benchmark corpus."],
        pass_criteria: ["Document-tree cases pass only when the expected node and page are both ranked first."],
        limitations: ["This corpus is intentionally small and controlled; it validates the baseline, not arbitrary production corpora."],
        heuristic_quality_note: "The quality score exposed in document monitoring remains a structural heuristic for operators. It is not ground truth and should be interpreted together with this benchmark instead of replacing it.",
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
                id: "service-payment-deadline",
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
}

describe("DocumentationPage", () => {
    beforeEach(() => {
        pathnameState.value = "/en";
        fetchRuntimeProfileMock.mockReset();
        fetchIndexingBenchmarkMock.mockReset();
        fetchProviderSupportMatrixMock.mockReset();
    });

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

        render(<DocumentationPage />, { wrapper: createTestWrapper() });

        await waitFor(() => {
            expect(screen.getByRole("heading", { level: 2, name: "Product focus" })).toBeInTheDocument();
        });

        expect(screen.getByTestId("app-shell")).toBeInTheDocument();
        expect(screen.getByText("Relational document platform with consolidated context")).toBeInTheDocument();
        expect(screen.getByText("Thin HTTP boundary")).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "Engineering quality" })).toBeInTheDocument();
        expect(screen.getByText("expand concrete provider validation in deployments beyond the current mainline path;")).toBeInTheDocument();
        expect(screen.getByText("docs/proposta-continuacao-tcc.md")).toBeInTheDocument();
        expect(screen.getByText("docs/README.md")).toBeInTheDocument();
    });

    it("localizes dynamic documentation resources when pt-BR is selected", async () => {
        pathnameState.value = "/pt-BR";
        mockDocumentationPayloads();

        render(<DocumentationPage />, { wrapper: createTestWrapper("pt-BR") });

        await waitFor(() => {
            expect(screen.getByRole("heading", { level: 2, name: "Foco do produto" })).toBeInTheDocument();
        });

        expect(screen.getByText("Inteligência documental relacional para conhecimento operacional auditável.")).toBeInTheDocument();
        expect(screen.getByText(/Equipes de operações e compliance:/i)).toBeInTheDocument();
        expect(screen.getByText("RFC de foco do produto")).toBeInTheDocument();
        expect(screen.getAllByText("Contrato Mestre de Serviços").length).toBeGreaterThan(0);
        expect(screen.getByText("prazo de pagamento recebimento da fatura")).toBeInTheDocument();
        expect(screen.getByText("Árvore documental")).toBeInTheDocument();
        expect(screen.getByText("Pronto para o runtime principal com configuração padrão direta da OpenAI, teste de conexão, descoberta de modelos e chat/indexação grounded.")).toBeInTheDocument();
    });
});
