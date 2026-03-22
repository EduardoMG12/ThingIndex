import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { apiErrorMessage } from "@/lib/api/error";
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

type AuditCountBucketApiRecord = {
    key?: string;
    count?: number;
};

type AuditPageInfoApiRecord = {
    limit?: number;
    returned?: number;
    has_next?: boolean;
    next_cursor?: string;
};

type DocumentAuditSummaryApiRecord = {
    total_events?: number;
    unique_documents?: number;
    unique_collections?: number;
    unique_requests?: number;
    last_event_at?: string | null;
    action_breakdown?: AuditCountBucketApiRecord[] | null;
    status_breakdown?: AuditCountBucketApiRecord[] | null;
};

type ProviderAuditSummaryApiRecord = {
    total_events?: number;
    unique_providers?: number;
    unique_requests?: number;
    last_event_at?: string | null;
    action_breakdown?: AuditCountBucketApiRecord[] | null;
    status_breakdown?: AuditCountBucketApiRecord[] | null;
};

type DocumentAuditQueryApiRecord = {
    items?: DocumentAuditEventApiRecord[] | null;
    summary?: DocumentAuditSummaryApiRecord | null;
    page_info?: AuditPageInfoApiRecord | null;
};

type ProviderAuditQueryApiRecord = {
    items?: ProviderAuditEventApiRecord[] | null;
    summary?: ProviderAuditSummaryApiRecord | null;
    page_info?: AuditPageInfoApiRecord | null;
};

export type AuditCountBucket = {
    key: string;
    count: number;
};

