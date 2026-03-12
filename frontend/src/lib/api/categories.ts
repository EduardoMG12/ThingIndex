import type { ApiErrorResponse, ApiSuccessResponse } from "@/lib/api/types";

export type Category = {
    id: number;
    user_id: number;
    name: string;
    description: string;
    created_at: string;
    updated_at: string;
};

export async function fetchCategories(signal?: AbortSignal): Promise<Category[]> {
    const response = await fetch("/api/categories", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Category[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar as categorias.");
    }

    return Array.isArray(payload?.data) ? payload.data : [];
}
