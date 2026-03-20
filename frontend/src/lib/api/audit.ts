import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";

type DocumentAuditEventApiRecord = {
    id: number;
    user_id: number;
    document_id?: number | null;
    collection_id?: number | null;
    actor_type?: string;
    actor_id?: number | null;
    action?: string;
    status?: string;
    correlation_id?: string;
    origin?: string;
    title?: string;
    detail?: string;
    metadata?: Record<string, unknown> | null;
    created_at?: string;
};

type ProviderAuditEventApiRecord = {
    id: number;
    user_id: number;
    provider_config_id?: number | null;
    family?: string;
    provider_type?: string;
    actor_type?: string;
    actor_id?: number | null;
    action?: string;
    status?: string;
    correlation_id?: string;
    origin?: string;
    detail?: string;
    metadata?: Record<string, unknown> | null;
    created_at?: string;
};

export type DocumentAuditEvent = {
    id: number;
    user_id: number;
    document_id?: number;
    collection_id?: number;
    actor_type: string;
    actor_id?: number;
    action: string;
    status: string;
    correlation_id?: string;
    origin?: string;
    title?: string;
    detail?: string;
    metadata: Record<string, unknown>;
    created_at: string;
};

export type ProviderAuditEvent = {
    id: number;
    user_id: number;
    provider_config_id?: number;
    family?: string;
    provider_type?: string;
    actor_type: string;
    actor_id?: number;
    action: string;
    status: string;
    correlation_id?: string;
    origin?: string;
    detail?: string;
    metadata: Record<string, unknown>;
    created_at: string;
};

function normalizeMetadata(value: Record<string, unknown> | null | undefined) {
    return value && typeof value === "object" ? value : {};
}

function normalizeDocumentAuditEvent(record: DocumentAuditEventApiRecord): DocumentAuditEvent {
    return {
        id: record.id,
        user_id: record.user_id,
        document_id: typeof record.document_id === "number" ? record.document_id : undefined,
        collection_id: typeof record.collection_id === "number" ? record.collection_id : undefined,
        actor_type: typeof record.actor_type === "string" ? record.actor_type : "USER",
        actor_id: typeof record.actor_id === "number" ? record.actor_id : undefined,
        action: typeof record.action === "string" ? record.action : "UNKNOWN",
        status: typeof record.status === "string" ? record.status : "UNKNOWN",
        correlation_id: typeof record.correlation_id === "string" ? record.correlation_id : undefined,
        origin: typeof record.origin === "string" ? record.origin : undefined,
        title: typeof record.title === "string" ? record.title : undefined,
        detail: typeof record.detail === "string" ? record.detail : undefined,
        metadata: normalizeMetadata(record.metadata),
        created_at: typeof record.created_at === "string" ? record.created_at : new Date(0).toISOString(),
    };
}

function normalizeProviderAuditEvent(record: ProviderAuditEventApiRecord): ProviderAuditEvent {
    return {
        id: record.id,
        user_id: record.user_id,
        provider_config_id:
            typeof record.provider_config_id === "number" ? record.provider_config_id : undefined,
        family: typeof record.family === "string" ? record.family : undefined,
        provider_type: typeof record.provider_type === "string" ? record.provider_type : undefined,
        actor_type: typeof record.actor_type === "string" ? record.actor_type : "USER",
        actor_id: typeof record.actor_id === "number" ? record.actor_id : undefined,
        action: typeof record.action === "string" ? record.action : "UNKNOWN",
        status: typeof record.status === "string" ? record.status : "UNKNOWN",
        correlation_id: typeof record.correlation_id === "string" ? record.correlation_id : undefined,
        origin: typeof record.origin === "string" ? record.origin : undefined,
        detail: typeof record.detail === "string" ? record.detail : undefined,
        metadata: normalizeMetadata(record.metadata),
        created_at: typeof record.created_at === "string" ? record.created_at : new Date(0).toISOString(),
    };
}

function buildAuditQuery(input: Record<string, string | number | undefined>) {
    const params = new URLSearchParams();

    for (const [key, value] of Object.entries(input)) {
        if (value === undefined) {
            continue;
        }

        const normalized = String(value).trim();
        if (!normalized) {
            continue;
        }

        params.set(key, normalized);
    }

    return params.toString();
}

export async function fetchDocumentAuditEvents(input: {
    limit?: number;
    action?: string;
    status?: string;
    documentId?: number;
    collectionId?: number;
    signal?: AbortSignal;
} = {}): Promise<DocumentAuditEvent[]> {
    const query = buildAuditQuery({
        limit: input.limit,
        action: input.action,
        status: input.status,
        document_id: input.documentId,
        collection_id: input.collectionId,
    });

    const response = await fetchWithAuthRedirect(`/api/documents/audit${query ? `?${query}` : ""}`, {
        method: "GET",
        cache: "no-store",
        signal: input.signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentAuditEventApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar a auditoria documental.");
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeDocumentAuditEvent);
}

export async function fetchProviderAuditEvents(input: {
    limit?: number;
    family?: string;
    providerType?: string;
    action?: string;
    status?: string;
    signal?: AbortSignal;
} = {}): Promise<ProviderAuditEvent[]> {
    const query = buildAuditQuery({
        limit: input.limit,
        family: input.family,
        provider_type: input.providerType,
        action: input.action,
        status: input.status,
    });

    const response = await fetchWithAuthRedirect(`/api/user/providers/audit${query ? `?${query}` : ""}`, {
        method: "GET",
        cache: "no-store",
        signal: input.signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<ProviderAuditEventApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel carregar a auditoria de providers.");
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeProviderAuditEvent);
}
