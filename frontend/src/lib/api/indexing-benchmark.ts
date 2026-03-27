import { apiErrorMessage } from "@/lib/api/error";
import { hasResponseData, type ApiErrorResponse, type ApiSuccessResponse } from "@/lib/api/types";

type IndexingBenchmarkCorpusDocumentApiRecord = {
    id?: string;
    title?: string;
    mime_type?: string;
    focus?: string;
    query_count?: number;
    expected_outline_titles?: unknown;
    expected_table_count?: number;
    outline_node_count?: number;
    table_count?: number;
    structure_passed?: boolean;
    retrieval_ready?: boolean;
    heuristic_quality_score?: number;
};

type IndexingBenchmarkMetricsApiRecord = {
    corpus_documents?: number;
    document_queries?: number;
    collection_queries?: number;
    structure_pass_rate?: number;
    tree_node_top1_accuracy?: number;
    tree_node_recall_at_3?: number;
    tree_page_top1_accuracy?: number;
    tree_page_recall_at_3?: number;
    collection_document_top1_accuracy?: number;
    collection_document_recall_at_3?: number;
    exact_match_accuracy?: number;
    average_heuristic_score?: number;
    minimum_heuristic_score?: number;
    maximum_heuristic_score?: number;
};

type IndexingBenchmarkCaseApiRecord = {
    id?: string;
    scope?: string;
    query?: string;
    document_title?: string;
    expected_document_title?: string;
    actual_document_title?: string;
    document_matched?: boolean;
    document_in_top_3?: boolean;
    expected_node_title?: string;
    actual_node_title?: string;
    node_matched?: boolean;
    node_in_top_3?: boolean;
    expected_page_number?: number | null;
    actual_page_number?: number | null;
    page_matched?: boolean;
    page_in_top_3?: boolean;
    passed?: boolean;
};

type IndexingBenchmarkApiRecord = {
    dataset_version?: string;
    corpus_source_path?: string;
    last_reviewed_on?: string;
    summary?: string;
    methodology?: unknown;
    pass_criteria?: unknown;
    limitations?: unknown;
    heuristic_quality_note?: string;
    corpus?: unknown;
    metrics?: unknown;
    cases?: unknown;
};

export type IndexingBenchmarkCorpusDocument = {
    id: string;
    title: string;
    mime_type: string;
    focus: string;
    query_count: number;
    expected_outline_titles: string[];
    expected_table_count: number;
    outline_node_count: number;
    table_count: number;
    structure_passed: boolean;
    retrieval_ready: boolean;
    heuristic_quality_score: number;
};

export type IndexingBenchmarkMetrics = {
    corpus_documents: number;
    document_queries: number;
    collection_queries: number;
    structure_pass_rate: number;
    tree_node_top1_accuracy: number;
    tree_node_recall_at_3: number;
    tree_page_top1_accuracy: number;
    tree_page_recall_at_3: number;
    collection_document_top1_accuracy: number;
    collection_document_recall_at_3: number;
    exact_match_accuracy: number;
    average_heuristic_score: number;
    minimum_heuristic_score: number;
    maximum_heuristic_score: number;
};

export type IndexingBenchmarkCase = {
    id: string;
    scope: string;
    query: string;
    document_title: string;
    expected_document_title: string;
    actual_document_title: string;
    document_matched: boolean;
    document_in_top_3: boolean;
    expected_node_title: string;
    actual_node_title: string;
    node_matched: boolean;
    node_in_top_3: boolean;
    expected_page_number: number | null;
    actual_page_number: number | null;
    page_matched: boolean;
    page_in_top_3: boolean;
    passed: boolean;
};

export type IndexingBenchmarkReport = {
    dataset_version: string;
    corpus_source_path: string;
    last_reviewed_on: string;
    summary: string;
    methodology: string[];
    pass_criteria: string[];
    limitations: string[];
    heuristic_quality_note: string;
    corpus: IndexingBenchmarkCorpusDocument[];
    metrics: IndexingBenchmarkMetrics;
    cases: IndexingBenchmarkCase[];
};

function normalizeStringList(value: unknown): string[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<string[]>((items, item) => {
        if (typeof item === "string" && item.trim()) {
            items.push(item.trim());
        }
        return items;
    }, []);
}

function normalizeCorpusDocument(record: unknown): IndexingBenchmarkCorpusDocument | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkCorpusDocumentApiRecord;
    if (!item.id || !item.title || !item.mime_type || !item.focus) {
        return null;
    }

    return {
        id: item.id,
        title: item.title,
        mime_type: item.mime_type,
        focus: item.focus,
        query_count: typeof item.query_count === "number" ? item.query_count : 0,
        expected_outline_titles: normalizeStringList(item.expected_outline_titles),
        expected_table_count: typeof item.expected_table_count === "number" ? item.expected_table_count : 0,
        outline_node_count: typeof item.outline_node_count === "number" ? item.outline_node_count : 0,
        table_count: typeof item.table_count === "number" ? item.table_count : 0,
        structure_passed: Boolean(item.structure_passed),
        retrieval_ready: Boolean(item.retrieval_ready),
        heuristic_quality_score: typeof item.heuristic_quality_score === "number" ? item.heuristic_quality_score : 0,
    };
}

