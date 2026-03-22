import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { apiErrorMessage } from "@/lib/api/error";
import { REQUEST_ID_HEADER_NAME } from "@/lib/api/request-id";
import type { Category } from "@/lib/api/categories";
import {
    getCollectionName,
    normalizeCollectionSummary,
    normalizeMetadataPairs,
    normalizeOutlineNodes,
    normalizeParagraphNodes,
    normalizeTableSections,
    type CollectionSummary,
    type MetadataPair,
    type OutlineNode,
    type ParagraphNode,
    type TableSection,
} from "@/lib/api/contracts/document-context";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import { extractFilenameFromDisposition } from "@/lib/http/content-disposition";

type DocumentApiRecord = {
    id: number;
    user_id: number;
    title: string;
    file_url?: string;
    file_path?: string;
    size: number;
    mime_type: string;
    status: string;
    processing_stage?: string;
    status_detail?: string;
    index_started_at?: string | null;
    indexed_at?: string | null;
    last_processing_error?: string;
    collection?: string | Record<string, unknown> | null;
    category_id?: number | null;
    category?: Category | null;
    processing_attempts?: Array<{
        id?: number;
        stage?: string;
        failure_kind?: string;
        disposition?: string;
        detail?: string;
        error_message?: string;
        retry_count?: number;
        next_retry_attempt?: number;
        max_retry_attempts?: number;
        retry_delay_seconds?: number;
        created_at?: string;
    }> | null;
    metadata_pairs?: Array<{ key?: string; value?: string }> | null;
    created_at: string;
    updated_at: string;
};

export type DocumentProcessingAttempt = {
    id: number;
    stage?: string;
    failure_kind?: string;
    disposition?: string;
    detail?: string;
    error_message?: string;
    retry_count: number;
    next_retry_attempt?: number;
    max_retry_attempts?: number;
    retry_delay_seconds?: number;
    created_at: string;
};

export type Document = {
    id: number;
    user_id: number;
    title: string;
    file_url: string;
    file_path: string;
    size: number;
    mime_type: string;
    status: string;
    processing_stage?: string;
    status_detail?: string;
    index_started_at?: string;
    indexed_at?: string;
    last_processing_error?: string;
    collection_id?: number | null;
    collection?: string;
    collection_ref?: CollectionSummary | null;
    category_id?: number | null;
    category?: Category | null;
    processing_attempts?: DocumentProcessingAttempt[];
    document_summary?: string;
    outline_nodes?: OutlineNode[];
    created_at: string;
    updated_at: string;
};

function normalizeProcessingAttempts(value: DocumentApiRecord["processing_attempts"]): DocumentProcessingAttempt[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<DocumentProcessingAttempt[]>((items, attempt, index) => {
        if (!attempt || typeof attempt.created_at !== "string") {
            return items;
        }

        items.push({
            id: typeof attempt.id === "number" ? attempt.id : index + 1,
            stage: typeof attempt.stage === "string" ? attempt.stage : undefined,
            failure_kind: typeof attempt.failure_kind === "string" ? attempt.failure_kind : undefined,
            disposition: typeof attempt.disposition === "string" ? attempt.disposition : undefined,
            detail: typeof attempt.detail === "string" ? attempt.detail : undefined,
            error_message: typeof attempt.error_message === "string" ? attempt.error_message : undefined,
            retry_count: typeof attempt.retry_count === "number" ? attempt.retry_count : 0,
            next_retry_attempt:
                typeof attempt.next_retry_attempt === "number" ? attempt.next_retry_attempt : undefined,
            max_retry_attempts:
                typeof attempt.max_retry_attempts === "number" ? attempt.max_retry_attempts : undefined,
            retry_delay_seconds:
                typeof attempt.retry_delay_seconds === "number" ? attempt.retry_delay_seconds : undefined,
            created_at: attempt.created_at,
        });

        return items;
    }, []);
}

