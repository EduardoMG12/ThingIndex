import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { apiErrorMessage } from "@/lib/api/error";
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
        throw new Error(payload?.message || apiErrorMessage("categories.list"));
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
        throw new Error(payload?.message || apiErrorMessage("categories.create"));
    }

    if (!hasResponseData(payload) || !payload.data) {
        throw new Error(apiErrorMessage("categories.createInvalid"));
    }

    return payload.data;
}