function normalizeMetrics(record: unknown): IndexingBenchmarkMetrics {
    const metrics = (record && typeof record === "object" ? record : {}) as IndexingBenchmarkMetricsApiRecord;

    return {
        corpus_documents: typeof metrics.corpus_documents === "number" ? metrics.corpus_documents : 0,
        document_queries: typeof metrics.document_queries === "number" ? metrics.document_queries : 0,
        collection_queries: typeof metrics.collection_queries === "number" ? metrics.collection_queries : 0,
        structure_pass_rate: typeof metrics.structure_pass_rate === "number" ? metrics.structure_pass_rate : 0,
        tree_node_top1_accuracy: typeof metrics.tree_node_top1_accuracy === "number" ? metrics.tree_node_top1_accuracy : 0,
        tree_node_recall_at_3: typeof metrics.tree_node_recall_at_3 === "number" ? metrics.tree_node_recall_at_3 : 0,
        tree_page_top1_accuracy: typeof metrics.tree_page_top1_accuracy === "number" ? metrics.tree_page_top1_accuracy : 0,
        tree_page_recall_at_3: typeof metrics.tree_page_recall_at_3 === "number" ? metrics.tree_page_recall_at_3 : 0,
        collection_document_top1_accuracy:
            typeof metrics.collection_document_top1_accuracy === "number" ? metrics.collection_document_top1_accuracy : 0,
        collection_document_recall_at_3:
            typeof metrics.collection_document_recall_at_3 === "number" ? metrics.collection_document_recall_at_3 : 0,
        exact_match_accuracy: typeof metrics.exact_match_accuracy === "number" ? metrics.exact_match_accuracy : 0,
        average_heuristic_score: typeof metrics.average_heuristic_score === "number" ? metrics.average_heuristic_score : 0,
        minimum_heuristic_score: typeof metrics.minimum_heuristic_score === "number" ? metrics.minimum_heuristic_score : 0,
        maximum_heuristic_score: typeof metrics.maximum_heuristic_score === "number" ? metrics.maximum_heuristic_score : 0,
    };
}

function normalizeCase(record: unknown): IndexingBenchmarkCase | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkCaseApiRecord;
    if (!item.id || !item.scope || !item.query) {
        return null;
    }

    return {
        id: item.id,
        scope: item.scope,
        query: item.query,
        document_title: item.document_title ?? "",
        expected_document_title: item.expected_document_title ?? "",
        actual_document_title: item.actual_document_title ?? "",
        document_matched: Boolean(item.document_matched),
        document_in_top_3: Boolean(item.document_in_top_3),
        expected_node_title: item.expected_node_title ?? "",
        actual_node_title: item.actual_node_title ?? "",
        node_matched: Boolean(item.node_matched),
        node_in_top_3: Boolean(item.node_in_top_3),
        expected_page_number: typeof item.expected_page_number === "number" ? item.expected_page_number : null,
        actual_page_number: typeof item.actual_page_number === "number" ? item.actual_page_number : null,
        page_matched: Boolean(item.page_matched),
        page_in_top_3: Boolean(item.page_in_top_3),
        passed: Boolean(item.passed),
    };
}

function normalizeReport(record: unknown): IndexingBenchmarkReport | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const report = record as IndexingBenchmarkApiRecord;
    if (
        !report.dataset_version ||
        !report.corpus_source_path ||
        !report.last_reviewed_on ||
        !report.summary ||
        !report.heuristic_quality_note
    ) {
        return null;
    }

    return {
        dataset_version: report.dataset_version,
        corpus_source_path: report.corpus_source_path,
        last_reviewed_on: report.last_reviewed_on,
        summary: report.summary,
        methodology: normalizeStringList(report.methodology),
        pass_criteria: normalizeStringList(report.pass_criteria),
        limitations: normalizeStringList(report.limitations),
        heuristic_quality_note: report.heuristic_quality_note,
        corpus: Array.isArray(report.corpus)
            ? report.corpus.map(normalizeCorpusDocument).filter((item): item is IndexingBenchmarkCorpusDocument => item !== null)
            : [],
        metrics: normalizeMetrics(report.metrics),
        cases: Array.isArray(report.cases)
            ? report.cases.map(normalizeCase).filter((item): item is IndexingBenchmarkCase => item !== null)
            : [],
    };
}

export async function fetchIndexingBenchmark(signal?: AbortSignal): Promise<IndexingBenchmarkReport | null> {
    const response = await fetch("/api/runtime/indexing-benchmark", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("indexingBenchmark"));
    }

    return normalizeReport(hasResponseData(payload) ? payload.data : null);
}
