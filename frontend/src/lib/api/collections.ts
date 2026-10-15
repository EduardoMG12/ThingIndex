import { normalizeViewState, type WorkspaceViewState } from "@/lib/api/contracts/view-state";
import {
    normalizeHighlight,
    normalizeHighlights,
    normalizeLegend,
    type Highlight,
    type HighlightColor,
    type Legend,
} from "@/lib/api/contracts/highlights";
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
import {
    normalizeNodeConnections,
    type NodeConnectionsResult,
} from "@/lib/api/contracts/node-connections";
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

// minScore explicito: subir o limiar tem de poder zerar as conexoes.
export async function fetchNodeConnections(
    collectionId: number,
    nodeId: string,
    minScore?: number,
    signal?: AbortSignal
): Promise<NodeConnectionsResult> {
    const params = new URLSearchParams({ node_id: nodeId });
    if (typeof minScore === "number") {
        params.set("min_score", String(minScore));
    }

    const response = await fetchWithAuthRedirect(
        `/api/documents/collections/${collectionId}/connections?${params.toString()}`,
        {
            method: "GET",
            cache: "no-store",
            signal,
        }
    );

    const payload = (await response.json().catch(() => null)) as
        | ApiSuccessResponse<unknown>
        | ApiErrorResponse
        | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("collections.connections"));
    }

    return normalizeNodeConnections(hasResponseData(payload) ? payload.data : null);
}

// Anotacoes (marcas e legenda). Sem redirect em 401: quem decide se a sessao
// acabou e o carregamento do contexto; aqui vira erro na sidebar.

const SEM_REDIRECIONAR = { redirectOnUnauthorized: false } as const;

async function lerJson(response: Response, chaveDeErro: string): Promise<unknown> {
    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage(chaveDeErro));
    }
    return payload && "data" in payload ? payload.data : null;
}

export async function fetchHighlights(collectionId: number, signal?: AbortSignal): Promise<Highlight[]> {
    const response = await fetchWithAuthRedirect(`/api/documents/collections/${collectionId}/highlights`, {
        method: "GET",
        cache: "no-store",
        signal,
    }, SEM_REDIRECIONAR);
    return normalizeHighlights(await lerJson(response, "collections.highlights"));
}

export async function putHighlight(collectionId: number, paragraphNodeId: number, color: HighlightColor): Promise<Highlight | null> {
    const response = await fetchWithAuthRedirect(`/api/documents/collections/${collectionId}/highlights/${paragraphNodeId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ color }),
    }, SEM_REDIRECIONAR);
    return normalizeHighlight(await lerJson(response, "collections.highlightSave"));
}

export async function deleteHighlight(collectionId: number, paragraphNodeId: number): Promise<void> {
    const response = await fetchWithAuthRedirect(`/api/documents/collections/${collectionId}/highlights/${paragraphNodeId}`, {
        method: "DELETE",
    }, SEM_REDIRECIONAR);
    await lerJson(response, "collections.highlightRemove");
}

export async function fetchLegend(collectionId: number, signal?: AbortSignal): Promise<Legend> {
    const response = await fetchWithAuthRedirect(`/api/documents/collections/${collectionId}/legend`, {
        method: "GET",
        cache: "no-store",
        signal,
    }, SEM_REDIRECIONAR);
    return normalizeLegend(await lerJson(response, "collections.legend"));
}

export async function putLegend(collectionId: number, labels: Legend): Promise<Legend> {
    const response = await fetchWithAuthRedirect(`/api/documents/collections/${collectionId}/legend`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ labels }),
    }, SEM_REDIRECIONAR);
    return normalizeLegend(await lerJson(response, "collections.legendSave"));
}

export async function fetchViewState(collectionId: number, signal?: AbortSignal): Promise<WorkspaceViewState> {
    const response = await fetchWithAuthRedirect(
        `/api/documents/collections/${collectionId}/view-state`,
        { method: "GET", cache: "no-store", signal },
        SEM_REDIRECIONAR
    );
    return normalizeViewState(await lerJson(response, "collections.viewState"));
}

/** `keepalive` deixa o envio sobreviver ao fechamento da aba (pagehide). */
export async function putViewState(collectionId: number, estado: WorkspaceViewState, opcoes: { keepalive?: boolean } = {}): Promise<void> {
    const response = await fetchWithAuthRedirect(
        `/api/documents/collections/${collectionId}/view-state`,
        {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ state: estado }),
            keepalive: opcoes.keepalive ?? false,
        },
        SEM_REDIRECIONAR
    );
    await lerJson(response, "collections.viewStateSave");
}
