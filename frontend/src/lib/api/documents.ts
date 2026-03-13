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
    };
}
