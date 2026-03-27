import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import ProviderRuntimeCallout from "@/components/runtime/ProviderRuntimeCallout";
import { createTestWrapper } from "@/test/test-utils";

describe("ProviderRuntimeCallout", () => {
    it("renders the active provider and model for the selected role", () => {
        render(
            <ProviderRuntimeCallout
                role="chat"
                assignment={{
                    family: "LLM",
                    activation_target: "CHAT",
                    provider_type: "OPENAI",
                    public_metadata: {
                        model: "gpt-4o-mini",
                    },
                    updated_at: "2026-03-20T12:00:00Z",
                }}
                isLoading={false}
                error={null}
            />,
            { wrapper: createTestWrapper("en") }
        );

        expect(screen.getByRole("heading", { name: "Active chat LLM" })).toBeInTheDocument();
        expect(screen.getByText("OpenAI")).toBeInTheDocument();
        expect(screen.getByText("gpt-4o-mini")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Review in settings/i })).toHaveAttribute("href", "/en/settings");
    });

    it("renders a clear missing-state warning when no provider is active", () => {
        render(
            <ProviderRuntimeCallout
                role="indexing"
                assignment={null}
                isLoading={false}
                error={null}
            />,
            { wrapper: createTestWrapper("en") }
        );

        expect(screen.getByRole("heading", { name: "Active indexing LLM" })).toBeInTheDocument();
        expect(screen.getByText(/No LLM is active for indexing/i)).toBeInTheDocument();
    });
});
