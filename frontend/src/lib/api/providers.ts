import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { apiErrorMessage } from "@/lib/api/error";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";

export type ProviderFamily = "STORAGE" | "LLM";

export type ProviderType =
    | "MINIO"
    | "AWS_S3"
    | "CLOUDFLARE_R2"
    | "DIGITALOCEAN_SPACES"
    | "WASABI"
    | "BACKBLAZE_B2"
    | "OPENAI"
    | "GOOGLE_GEMINI"
    | "DEEP_SEEK"
    | "GROK"
    | "CLAUDE"
    | "OLLAMA"
    | "OPENROUTER";

type ProviderApiRecord = {
    id: number;
    user_id: number;
    family: ProviderFamily;
    provider_type: ProviderType;
    is_enabled: boolean;
    enabled_for_indexing: boolean;
    enabled_for_chat: boolean;
    public_metadata?: Record<string, string>;
    created_at: string;
    updated_at: string;
};

type ProviderRuntimeAssignmentApiRecord = {
    family: ProviderFamily;
    activation_target: "STORAGE" | "INDEXING" | "CHAT";
    provider_type: ProviderType;
    public_metadata?: Record<string, string>;
    updated_at: string;
};

export type ProviderConfig = {
    id: number;
    user_id: number;
    family: ProviderFamily;
    provider_type: ProviderType;
    is_enabled: boolean;
    enabled_for_indexing: boolean;
    enabled_for_chat: boolean;
    public_metadata: Record<string, string>;
    created_at: string;
    updated_at: string;
};

export type ProviderRuntimeAssignment = {
    family: ProviderFamily;
    activation_target: "STORAGE" | "INDEXING" | "CHAT";
    provider_type: ProviderType;
    public_metadata: Record<string, string>;
    updated_at: string;
};

export type ProviderRuntimeOverview = {
    storage: ProviderRuntimeAssignment | null;
    indexing: ProviderRuntimeAssignment | null;
    chat: ProviderRuntimeAssignment | null;
};

type ProviderSupportCapabilitiesApiRecord = {
    supports_credential_validation?: boolean;
    supports_connection_test?: boolean;
    supports_model_discovery?: boolean;
    supports_streaming?: boolean;
    supports_indexing?: boolean;
    supports_chat?: boolean;
    supports_optional_live_tests?: boolean;
    requires_explicit_model?: boolean;
    requires_explicit_base_url?: boolean;
    requires_openai_compatible_bridge?: boolean;
};

type ProviderSupportApiRecord = {
    family: ProviderFamily;
    provider_type: ProviderType;
    display_name: string;
    support_level: "READY" | "PARTIAL" | "EXPERIMENTAL";
    transport: "NATIVE_OPENAI" | "NATIVE_OLLAMA" | "OPENAI_COMPATIBLE" | "OPENAI_COMPATIBLE_BRIDGE";
    summary: string;
    suggested_base_url?: string;
    suggested_model?: string;
    capabilities?: ProviderSupportCapabilitiesApiRecord;
    notes?: unknown;
};

export type ProviderSupportCapabilities = {
    supports_credential_validation: boolean;
    supports_connection_test: boolean;
    supports_model_discovery: boolean;
    supports_streaming: boolean;
    supports_indexing: boolean;
    supports_chat: boolean;
    supports_optional_live_tests: boolean;
    requires_explicit_model: boolean;
    requires_explicit_base_url: boolean;
    requires_openai_compatible_bridge: boolean;
};

export type ProviderSupport = {
    family: ProviderFamily;
    provider_type: ProviderType;
    display_name: string;
    support_level: "READY" | "PARTIAL" | "EXPERIMENTAL";
    transport: "NATIVE_OPENAI" | "NATIVE_OLLAMA" | "OPENAI_COMPATIBLE" | "OPENAI_COMPATIBLE_BRIDGE";
    summary: string;
    suggested_base_url: string | null;
    suggested_model: string | null;
    capabilities: ProviderSupportCapabilities;
    notes: string[];
};

type ProviderReadinessCheckApiRecord = {
    code: string;
    status: string;
    metadata?: Record<string, string>;
};

type ProviderReadinessReportApiRecord = {
    family: ProviderFamily;
    provider_type: ProviderType;
    status: "READY" | "ATTENTION" | "FAILED";
    checked_at: string;
    checks?: ProviderReadinessCheckApiRecord[];
    discovered_models?: unknown;
};

export type ProviderReadinessCheck = {
    code: string;
    status: string;
    metadata: Record<string, string>;
};

