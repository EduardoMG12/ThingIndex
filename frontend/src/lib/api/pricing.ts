import {
    hasResponseData,
    type ApiErrorResponse,
    type ApiSuccessResponse,
} from "@/lib/api/types";
import { apiErrorMessage } from "@/lib/api/error";
import { fetchWithAuthRedirect } from "@/lib/api/fetch-with-auth-redirect";

export type PricingProviderType = "OPENAI" | "GOOGLE_GEMINI" | "DEEP_SEEK" | "OLLAMA";

type PricingSourceApiRecord = {
    id: number;
    user_id: number;
    provider_type: PricingProviderType;
    name: string;
    source_url: string;
    is_enabled: boolean;
    last_synced_at?: string | null;
    last_sync_status?: string;
    last_sync_error?: string;
    last_sync_duration_ms?: number;
    last_price_count?: number;
    created_at: string;
    updated_at: string;
};

type ModelPriceApiRecord = {
    id: number;
    pricing_source_id: number;
    provider_type: PricingProviderType;
    model_key: string;
    model_name: string;
    unit: string;
    currency: string;
    flat_price_usd: number;
    input_cost_per_million_usd: number;
    output_cost_per_million_usd: number;
    cached_input_cost_per_million_usd: number;
    context_cost_per_million_usd: number;
    price_notes?: string;
    last_synced_at: string;
};

type ModelPriceHistoryApiRecord = {
    id: number;
    pricing_source_id: number;
    provider_type: PricingProviderType;
    model_key: string;
    model_name: string;
    unit: string;
    currency: string;
    flat_price_usd: number;
    input_cost_per_million_usd: number;
    output_cost_per_million_usd: number;
    cached_input_cost_per_million_usd: number;
    context_cost_per_million_usd: number;
    price_notes?: string;
    captured_at: string;
};

export type PricingSource = PricingSourceApiRecord;
export type ModelPrice = ModelPriceApiRecord;
export type ModelPriceHistory = ModelPriceHistoryApiRecord;

export async function fetchPricingSources(signal?: AbortSignal): Promise<PricingSource[]> {
    const response = await fetchWithAuthRedirect("/api/user/pricing-sources", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<PricingSourceApiRecord[]> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("pricing.sources"));
    }
    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }
    return payload.data;
}

export async function savePricingSource(input: {
    providerType: PricingProviderType;
    name: string;
    sourceURL: string;
    isEnabled?: boolean;
}): Promise<void> {
    const response = await fetchWithAuthRedirect("/api/user/pricing-sources", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            provider_type: input.providerType,
            name: input.name,
            source_url: input.sourceURL,
            is_enabled: input.isEnabled,
        }),
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<null> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("pricing.saveSource"));
    }
}

export async function fetchPrices(signal?: AbortSignal): Promise<ModelPrice[]> {
    const response = await fetchWithAuthRedirect("/api/user/prices", {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<ModelPriceApiRecord[]> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("pricing.list"));
    }
    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }
    return payload.data;
}

export async function fetchPriceHistory(limit = 100, signal?: AbortSignal): Promise<ModelPriceHistory[]> {
    const response = await fetchWithAuthRedirect(`/api/user/prices/history?limit=${limit}`, {
        method: "GET",
        cache: "no-store",
        signal,
    });

    const payload = (await response.json().catch(() => null)) as ApiSuccessResponse<ModelPriceHistoryApiRecord[]> | ApiErrorResponse | null;
    if (!response.ok) {
        throw new Error(payload?.message || apiErrorMessage("pricing.history"));
    }
    if (!hasResponseData(payload) || !Array.isArray(payload.data)) {
        return [];
    }
    return payload.data;
}
