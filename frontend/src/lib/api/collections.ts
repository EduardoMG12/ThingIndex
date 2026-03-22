import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { apiErrorMessage } from "@/lib/api/error";
import {
    normalizeCollectionContext,
    normalizeCollectionSummaries,
    type CollectionContext,
    type CollectionSummary,
} from "@/lib/api/contracts/document-context";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";

export async function fetchCollections(signal?: AbortSignal): Promise<CollectionSummary[]> {
    const response = await fetchWithAuthRedirect("/api/collections", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("collections.list"));
    }

    return normalizeCollectionSummaries(hasResponseData(payload) ? payload.data : null);
}

export async function fetchCollectionContext(
    collectionId: number,
    signal?: AbortSignal
): Promise<CollectionContext> {
    const response = await fetchWithAuthRedirect(`/api/collections/${collectionId}/context`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("collections.context"));
    }

    const normalized = normalizeCollectionContext(hasResponseData(payload) ? payload.data : null);
    if (!normalized) {
        throw new Error(apiErrorMessage("collections.contextInvalid"));
    }

    return normalized;
}
