import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import type { Category } from "@/lib/api/categories";
import {
    getCollectionName,
    normalizeCollectionSummary,
    normalizeOutlineNodes,
    type CollectionSummary,
    type OutlineNode,
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
    outline_nodes?: unknown[];
    document_summary?: string;
};

export type DocumentIndex = {
    document: Document | null;
    document_summary: string;
    outline_nodes: OutlineNode[];
};

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
        document_summary: payload.data.document_summary ?? "",
        outline_nodes: normalizeOutlineNodes(payload.data.outline_nodes),
    };
}
