import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import type { Category } from "@/lib/api/categories";
import {
    getCollectionName,
    normalizeCollectionSummary,
    type CollectionSummary,
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
    collection?: string;
    collection_ref?: CollectionSummary | null;
    category_id?: number | null;
    category?: Category | null;
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
        collection: collectionRef ? getCollectionName(collectionRef) : typeof record.collection === "string" ? record.collection : undefined,
        collection_ref: collectionRef,
        category_id: record.category_id,
        category: record.category,
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
