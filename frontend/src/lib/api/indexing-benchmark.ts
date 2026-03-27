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

type IndexingBenchmarkSnapshotApiRecord = {
    id?: string;
    label?: string;
    recorded_on?: string;
    summary?: string;
    dataset_version?: string;
    pipeline_version?: string;
    prompt_profile_version?: string;
    chunk_strategy_version?: string;
    metrics?: unknown;
};

type IndexingBenchmarkMetricDeltaApiRecord = {
    metric?: string;
    previous_value?: number;
    current_value?: number;
    delta?: number;
};

type IndexingBenchmarkComparisonApiRecord = {
    id?: string;
    title?: string;
    recorded_on?: string;
    summary?: string;
    before?: unknown;
    after?: unknown;
    key_metric_deltas?: unknown;
};

type IndexingBenchmarkReferenceArtifactApiRecord = {
    id?: string;
    label?: string;
    source_path?: string;
    document_name?: string;
    total_nodes?: number;
    max_depth?: number;
    summary_coverage_ratio?: number;
    invalid_range_count?: number;
    average_title_length?: number;
    average_summary_length?: number;
    notes?: unknown;
};

type IndexingBenchmarkReferenceChecklistItemApiRecord = {
    id?: string;
    dimension?: string;
    label?: string;
    status?: string;
    current_value?: string;
    reference_value?: string;
    summary?: string;
};

type IndexingBenchmarkReferenceComparisonApiRecord = {
    source_path?: string;
    last_reviewed_on?: string;
    summary?: string;
    artifacts?: unknown;
    checklist?: unknown;
};

type IndexingBenchmarkStructuralSignalApiRecord = {
    id?: string;
    direction?: string;
    status?: string;
    rate?: number;
    count?: number;
    total?: number;
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
    structural_signals?: unknown;
    cases?: unknown;
    current_snapshot?: unknown;
    latest_comparison?: unknown;
    comparison_history?: unknown;
    reference_comparison?: unknown;
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

export type IndexingBenchmarkSnapshot = {
    id: string;
    label: string;
    recorded_on: string;
    summary: string;
    dataset_version: string;
    pipeline_version: string;
    prompt_profile_version: string;
    chunk_strategy_version: string;
    metrics: IndexingBenchmarkMetrics;
};

export type IndexingBenchmarkMetricDelta = {
    metric: string;
    previous_value: number;
    current_value: number;
    delta: number;
};

export type IndexingBenchmarkComparison = {
    id: string;
    title: string;
    recorded_on: string;
    summary: string;
    before: IndexingBenchmarkSnapshot;
    after: IndexingBenchmarkSnapshot;
    key_metric_deltas: IndexingBenchmarkMetricDelta[];
};

export type IndexingBenchmarkReferenceArtifact = {
    id: string;
    label: string;
    source_path: string;
    document_name: string;
    total_nodes: number;
    max_depth: number;
    summary_coverage_ratio: number;
    invalid_range_count: number;
    average_title_length: number;
    average_summary_length: number;
    notes: string[];
};

export type IndexingBenchmarkReferenceChecklistItem = {
    id: string;
    dimension: string;
    label: string;
    status: string;
    current_value: string;
    reference_value: string;
    summary: string;
};

export type IndexingBenchmarkReferenceComparison = {
    source_path: string;
    last_reviewed_on: string;
    summary: string;
    artifacts: IndexingBenchmarkReferenceArtifact[];
    checklist: IndexingBenchmarkReferenceChecklistItem[];
};

export type IndexingBenchmarkStructuralSignal = {
    id: string;
    direction: string;
    status: string;
    rate: number;
    count: number;
    total: number;
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
    structural_signals: IndexingBenchmarkStructuralSignal[];
    cases: IndexingBenchmarkCase[];
    current_snapshot?: IndexingBenchmarkSnapshot;
    latest_comparison?: IndexingBenchmarkComparison;
    comparison_history: IndexingBenchmarkComparison[];
    reference_comparison?: IndexingBenchmarkReferenceComparison;
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

function normalizeSnapshot(record: unknown): IndexingBenchmarkSnapshot | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkSnapshotApiRecord;
    if (!item.id || !item.label || !item.recorded_on || !item.dataset_version) {
        return null;
    }

    return {
        id: item.id,
        label: item.label,
        recorded_on: item.recorded_on,
        summary: item.summary ?? "",
        dataset_version: item.dataset_version,
        pipeline_version: item.pipeline_version ?? "",
        prompt_profile_version: item.prompt_profile_version ?? "",
        chunk_strategy_version: item.chunk_strategy_version ?? "",
        metrics: normalizeMetrics(item.metrics),
    };
}

function normalizeMetricDelta(record: unknown): IndexingBenchmarkMetricDelta | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkMetricDeltaApiRecord;
    if (!item.metric) {
        return null;
    }

    return {
        metric: item.metric,
        previous_value: typeof item.previous_value === "number" ? item.previous_value : 0,
        current_value: typeof item.current_value === "number" ? item.current_value : 0,
        delta: typeof item.delta === "number" ? item.delta : 0,
    };
}