export type ProviderReadinessReport = {
    family: ProviderFamily;
    provider_type: ProviderType;
    status: "READY" | "ATTENTION" | "FAILED";
    checked_at: string;
    checks: ProviderReadinessCheck[];
    discovered_models: string[];
};

function normalizeProvider(record: ProviderApiRecord): ProviderConfig {
    return {
        id: record.id,
        user_id: record.user_id,
        family: record.family,
        provider_type: record.provider_type,
        is_enabled: record.is_enabled,
        enabled_for_indexing: record.enabled_for_indexing,
        enabled_for_chat: record.enabled_for_chat,
        public_metadata: record.public_metadata ?? {},
        created_at: record.created_at,
        updated_at: record.updated_at,
    };
}

function normalizeProviderRuntimeAssignment(record: unknown): ProviderRuntimeAssignment | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const runtimeRecord = record as Partial<ProviderRuntimeAssignmentApiRecord>;
    if (!runtimeRecord.family || !runtimeRecord.activation_target || !runtimeRecord.provider_type || !runtimeRecord.updated_at) {
        return null;
    }

    return {
        family: runtimeRecord.family,
        activation_target: runtimeRecord.activation_target,
        provider_type: runtimeRecord.provider_type,
        public_metadata: runtimeRecord.public_metadata ?? {},
        updated_at: runtimeRecord.updated_at,
    };
}

function normalizeProviderReadinessCheck(record: unknown): ProviderReadinessCheck | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const readinessRecord = record as Partial<ProviderReadinessCheckApiRecord>;
    if (!readinessRecord.code || !readinessRecord.status) {
        return null;
    }

    return {
        code: readinessRecord.code,
        status: readinessRecord.status,
        metadata: readinessRecord.metadata ?? {},
    };
}

function normalizeProviderReadinessReport(record: unknown): ProviderReadinessReport | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const readinessRecord = record as Partial<ProviderReadinessReportApiRecord>;
    if (!readinessRecord.family || !readinessRecord.provider_type || !readinessRecord.status || !readinessRecord.checked_at) {
        return null;
    }

    const checks = Array.isArray(readinessRecord.checks)
        ? readinessRecord.checks.map(normalizeProviderReadinessCheck).filter((item): item is ProviderReadinessCheck => item !== null)
        : [];

    const discovered_models = Array.isArray(readinessRecord.discovered_models)
        ? readinessRecord.discovered_models.reduce<string[]>((items, model) => {
            if (typeof model === "string" && model.trim()) {
                items.push(model.trim());
            }
            return items;
        }, [])
        : [];

    return {
        family: readinessRecord.family,
        provider_type: readinessRecord.provider_type,
        status: readinessRecord.status,
        checked_at: readinessRecord.checked_at,
        checks,
        discovered_models,
    };
}

function normalizeProviderSupport(record: unknown): ProviderSupport | null {
    if (!record || typeof record !== "object") {
        return null;
    }

    const supportRecord = record as Partial<ProviderSupportApiRecord>;
    if (!supportRecord.family || !supportRecord.provider_type || !supportRecord.display_name || !supportRecord.support_level || !supportRecord.transport || !supportRecord.summary) {
        return null;
    }

    const capabilities = supportRecord.capabilities ?? {};
    const notes = Array.isArray(supportRecord.notes)
        ? supportRecord.notes.reduce<string[]>((items, note) => {
            if (typeof note === "string" && note.trim()) {
                items.push(note.trim());
            }
            return items;
        }, [])
        : [];

    return {
        family: supportRecord.family,
        provider_type: supportRecord.provider_type,
        display_name: supportRecord.display_name,
        support_level: supportRecord.support_level,
        transport: supportRecord.transport,
        summary: supportRecord.summary,
        suggested_base_url: typeof supportRecord.suggested_base_url === "string" && supportRecord.suggested_base_url.trim()
            ? supportRecord.suggested_base_url.trim()
            : null,
        suggested_model: typeof supportRecord.suggested_model === "string" && supportRecord.suggested_model.trim()
            ? supportRecord.suggested_model.trim()
            : null,
        capabilities: {
            supports_credential_validation: Boolean(capabilities.supports_credential_validation),
            supports_connection_test: Boolean(capabilities.supports_connection_test),
            supports_model_discovery: Boolean(capabilities.supports_model_discovery),
            supports_streaming: Boolean(capabilities.supports_streaming),
            supports_indexing: Boolean(capabilities.supports_indexing),
            supports_chat: Boolean(capabilities.supports_chat),
            supports_optional_live_tests: Boolean(capabilities.supports_optional_live_tests),
            requires_explicit_model: Boolean(capabilities.requires_explicit_model),
            requires_explicit_base_url: Boolean(capabilities.requires_explicit_base_url),
            requires_openai_compatible_bridge: Boolean(capabilities.requires_openai_compatible_bridge),
        },
        notes,
    };
}

