import type { ProviderConfig, ProviderType } from "@/lib/api/providers";

import { LLM_PRICING_FIELD_KEYS } from "@/components/settings/provider-catalog";

export type LLMPricingFieldKey = (typeof LLM_PRICING_FIELD_KEYS)[number];

export function isLLMPricingField(key: string): key is LLMPricingFieldKey {
    return (LLM_PRICING_FIELD_KEYS as readonly string[]).includes(key);
}

/**
 * Le o preco digitado no formulario. Aceita ponto ou virgula decimal;
 * vazio significa "sem preco" e nao e erro. Devolve o valor normalizado com
 * ponto, que e o formato que o backend grava nas credenciais.
 */
export function parsePricingInput(raw: string): { valid: boolean; normalized: string } {
    const trimmed = raw.trim();
    if (!trimmed) {
        return { valid: true, normalized: "" };
    }

    const normalized = trimmed.replace(",", ".");
    if (!/^(\d+(\.\d*)?|\.\d+)$/.test(normalized)) {
        return { valid: false, normalized: trimmed };
    }

    const value = Number(normalized);
    if (!Number.isFinite(value) || value < 0) {
        return { valid: false, normalized: trimmed };
    }

    return { valid: true, normalized };
}

/** Chaves de preco com valor invalido no formulario do provedor. */
export function getInvalidPricingFields(form: Record<string, string> | undefined): LLMPricingFieldKey[] {
    return LLM_PRICING_FIELD_KEYS.filter((key) => !parsePricingInput(form?.[key] ?? "").valid);
}

/**
 * Preenche o formulario com o preco salvo (vem no public_metadata, pois nao e
 * segredo). So preenche campo vazio, para nao apagar o que o usuario digitou.
 */
export function applySavedPricing(
    forms: Record<ProviderType, Record<string, string>>,
    configs: ProviderConfig[],
): Record<ProviderType, Record<string, string>> {
    let changed = false;
    const next = { ...forms };

    for (const config of configs) {
        if (config.family !== "LLM") {
            continue;
        }

        for (const key of LLM_PRICING_FIELD_KEYS) {
            const saved = config.public_metadata?.[key]?.trim();
            const current = next[config.provider_type]?.[key] ?? "";
            if (!saved || current.trim()) {
                continue;
            }

            next[config.provider_type] = {
                ...next[config.provider_type],
                [key]: saved,
            };
            changed = true;
        }
    }

    return changed ? next : forms;
}
