import { apiErrorMessage } from "@/lib/api/error";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";
import { hasResponseData, type ApiErrorResponse, type ApiSuccessResponse } from "@/lib/api/types";

/** O que a LLM gera na indexacao dos documentos do usuario. */
export type IndexingPreferences = {
    documentSynopsis: boolean;
    pageSummaries: boolean;
    sectionSummaries: boolean;
};

type Record_ = { document_synopsis?: boolean; page_summaries?: boolean; section_summaries?: boolean };

function normalize(value: unknown): IndexingPreferences {
    const r = (value ?? {}) as Record_;
    return {
        documentSynopsis: r.document_synopsis !== false,
        pageSummaries: r.page_summaries !== false,
        sectionSummaries: r.section_summaries !== false,
    };
}

export async function fetchIndexingPreferences(signal?: AbortSignal): Promise<IndexingPreferences> {
    const response = await fetchWithAuthRedirect("/api/user/indexing-preferences", { method: "GET", cache: "no-store", signal });
    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Record_> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("indexing.preferences"));
    }
    return normalize(hasResponseData(payload) ? payload.data : null);
}

export async function saveIndexingPreferences(prefs: IndexingPreferences): Promise<IndexingPreferences> {
    const response = await fetchWithAuthRedirect("/api/user/indexing-preferences", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ document_synopsis: prefs.documentSynopsis, page_summaries: prefs.pageSummaries, section_summaries: prefs.sectionSummaries }),
    });
    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<Record_> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("indexing.preferences"));
    }
    return normalize(hasResponseData(payload) ? payload.data : prefs);
}