function normalizeDocument(record: DocumentApiRecord): Document {
    const collectionRef = normalizeCollectionSummary(record.collection);

    return {
        id: record.id,
        user_id: record.user_id,
        title: record.title,
        file_url: record.file_url ?? record.file_path ?? "",
        file_path: record.file_path ?? record.file_url ?? "",
        size: record.size,
        mime_type: record.mime_type,
        status: record.status,
        processing_stage: record.processing_stage ?? undefined,
        status_detail: record.status_detail ?? undefined,
        index_started_at: record.index_started_at ?? undefined,
        indexed_at: record.indexed_at ?? undefined,
        last_processing_error: record.last_processing_error ?? undefined,
        collection_id:
            collectionRef && typeof collectionRef.id === "number"
                ? collectionRef.id
                : undefined,
        collection: collectionRef ? getCollectionName(collectionRef) : typeof record.collection === "string" ? record.collection : undefined,
        collection_ref: collectionRef,
        category_id: record.category_id,
        category: record.category,
        processing_attempts: normalizeProcessingAttempts(record.processing_attempts),
        document_summary:
            record.metadata_pairs?.find((item) => item?.key === "document_summary")?.value ?? undefined,
        outline_nodes: [],
        created_at: record.created_at,
        updated_at: record.updated_at,
    };
}

async function parseDocumentResponse(response: Response, fallbackMessage: string): Promise<Document> {
    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentApiRecord> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(buildApiErrorMessage(response, payload, fallbackMessage));
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error(apiErrorMessage("documents.fetchInvalid"));
    }

    return normalizeDocument(payload.data);
}

function buildApiErrorMessage(
    response: Response,
    payload: ApiErrorResponse | ApiSuccessResponse<unknown> | null,
    fallbackMessage: string
) {
    const baseMessage = payload?.message || fallbackMessage;
    const requestId =
        response.headers.get(REQUEST_ID_HEADER_NAME) ||
        (payload && "request_id" in payload && typeof payload.request_id === "string" ? payload.request_id : undefined);

    return requestId ? `${baseMessage} Request ID: ${requestId}` : baseMessage;
}

export async function uploadDocument(input: {
    title: string;
    file: File;
    category?: string;
    collection?: string;
}): Promise<Document> {
    const formData = new FormData();

    formData.append("title", input.title);
    formData.append("file", input.file);

    const category = input.category?.trim();
    if (category) {
        formData.append("category", category);
    }

    const collection = input.collection?.trim();
    if (collection) {
        formData.append("collection", collection);
    }

    const response = await fetchWithAuthRedirect("/api/documents/upload", {
        method: "POST",
        body: formData,
    });

    return parseDocumentResponse(response, apiErrorMessage("documents.upload"));
}