function normalizeComparison(record: unknown): IndexingBenchmarkComparison | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkComparisonApiRecord;
    const before = normalizeSnapshot(item.before);
    const after = normalizeSnapshot(item.after);
    if (!item.id || !item.title || !item.recorded_on || before === null || after === null) {
        return null;
    }

    return {
        id: item.id,
        title: item.title,
        recorded_on: item.recorded_on,
        summary: item.summary ?? "",
        before,
        after,
        key_metric_deltas: Array.isArray(item.key_metric_deltas)
            ? item.key_metric_deltas.map(normalizeMetricDelta).filter((delta): delta is IndexingBenchmarkMetricDelta => delta !== null)
            : [],
    };
}

function normalizeReferenceArtifact(record: unknown): IndexingBenchmarkReferenceArtifact | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkReferenceArtifactApiRecord;
    if (!item.id || !item.label || !item.source_path || !item.document_name) {
        return null;
    }

    return {
        id: item.id,
        label: item.label,
        source_path: item.source_path,
        document_name: item.document_name,
        total_nodes: typeof item.total_nodes === "number" ? item.total_nodes : 0,
        max_depth: typeof item.max_depth === "number" ? item.max_depth : 0,
        summary_coverage_ratio: typeof item.summary_coverage_ratio === "number" ? item.summary_coverage_ratio : 0,
        invalid_range_count: typeof item.invalid_range_count === "number" ? item.invalid_range_count : 0,
        average_title_length: typeof item.average_title_length === "number" ? item.average_title_length : 0,
        average_summary_length: typeof item.average_summary_length === "number" ? item.average_summary_length : 0,
        notes: normalizeStringList(item.notes),
    };
}

function normalizeReferenceChecklistItem(record: unknown): IndexingBenchmarkReferenceChecklistItem | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkReferenceChecklistItemApiRecord;
    if (!item.id || !item.dimension || !item.label || !item.status) {
        return null;
    }

    return {
        id: item.id,
        dimension: item.dimension,
        label: item.label,
        status: item.status,
        current_value: item.current_value ?? "",
        reference_value: item.reference_value ?? "",
        summary: item.summary ?? "",
    };
}

function normalizeReferenceComparison(record: unknown): IndexingBenchmarkReferenceComparison | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkReferenceComparisonApiRecord;
    if (!item.source_path || !item.last_reviewed_on || !item.summary) {
        return null;
    }

    return {
        source_path: item.source_path,
        last_reviewed_on: item.last_reviewed_on,
        summary: item.summary,
        artifacts: Array.isArray(item.artifacts)
            ? item.artifacts.map(normalizeReferenceArtifact).filter((artifact): artifact is IndexingBenchmarkReferenceArtifact => artifact !== null)
            : [],
        checklist: Array.isArray(item.checklist)
            ? item.checklist
                .map(normalizeReferenceChecklistItem)
                .filter((checklistItem): checklistItem is IndexingBenchmarkReferenceChecklistItem => checklistItem !== null)
            : [],
    };
}

function normalizeStructuralSignal(record: unknown): IndexingBenchmarkStructuralSignal | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const item = record as IndexingBenchmarkStructuralSignalApiRecord;
    if (!item.id || !item.direction || !item.status) {
        return null;
    }

    return {
        id: item.id,
        direction: item.direction,
        status: item.status,
        rate: typeof item.rate === "number" ? item.rate : 0,
        count: typeof item.count === "number" ? item.count : 0,
        total: typeof item.total === "number" ? item.total : 0,
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
        structural_signals: Array.isArray(report.structural_signals)
            ? report.structural_signals
                .map(normalizeStructuralSignal)
                .filter((item): item is IndexingBenchmarkStructuralSignal => item !== null)
            : [],
        cases: Array.isArray(report.cases)
            ? report.cases.map(normalizeCase).filter((item): item is IndexingBenchmarkCase => item !== null)
            : [],
        current_snapshot: normalizeSnapshot(report.current_snapshot) ?? undefined,
        latest_comparison: normalizeComparison(report.latest_comparison) ?? undefined,
        comparison_history: Array.isArray(report.comparison_history)
            ? report.comparison_history
                .map(normalizeComparison)
                .filter((item): item is IndexingBenchmarkComparison => item !== null)
            : [],
        reference_comparison: normalizeReferenceComparison(report.reference_comparison) ?? undefined,
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
