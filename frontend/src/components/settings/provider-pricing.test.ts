import { describe, expect, it } from "vitest";

import { getInitialFormState, getLocalizedProviders } from "@/components/settings/provider-catalog";
import {
    applySavedPricing,
    getInvalidPricingFields,
    parsePricingInput,
} from "@/components/settings/provider-pricing";
import type { ProviderConfig } from "@/lib/api/providers";

function llmConfig(overrides: Partial<ProviderConfig>): ProviderConfig {
    return {
        id: 1,
        user_id: 1,
        family: "LLM",
        provider_type: "OLLAMA",
        is_enabled: false,
        enabled_for_indexing: true,
        enabled_for_chat: true,
        public_metadata: {},
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

describe("provider-pricing", () => {
    it("aceita ponto ou virgula e trata vazio como sem preco", () => {
        expect(parsePricingInput("")).toEqual({ valid: true, normalized: "" });
        expect(parsePricingInput(" 0,15 ")).toEqual({ valid: true, normalized: "0.15" });
        expect(parsePricingInput("2.5")).toEqual({ valid: true, normalized: "2.5" });
        expect(parsePricingInput("0")).toEqual({ valid: true, normalized: "0" });
    });

    it("rejeita negativo e texto", () => {
        expect(parsePricingInput("-1").valid).toBe(false);
        expect(parsePricingInput("abc").valid).toBe(false);
        expect(parsePricingInput("1,5,3").valid).toBe(false);
        expect(getInvalidPricingFields({ input_cost_per_million_usd: "-1", output_cost_per_million_usd: "" })).toEqual([
            "input_cost_per_million_usd",
        ]);
    });

    it("mostra os campos de preco em portugues nos provedores de LLM", () => {
        const ollama = getLocalizedProviders("pt-BR").find((provider) => provider.providerType === "OLLAMA");
        const labels = ollama?.fields.filter((field) => field.kind === "price").map((field) => field.label);

        expect(labels).toEqual([
            "Preço por 1 milhão de tokens de entrada (US$)",
            "Preço por 1 milhão de tokens de saída (US$)",
        ]);
        expect(ollama?.fields.find((field) => field.kind === "price")?.optional).toBe(true);
        const minio = getLocalizedProviders("pt-BR").find((provider) => provider.providerType === "MINIO");
        expect(minio?.fields.some((field) => field.kind === "price")).toBe(false);
    });

    it("preenche o preco salvo sem sobrescrever o que o usuario digitou", () => {
        const forms = getInitialFormState();
        forms.OPENAI.output_cost_per_million_usd = "9";

        const next = applySavedPricing(forms, [
            llmConfig({
                public_metadata: { input_cost_per_million_usd: "0.2", output_cost_per_million_usd: "0.8" },
            }),
            llmConfig({
                provider_type: "OPENAI",
                public_metadata: { input_cost_per_million_usd: "1", output_cost_per_million_usd: "2" },
            }),
        ]);

        expect(next.OLLAMA.input_cost_per_million_usd).toBe("0.2");
        expect(next.OLLAMA.output_cost_per_million_usd).toBe("0.8");
        expect(next.OLLAMA.base_url).toBe("https://ollama.com/api");
        expect(next.OPENAI.input_cost_per_million_usd).toBe("1");
        expect(next.OPENAI.output_cost_per_million_usd).toBe("9");
    });

    it("devolve o mesmo estado quando nao ha preco salvo", () => {
        const forms = getInitialFormState();
        expect(applySavedPricing(forms, [llmConfig({})])).toBe(forms);
    });
});
