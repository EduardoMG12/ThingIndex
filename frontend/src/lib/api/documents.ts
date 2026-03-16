import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
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
    metadata_pairs?: Array<{ key?: string; value?: string }> | null;
    created_at: string;
    updated_at: string;
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
    document_summary?: string;
    outline_nodes?: OutlineNode[];
    created_at: string;
    updated_at: string;
};

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
        document_summary:
            record.metadata_pairs?.find((item) => item?.key === "document_summary")?.value ?? undefined,
        outline_nodes: [],
        created_at: record.created_at,
        updated_at: record.updated_at,
    };
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

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentApiRecord> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel enviar o documento.");
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error("Resposta invalida ao enviar documento.");
    }

    return normalizeDocument(payload.data);
}

export async function fetchDocuments(signal?: AbortSignal): Promise<Document[]> {
    const response = await fetchWithAuthRedirect("/api/documents", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar os documentos.");
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeDocument);
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

export async function fetchDocumentIndex(documentId: number, signal?: AbortSignal): Promise<DocumentIndex> {
    const response = await fetchWithAuthRedirect(`/api/documents/${documentId}/index`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentIndexApiRecord> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar o indice do documento.");
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error("Resposta invalida ao carregar o indice do documento.");
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
