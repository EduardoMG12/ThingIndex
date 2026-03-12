import type { ApiErrorResponse, ApiSuccessResponse } from "@/lib/api/types";

export async function fetchCollections(signal?: AbortSignal): Promise<string[]> {
    const response = await fetch("/api/collections", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<string[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar as collections.");
    }

    return Array.isArray(payload?.data) ? payload.data : [];
}
