import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
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
    | "CLAUDE";

type ProviderApiRecord = {
    id: number;
    user_id: number;
    family: ProviderFamily;
    provider_type: ProviderType;
    is_enabled: boolean;
    public_metadata?: Record<string, string>;
    created_at: string;
    updated_at: string;
};

export type ProviderConfig = {
    id: number;
    user_id: number;
    family: ProviderFamily;
    provider_type: ProviderType;
    is_enabled: boolean;
    public_metadata: Record<string, string>;
    created_at: string;
    updated_at: string;
};

function normalizeProvider(record: ProviderApiRecord): ProviderConfig {
    return {
        id: record.id,
        user_id: record.user_id,
        family: record.family,
        provider_type: record.provider_type,
        is_enabled: record.is_enabled,
        public_metadata: record.public_metadata ?? {},
        created_at: record.created_at,
        updated_at: record.updated_at,
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
        throw new Error(payload?.message || "Nao foi possivel carregar os providers.");
    }

    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }

    return payload.data.map(normalizeProvider);
}

export async function saveProvider(input: {
    family: ProviderFamily;
    providerType: ProviderType;
    credentials: Record<string, string>;
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
        }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<null> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel salvar o provider.");
    }
}

export async function setProviderEnabled(input: {
    family: ProviderFamily;
    providerType: ProviderType;
    isEnabled: boolean;
}): Promise<void> {
    const response = await fetchWithAuthRedirect(`/api/user/providers/${input.providerType}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            family: input.family,
            is_enabled: input.isEnabled,
        }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<null> | ApiErrorResponse | null;

    if (!response.ok) {
        throw new Error(payload?.message || "Nao foi possivel atualizar o provider.");
    }
}
