import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import ProviderReadinessPanel from "@/components/settings/ProviderReadinessPanel";
import { PROVIDERS } from "@/components/settings/provider-catalog";

const openAIProvider = PROVIDERS.find((provider) => provider.providerType === "OPENAI");

if (!openAIProvider) {
    throw new Error("OpenAI provider definition is required for Storybook stories.");
}

const meta = {
    title: "Settings/ProviderReadinessPanel",
    component: ProviderReadinessPanel,
    tags: ["autodocs"],
    args: {
        provider: openAIProvider,
        discoveredModels: [],
        isLoadingModels: false,
        onLoadModels: () => undefined,
    },
} satisfies Meta<typeof ProviderReadinessPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Ready: Story = {
    args: {
        report: {
            family: "LLM",
            provider_type: "OPENAI",
            status: "READY",
            checked_at: "2026-03-22T12:00:00Z",
            checks: [
                { code: "CREDENTIALS", status: "PASSED", metadata: {} },
                { code: "CONNECTION", status: "PASSED", metadata: {} },
                { code: "MODEL_DISCOVERY", status: "PASSED", metadata: { count: "3" } },
            ],
            discovered_models: ["gpt-4.1-mini", "gpt-4o-mini", "gpt-4.1"],
        },
    },
};

export const Attention: Story = {
    args: {
        report: {
            family: "LLM",
            provider_type: "OPENAI",
            status: "ATTENTION",
            checked_at: "2026-03-22T12:05:00Z",
            checks: [
                { code: "CREDENTIALS", status: "PASSED", metadata: {} },
                { code: "CONNECTION", status: "PASSED", metadata: {} },
                { code: "MODEL_DISCOVERY", status: "WARNING", metadata: {} },
            ],
            discovered_models: [],
        },
    },
};

export const IdleWithLoadedModels: Story = {
    args: {
        report: null,
        discoveredModels: ["gpt-4.1-mini", "gpt-4o-mini"],
    },
};