export async function fetchProviders(signal?: AbortSignal): Promise<ProviderConfig[]> {
    const response = await fetchWithAuthRedirect("/api/user/providers", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<ProviderApiRecord[]> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("providers.list"));
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeProvider);
}

export async function fetchProviderRuntimeOverview(signal?: AbortSignal): Promise<ProviderRuntimeOverview> {
    const response = await fetchWithAuthRedirect("/api/user/providers/runtime", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as
        | ApiSuccessResponse<{
            storage?: unknown;
            indexing?: unknown;
            chat?: unknown;
        }>
        | ApiErrorResponse
        | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("providers.runtime"));
    }

    if (!hasResponseData(payload) || !payload.data || typeof payload.data !== "object") {
        return {
            storage: null,
            indexing: null,
            chat: null,
        };
    }

    return {
        storage: normalizeProviderRuntimeAssignment(payload.data.storage),
        indexing: normalizeProviderRuntimeAssignment(payload.data.indexing),
        chat: normalizeProviderRuntimeAssignment(payload.data.chat),
    };
}

export async function fetchProviderSupportMatrix(
    options?: {
        family?: ProviderFamily;
        signal?: AbortSignal;
    }
): Promise<ProviderSupport[]> {
    const searchParams = new URLSearchParams();
    if (options?.family) {
        searchParams.set("family", options.family);
    }

    const query = searchParams.toString();
    const response = await fetch(`/api/providers/support${query ? `?${query}` : ""}`, {
        method: "GET",
        cache: "no-store",
        signal: options?.signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Could not load the provider support matrix.");
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeProviderSupport).filter((item): item is ProviderSupport => item !== null);
}

export async function saveProvider(input: {
    family: ProviderFamily;
    providerType: ProviderType;
    credentials: Record<string, string>;
    isEnabled?: boolean;
    enabledForIndexing?: boolean;
    enabledForChat?: boolean;
}): Promise<void> {
    const response = await fetchWithAuthRedirect("/api/user/providers", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            family: input.family,
            provider_type: input.providerType,
            credentials: input.credentials,
            is_enabled: input.isEnabled,
            enabled_for_indexing: input.enabledForIndexing,
            enabled_for_chat: input.enabledForChat,
        }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<null> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("providers.save"));
    }
}

export async function setProviderEnabled(input: {
    family: ProviderFamily;
    providerType: ProviderType;
    activationTarget: "STORAGE" | "INDEXING" | "CHAT";
    enabled: boolean;
}): Promise<void> {
    const response = await fetchWithAuthRedirect(`/api/user/providers/${input.providerType}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            provider_type: input.providerType,
            family: input.family,
            activation_target: input.activationTarget,
            enabled: input.enabled,
            is_enabled: input.activationTarget === "STORAGE" ? input.enabled : undefined,
            enabled_for_indexing: input.activationTarget === "INDEXING" ? input.enabled : undefined,
            enabled_for_chat: input.activationTarget === "CHAT" ? input.enabled : undefined,
        }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<null> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("providers.update"));
    }
}

export async function testProviderConnection(input: {
    family: ProviderFamily;
    providerType: ProviderType;
    credentials: Record<string, string>;
}): Promise<ProviderReadinessReport | null> {
    const response = await fetchWithAuthRedirect("/api/user/providers/test", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            family: input.family,
            provider_type: input.providerType,
            credentials: input.credentials,
        }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<unknown> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("providers.test"));
    }

    if (!hasResponseData(payload)) {
        return null;
    }

    return normalizeProviderReadinessReport(payload.data);
}

export async function listProviderModels(input: {
    family: ProviderFamily;
    providerType: ProviderType;
    credentials: Record<string, string>;
}): Promise<string[]> {
    const response = await fetchWithAuthRedirect("/api/user/providers/models", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            family: input.family,
            provider_type: input.providerType,
            credentials: input.credentials,
        }),
    });

    const payload = (await response.json().catch(() => null)) as
        | ApiSuccessResponse<{ models?: unknown }>
        | ApiErrorResponse
        | null;

    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("providers.models"));
    }

    if (!hasResponseData(payload) || !payload.data || !Array.isArray(payload.data.models)) {
        return [];
    }

    return payload.data.models.reduce<string[]>((items, model) => {
        if (typeof model === "string" && model.trim()) {
            items.push(model.trim());
        }
        return items;
    }, []);
}