export type AuditPageInfo = {
    limit: number;
    returned: number;
    has_next: boolean;
    next_cursor?: string;
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

export type DocumentAuditSummary = {
    total_events: number;
    unique_documents: number;
    unique_collections: number;
    unique_requests: number;
    last_event_at?: string;
    action_breakdown: AuditCountBucket[];
    status_breakdown: AuditCountBucket[];
};

export type ProviderAuditSummary = {
    total_events: number;
    unique_providers: number;
    unique_requests: number;
    last_event_at?: string;
    action_breakdown: AuditCountBucket[];
    status_breakdown: AuditCountBucket[];
};

export type DocumentAuditQueryResult = {
    items: DocumentAuditEvent[];
    summary: DocumentAuditSummary;
    page_info: AuditPageInfo;
};

export type ProviderAuditQueryResult = {
    items: ProviderAuditEvent[];
    summary: ProviderAuditSummary;
    page_info: AuditPageInfo;
};

function normalizeMetadata(value: Record<string, unknown> | null | undefined) {
    return value && typeof value === "object" ? value : {};
}

function normalizeAuditBuckets(input: AuditCountBucketApiRecord[] | null | undefined): AuditCountBucket[] {
    if (!Array.isArray(input)) {
        return [];
    }

    return input
        .filter((item) => typeof item?.key === "string")
        .map((item) => ({
            key: String(item.key),
            count: typeof item.count === "number" ? item.count : 0,
        }));
}

function normalizeAuditPageInfo(input: AuditPageInfoApiRecord | null | undefined, fallbackReturned = 0): AuditPageInfo {
    return {
        limit: typeof input?.limit === "number" ? input.limit : fallbackReturned,
        returned: typeof input?.returned === "number" ? input.returned : fallbackReturned,
        has_next: Boolean(input?.has_next),
        next_cursor: typeof input?.next_cursor === "string" ? input.next_cursor : undefined,
    };
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

function normalizeDocumentAuditQueryRecord(record: DocumentAuditQueryApiRecord): DocumentAuditQueryResult {
    const items = Array.isArray(record.items) ? record.items.map(normalizeDocumentAuditEvent) : [];

    return {
        items,
        summary: {
            total_events: typeof record.summary?.total_events === "number" ? record.summary.total_events : items.length,
            unique_documents: typeof record.summary?.unique_documents === "number" ? record.summary.unique_documents : 0,
            unique_collections: typeof record.summary?.unique_collections === "number" ? record.summary.unique_collections : 0,
            unique_requests: typeof record.summary?.unique_requests === "number" ? record.summary.unique_requests : 0,
            last_event_at: typeof record.summary?.last_event_at === "string" ? record.summary.last_event_at : undefined,
            action_breakdown: normalizeAuditBuckets(record.summary?.action_breakdown),
            status_breakdown: normalizeAuditBuckets(record.summary?.status_breakdown),
        },
        page_info: normalizeAuditPageInfo(record.page_info, items.length),
    };
}

function normalizeProviderAuditQueryRecord(record: ProviderAuditQueryApiRecord): ProviderAuditQueryResult {
    const items = Array.isArray(record.items) ? record.items.map(normalizeProviderAuditEvent) : [];

    return {
        items,
        summary: {
            total_events: typeof record.summary?.total_events === "number" ? record.summary.total_events : items.length,
            unique_providers: typeof record.summary?.unique_providers === "number" ? record.summary.unique_providers : 0,
            unique_requests: typeof record.summary?.unique_requests === "number" ? record.summary.unique_requests : 0,
            last_event_at: typeof record.summary?.last_event_at === "string" ? record.summary.last_event_at : undefined,
            action_breakdown: normalizeAuditBuckets(record.summary?.action_breakdown),
            status_breakdown: normalizeAuditBuckets(record.summary?.status_breakdown),
        },
        page_info: normalizeAuditPageInfo(record.page_info, items.length),
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

function emptyDocumentAuditQueryResult(): DocumentAuditQueryResult {
    return {
        items: [],
        summary: {
            total_events: 0,
            unique_documents: 0,
            unique_collections: 0,
            unique_requests: 0,
            action_breakdown: [],
            status_breakdown: [],
        },
        page_info: {
            limit: 0,
            returned: 0,
            has_next: false,
        },
    };
}

function emptyProviderAuditQueryResult(): ProviderAuditQueryResult {
    return {
        items: [],
        summary: {
            total_events: 0,
            unique_providers: 0,
            unique_requests: 0,
            action_breakdown: [],
            status_breakdown: [],
        },
        page_info: {
            limit: 0,
            returned: 0,
            has_next: false,
        },
    };
}

export async function fetchDocumentAuditEvents(input: {
    limit?: number;
    cursor?: string;
    query?: string;
    action?: string;
    status?: string;
    actorType?: string;
    actorId?: number;
    correlationId?: string;
    documentId?: number;
    collectionId?: number;
    createdFrom?: string;
    createdTo?: string;
    signal?: AbortSignal;
} = {}): Promise<DocumentAuditQueryResult> {
    const query = buildAuditQuery({
        limit: input.limit,
        cursor: input.cursor,
        q: input.query,
        action: input.action,
        status: input.status,
        actor_type: input.actorType,
        actor_id: input.actorId,
        correlation_id: input.correlationId,
        document_id: input.documentId,
        collection_id: input.collectionId,
        created_from: input.createdFrom,
        created_to: input.createdTo,
    });

    const response = await fetchWithAuthRedirect(`/api/documents/audit${query ? `?${query}` : ""}`, {
        method: "GET",
        cache: "no-store",
        signal: input.signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<DocumentAuditQueryApiRecord | DocumentAuditEventApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("audit.documents"));
    }

    if (!hasResponseData(payload)) {
        return emptyDocumentAuditQueryResult();
    }

    if (Array.isArray(payload.data)) {
        const items = payload.data.map(normalizeDocumentAuditEvent);
        return {
            items,
            summary: {
                total_events: items.length,
                unique_documents: 0,
                unique_collections: 0,
                unique_requests: 0,
                action_breakdown: [],
                status_breakdown: [],
            },
            page_info: normalizeAuditPageInfo(undefined, items.length),
        };
    }

    return normalizeDocumentAuditQueryRecord(payload.data);
}

export async function fetchProviderAuditEvents(input: {
    limit?: number;
    cursor?: string;
    query?: string;
    family?: string;
    providerType?: string;
    action?: string;
    status?: string;
    actorType?: string;
    actorId?: number;
    correlationId?: string;
    createdFrom?: string;
    createdTo?: string;
    signal?: AbortSignal;
} = {}): Promise<ProviderAuditQueryResult> {
    const query = buildAuditQuery({
        limit: input.limit,
        cursor: input.cursor,
        q: input.query,
        family: input.family,
        provider_type: input.providerType,
        action: input.action,
        status: input.status,
        actor_type: input.actorType,
        actor_id: input.actorId,
        correlation_id: input.correlationId,
        created_from: input.createdFrom,
        created_to: input.createdTo,
    });

    const response = await fetchWithAuthRedirect(`/api/user/providers/audit${query ? `?${query}` : ""}`, {
        method: "GET",
        cache: "no-store",
        signal: input.signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<ProviderAuditQueryApiRecord | ProviderAuditEventApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("audit.providers"));
    }

    if (!hasResponseData(payload)) {
        return emptyProviderAuditQueryResult();
    }

    if (Array.isArray(payload.data)) {
        const items = payload.data.map(normalizeProviderAuditEvent);
        return {
            items,
            summary: {
                total_events: items.length,
                unique_providers: 0,
                unique_requests: 0,
                action_breakdown: [],
                status_breakdown: [],
            },
            page_info: normalizeAuditPageInfo(undefined, items.length),
        };
    }

    return normalizeProviderAuditQueryRecord(payload.data);
}
