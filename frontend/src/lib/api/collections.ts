import type { ApiErrorResponse, ApiSuccessResponse } from "@/lib/api/types";
import {
    normalizeCollectionSummaries,
    type CollectionSummary,
} from "@/lib/api/contracts/document-context";

export async function fetchCollections(signal?: AbortSignal): Promise<CollectionSummary[]> {
    const response = await fetch("/api/collections", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar as collections.");
    }

    return normalizeCollectionSummaries(payload?.data);
}
