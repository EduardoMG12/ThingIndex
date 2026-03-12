import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";

export type Category = {
    id: number;
    user_id: number;
    name: string;
    description: string;
    created_at: string;
    updated_at: string;
};

export async function fetchCategories(signal?: AbortSignal): Promise<Category[]> {
    const response = await fetchWithAuthRedirect("/api/categories", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Category[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar as categorias.");
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data;
}

export async function createCategory(name: string): Promise<Category> {
    const response = await fetchWithAuthRedirect("/api/categories", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ name }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Category> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel criar a categoria.");
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error("Resposta invalida ao criar categoria.");
    }

    return payload.data;
}

export async function updateCategory(
    id: number,
    input: {
        name?: string;
        description?: string;
    }
): Promise<Category> {
    const response = await fetchWithAuthRedirect(`/api/categories/${id}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(input),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Category> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel atualizar a categoria.");
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error("Resposta invalida ao atualizar categoria.");
    }

    return payload.data;
}