export async function fetchDocument(documentId: number, signal?: AbortSignal): Promise<Document> {
    const response = await fetchWithAuthRedirect(`/api/documents/${documentId}`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    return parseDocumentResponse(response, apiErrorMessage("documents.fetch"));
}

export async function fetchDocuments(signal?: AbortSignal): Promise<Document[]> {
    const response = await fetchWithAuthRedirect("/api/documents", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(buildApiErrorMessage(response, payload, apiErrorMessage("documents.list")));
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeDocument);
}

export async function retryDocumentIndexing(documentId: number): Promise<Document> {
    const response = await fetchWithAuthRedirect(`/api/documents/${documentId}/retry`, {
        method: "POST",
    });

    return parseDocumentResponse(response, apiErrorMessage("documents.retry"));
}

export async function deleteDocument(documentId: number): Promise<void> {
    const response = await fetchWithAuthRedirect(`/api/documents/${documentId}`, {
        method: "DELETE",
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<null> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(buildApiErrorMessage(response, payload, apiErrorMessage("documents.delete")));
    }
}

export async function deleteCollectionDocuments(collectionId: number): Promise<{ deleted_documents: number }> {
    const response = await fetchWithAuthRedirect(`/api/documents/collections/${collectionId}/documents`, {
        method: "DELETE",
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<{ deleted_documents?: number }> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(buildApiErrorMessage(response, payload, apiErrorMessage("documents.deleteCollection")));
    }

    if (!hasResponseData(payload) || !payload.data) {
        return { deleted_documents: 0 };
    }

    return {
        deleted_documents: typeof payload.data.deleted_documents === "number" ? payload.data.deleted_documents : 0,
    };
}

export async function downloadDocumentOriginal(document: Pick<Document, "id" | "title">): Promise<void> {
    if (typeof window === "undefined") {
        throw new Error("Download disponivel apenas no navegador.");
    }

    const response = await fetchWithAuthRedirect(`/api/documents/${document.id}/download`, {
        method: "GET",
        cache: "no-store",
    });

    const payload = !response.ok
        ? ((await response.json().catch(() => null)) as ApiErrorResponse | null)
        : null;

    if (!response.ok) {
        throw new Error(buildApiErrorMessage(response, payload, apiErrorMessage("documents.download")));
    }

    const blob = await response.blob();
    const objectURL = window.URL.createObjectURL(blob);
    const anchor = window.document.createElement("a");

    anchor.href = objectURL;
    anchor.download = extractFilenameFromDisposition(response.headers.get("content-disposition"), document.title);
    anchor.rel = "noopener";

    window.document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();

    window.setTimeout(() => {
        window.URL.revokeObjectURL(objectURL);
    }, 0);
}

type DocumentIndexApiRecord = {
    document?: DocumentApiRecord | null;
    paragraph_nodes?: unknown[];
    table_sections?: unknown[];
    metadata_pairs?: Array<{ key?: string; value?: string; document_id?: number; sequence_order?: number; id?: number | string }>;
    chunks?: Array<{
        id?: number | string;
        document_id?: number;
        sequence_order?: number;
        chunk_type?: string;
        title?: string;
        content?: string;
        source_paragraph_start?: number;
        source_paragraph_end?: number;
        token_estimate?: number;
        metadata?: Record<string, unknown>;
    }>;
    outline_nodes?: unknown[];
    document_summary?: string;
    monitoring?: {
        status?: string;
        stage?: string;
        status_detail?: string;
        started_at?: string | null;
        completed_at?: string | null;
        duration_ms?: number;
        enrichment_status?: string;
        pipeline_version?: string;
        prompt_profile_version?: string;
        chunk_strategy_version?: string;
        tree_builder?: string;
        tree_verification_passed?: boolean;
        estimated_quality_score?: number;
        llm?: {
            provider_type?: string;
            model?: string;
            request_count?: number;
            estimated_input_tokens?: number;
            estimated_output_tokens?: number;
            estimated_total_tokens?: number;
            pricing_available?: boolean;
            input_cost_usd?: number;
            output_cost_usd?: number;
            total_cost_usd?: number;
        } | null;
    } | null;
};

export type DocumentChunk = {
    id: number | string;
    document_id: number;
    sequence_order: number;
    chunk_type: string;
    title?: string;
    content: string;
    source_paragraph_start?: number;
    source_paragraph_end?: number;
    token_estimate: number;
    metadata?: Record<string, unknown>;
};

export type DocumentIndex = {
    document: Document | null;
    paragraph_nodes: ParagraphNode[];
    table_sections: TableSection[];
    metadata_pairs: MetadataPair[];
    chunks: DocumentChunk[];
    document_summary: string;
    outline_nodes: OutlineNode[];
    monitoring: IndexMonitoring | null;
};

export type LLMMonitoring = {
    provider_type?: string;
    model?: string;
    request_count: number;
    estimated_input_tokens: number;
    estimated_output_tokens: number;
    estimated_total_tokens: number;
    pricing_available: boolean;
    input_cost_usd: number;
    output_cost_usd: number;
    total_cost_usd: number;
};

export type IndexMonitoring = {
    status?: string;
    stage?: string;
    status_detail?: string;
    started_at?: string;
    completed_at?: string;
    duration_ms: number;
    enrichment_status?: string;
    pipeline_version?: string;
    prompt_profile_version?: string;
    chunk_strategy_version?: string;
    tree_builder?: string;
    tree_verification_passed: boolean;
    estimated_quality_score: number;
    llm: LLMMonitoring | null;
};

type DocumentMonitoringSummaryApiRecord = {
    total_documents?: number;
    indexed_documents?: number;
    processing_documents?: number;
    queued_documents?: number;
    failed_documents?: number;
    permanent_failed_documents?: number;
    retrying_documents?: number;
    orphaned_documents?: number;
    dlq_documents?: number;
    documents_with_llm?: number;
    pricing_aware_documents?: number;
    total_llm_requests?: number;
    total_input_tokens?: number;
    total_output_tokens?: number;
    total_tokens?: number;
    total_estimated_cost_usd?: number;
    average_duration_ms?: number;
    average_quality_score?: number;
    last_completed_at?: string | null;
    model_breakdown?: Array<{
        provider_type?: string;
        model?: string;
        document_count?: number;
        request_count?: number;
        total_tokens?: number;
        total_cost_usd?: number;
    }>;
};

export type DocumentMonitoringModelAggregate = {
    provider_type?: string;
    model?: string;
    document_count: number;
    request_count: number;
    total_tokens: number;
    total_cost_usd: number;
};

export type DocumentMonitoringSummary = {
    total_documents: number;
    indexed_documents: number;
    processing_documents: number;
    queued_documents: number;
    failed_documents: number;
    permanent_failed_documents: number;
    retrying_documents: number;
    orphaned_documents: number;
    dlq_documents: number;
    documents_with_llm: number;
    pricing_aware_documents: number;
    total_llm_requests: number;
    total_input_tokens: number;
    total_output_tokens: number;
    total_tokens: number;
    total_estimated_cost_usd: number;
    average_duration_ms: number;
    average_quality_score: number;
    last_completed_at?: string;
    model_breakdown: DocumentMonitoringModelAggregate[];
};

function normalizeDocumentChunks(value: DocumentIndexApiRecord["chunks"]): DocumentChunk[] {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.reduce<DocumentChunk[]>((items, chunk) => {
        if (
            !chunk ||
            typeof chunk.document_id !== "number" ||
            typeof chunk.sequence_order !== "number" ||
            typeof chunk.chunk_type !== "string" ||
            typeof chunk.content !== "string"
        ) {
            return items;
        }

        items.push({
            id: chunk.id ?? `chunk:${chunk.document_id}:${chunk.sequence_order}`,
            document_id: chunk.document_id,
            sequence_order: chunk.sequence_order,
            chunk_type: chunk.chunk_type,
            title: chunk.title,
            content: chunk.content,
            source_paragraph_start: chunk.source_paragraph_start,
            source_paragraph_end: chunk.source_paragraph_end,
            token_estimate: chunk.token_estimate ?? 0,
            metadata: chunk.metadata ?? {},
        });

        return items;
    }, []);
}

function normalizeIndexMonitoring(value: DocumentIndexApiRecord["monitoring"]): IndexMonitoring | null {
    if (!value || typeof value !== "object") {
        return null;
    }

    const llm =
        value.llm && typeof value.llm === "object"
            ? {
                  provider_type: typeof value.llm.provider_type === "string" ? value.llm.provider_type : undefined,
                  model: typeof value.llm.model === "string" ? value.llm.model : undefined,
                  request_count: typeof value.llm.request_count === "number" ? value.llm.request_count : 0,
                  estimated_input_tokens:
                      typeof value.llm.estimated_input_tokens === "number" ? value.llm.estimated_input_tokens : 0,
                  estimated_output_tokens:
                      typeof value.llm.estimated_output_tokens === "number" ? value.llm.estimated_output_tokens : 0,
                  estimated_total_tokens:
                      typeof value.llm.estimated_total_tokens === "number" ? value.llm.estimated_total_tokens : 0,
                  pricing_available: Boolean(value.llm.pricing_available),
                  input_cost_usd: typeof value.llm.input_cost_usd === "number" ? value.llm.input_cost_usd : 0,
                  output_cost_usd: typeof value.llm.output_cost_usd === "number" ? value.llm.output_cost_usd : 0,
                  total_cost_usd: typeof value.llm.total_cost_usd === "number" ? value.llm.total_cost_usd : 0,
              }
            : null;

    return {
        status: typeof value.status === "string" ? value.status : undefined,
        stage: typeof value.stage === "string" ? value.stage : undefined,
        status_detail: typeof value.status_detail === "string" ? value.status_detail : undefined,
        started_at: typeof value.started_at === "string" ? value.started_at : undefined,
        completed_at: typeof value.completed_at === "string" ? value.completed_at : undefined,
        duration_ms: typeof value.duration_ms === "number" ? value.duration_ms : 0,
        enrichment_status: typeof value.enrichment_status === "string" ? value.enrichment_status : undefined,
        pipeline_version: typeof value.pipeline_version === "string" ? value.pipeline_version : undefined,
        prompt_profile_version:
            typeof value.prompt_profile_version === "string" ? value.prompt_profile_version : undefined,
        chunk_strategy_version:
            typeof value.chunk_strategy_version === "string" ? value.chunk_strategy_version : undefined,
        tree_builder: typeof value.tree_builder === "string" ? value.tree_builder : undefined,
        tree_verification_passed: Boolean(value.tree_verification_passed),
        estimated_quality_score:
            typeof value.estimated_quality_score === "number" ? value.estimated_quality_score : 0,
        llm,
    };
}

function normalizeDocumentMonitoringSummary(value: DocumentMonitoringSummaryApiRecord | null | undefined): DocumentMonitoringSummary {
    const modelBreakdown = Array.isArray(value?.model_breakdown)
        ? value.model_breakdown.map((item) => ({
              provider_type: typeof item?.provider_type === "string" ? item.provider_type : undefined,
              model: typeof item?.model === "string" ? item.model : undefined,
              document_count: typeof item?.document_count === "number" ? item.document_count : 0,
              request_count: typeof item?.request_count === "number" ? item.request_count : 0,
              total_tokens: typeof item?.total_tokens === "number" ? item.total_tokens : 0,
              total_cost_usd: typeof item?.total_cost_usd === "number" ? item.total_cost_usd : 0,
          }))
        : [];

    return {
        total_documents: typeof value?.total_documents === "number" ? value.total_documents : 0,
        indexed_documents: typeof value?.indexed_documents === "number" ? value.indexed_documents : 0,
        processing_documents: typeof value?.processing_documents === "number" ? value.processing_documents : 0,
        queued_documents: typeof value?.queued_documents === "number" ? value.queued_documents : 0,
        failed_documents: typeof value?.failed_documents === "number" ? value.failed_documents : 0,
        permanent_failed_documents:
            typeof value?.permanent_failed_documents === "number" ? value.permanent_failed_documents : 0,
        retrying_documents: typeof value?.retrying_documents === "number" ? value.retrying_documents : 0,
        orphaned_documents: typeof value?.orphaned_documents === "number" ? value.orphaned_documents : 0,
        dlq_documents: typeof value?.dlq_documents === "number" ? value.dlq_documents : 0,
        documents_with_llm: typeof value?.documents_with_llm === "number" ? value.documents_with_llm : 0,
        pricing_aware_documents: typeof value?.pricing_aware_documents === "number" ? value.pricing_aware_documents : 0,
        total_llm_requests: typeof value?.total_llm_requests === "number" ? value.total_llm_requests : 0,
        total_input_tokens: typeof value?.total_input_tokens === "number" ? value.total_input_tokens : 0,
        total_output_tokens: typeof value?.total_output_tokens === "number" ? value.total_output_tokens : 0,
        total_tokens: typeof value?.total_tokens === "number" ? value.total_tokens : 0,
        total_estimated_cost_usd:
            typeof value?.total_estimated_cost_usd === "number" ? value.total_estimated_cost_usd : 0,
        average_duration_ms: typeof value?.average_duration_ms === "number" ? value.average_duration_ms : 0,
        average_quality_score:
            typeof value?.average_quality_score === "number" ? value.average_quality_score : 0,
        last_completed_at: typeof value?.last_completed_at === "string" ? value.last_completed_at : undefined,
        model_breakdown: modelBreakdown,
    };
}

export async function fetchDocumentIndex(documentId: number, signal?: AbortSignal): Promise<DocumentIndex> {
    const response = await fetchWithAuthRedirect(`/api/documents/${documentId}/index`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentIndexApiRecord> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("documents.index"));
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error(apiErrorMessage("documents.indexInvalid"));
    }

    return {
        document: payload.data.document ? normalizeDocument(payload.data.document) : null,
        paragraph_nodes: normalizeParagraphNodes(payload.data.paragraph_nodes),
        table_sections: normalizeTableSections(payload.data.table_sections),
        metadata_pairs: normalizeMetadataPairs(payload.data.metadata_pairs),
        chunks: normalizeDocumentChunks(payload.data.chunks),
        document_summary: payload.data.document_summary ?? "",
        outline_nodes: normalizeOutlineNodes(payload.data.outline_nodes),
        monitoring: normalizeIndexMonitoring(payload.data.monitoring),
    };
}

export async function fetchDocumentMonitoringSummary(signal?: AbortSignal): Promise<DocumentMonitoringSummary> {
    const response = await fetchWithAuthRedirect("/api/documents/monitoring/summary", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as
        | ApiSuccessResponse<DocumentMonitoringSummaryApiRecord>
        | ApiErrorResponse
        | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("documents.monitoringSummary"));
    }

    if (!hasResponseData(payload) || !payload.data) {
        return normalizeDocumentMonitoringSummary(null);
    }

    return normalizeDocumentMonitoringSummary(payload.data);
}
