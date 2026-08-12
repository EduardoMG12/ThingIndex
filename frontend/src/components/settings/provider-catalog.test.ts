import { describe, expect, it } from "vitest";

import { getInitialFormState, getLocalizedProviders } from "@/components/settings/provider-catalog";

describe("provider-catalog", () => {
    it("prefills the Ollama base URL with the cloud API endpoint", () => {
        const formState = getInitialFormState();

        expect(formState.OLLAMA.base_url).toBe("https://ollama.com/api");
        expect(formState.OLLAMA.api_key).toBe("");
        expect(formState.OPENAI.base_url).toBe("");
    });

    it("surfaces the cloud API endpoint in the Ollama placeholder", () => {
        const ollamaProvider = getLocalizedProviders("en").find((provider) => provider.providerType === "OLLAMA");
        const baseUrlField = ollamaProvider?.fields.find((field) => field.key === "base_url");

        expect(baseUrlField?.placeholder).toContain("https://ollama.com/api");
    });
});
