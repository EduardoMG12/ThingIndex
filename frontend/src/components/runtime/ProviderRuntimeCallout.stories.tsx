import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import ProviderRuntimeCallout from "@/components/runtime/ProviderRuntimeCallout";

const meta = {
    title: "Runtime/ProviderRuntimeCallout",
    component: ProviderRuntimeCallout,
    tags: ["autodocs"],
    parameters: {
        nextjs: {
            appDirectory: true,
        },
    },
    args: {
        role: "chat",
        isLoading: false,
        error: null,
    },
} satisfies Meta<typeof ProviderRuntimeCallout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const ChatActiveRuntime: Story = {
    args: {
        role: "chat",
        assignment: {
            family: "LLM",
            activation_target: "CHAT",
            provider_type: "OPENAI",
            public_metadata: {
                model: "gpt-4.1-mini",
            },
            updated_at: "2026-03-22T12:30:00Z",
        },
    },
};

export const IndexingRuntimeMissing: Story = {
    args: {
        role: "indexing",
        assignment: null,
    },
};

export const LoadingState: Story = {
    args: {
        assignment: null,
        isLoading: true,
    },
};

export const ErrorState: Story = {
    args: {
        assignment: null,
        error: "Could not load the active provider runtime.",
    },
};
