import { Bot, Database } from "lucide-react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import ProviderReadinessPanel from "@/components/settings/ProviderReadinessPanel";
import { createTestWrapper } from "@/test/test-utils";

describe("ProviderReadinessPanel", () => {
    it("renders the readiness report and discovered models", () => {
        const onLoadModels = vi.fn();

        render(
            <ProviderReadinessPanel
                provider={{
                    family: "LLM",
                    providerType: "OPENAI",
                    name: "OpenAI",
                    tagline: "Default OpenAI-compatible setup",
                    description: "desc",
                    icon: Bot,
                    accentClass: "",
                    fields: [],
                }}
                report={{
                    family: "LLM",
                    provider_type: "OPENAI",
                    status: "READY",
                    checked_at: "2026-03-22T12:00:00Z",
                    checks: [
                        { code: "CREDENTIALS", status: "PASSED", metadata: {} },
                        { code: "CONNECTION", status: "PASSED", metadata: {} },
                        { code: "MODEL_DISCOVERY", status: "PASSED", metadata: {} },
                    ],
                    discovered_models: ["gpt-4o-mini", "gpt-4.1-mini"],
                }}
                discoveredModels={[]}
                isLoadingModels={false}
                onLoadModels={onLoadModels}
            />,
            { wrapper: createTestWrapper("en") }
        );

        expect(screen.getByText("Provider readiness")).toBeInTheDocument();
        expect(screen.getByTestId("provider-brand-OPENAI")).toBeInTheDocument();
        expect(screen.getByText("Ready")).toBeInTheDocument();
        expect(screen.getByText(/Validated live connectivity and model discovery/i)).toBeInTheDocument();
        expect(screen.getByText("gpt-4o-mini")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Load models/i })).toBeInTheDocument();
    });

    it("renders an idle state when no readiness report exists yet", () => {
        render(
            <ProviderReadinessPanel
                provider={{
                    family: "STORAGE",
                    providerType: "MINIO",
                    name: "MinIO / S3-compatible",
                    tagline: "Object storage",
                    description: "desc",
                    icon: Database,
                    accentClass: "",
                    fields: [],
                }}
                report={null}
                discoveredModels={[]}
                isLoadingModels={false}
                onLoadModels={vi.fn()}
            />,
            { wrapper: createTestWrapper("en") }
        );

        expect(screen.getByTestId("provider-brand-MINIO")).toBeInTheDocument();
        expect(screen.getByText("Not tested yet")).toBeInTheDocument();
        expect(screen.getByText(/Run the connection test/i)).toBeInTheDocument();
    });
});
