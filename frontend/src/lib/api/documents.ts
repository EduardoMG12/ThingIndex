import type { ApiErrorResponse, ApiSuccessResponse } from "@/lib/api/types";
import type { Category } from "@/lib/api/categories";

export type Document = {
    id: number;
    user_id: number;
    title: string;
    file_url: string;
    size: number;
    mime_type: string;
    status: string;
    collection?: string;
    category_id?: number | null;
    category?: Category | null;
    created_at: string;
    updated_at: string;
};

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

    const response = await fetch("/api/documents/upload", {
        method: "POST",
        body: formData,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Document> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel enviar o documento.");
    }

    if (!payload || !("data" in payload) || !payload.data) {
        throw new Error("Resposta invalida ao enviar documento.");
    }

    return payload.data;
}
