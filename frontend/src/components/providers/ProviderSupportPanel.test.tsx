import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ProviderSupportPanel } from "@/components/providers/ProviderSupportPanel";

describe("ProviderSupportPanel", () => {
    it("renders support tier, setup hints and operational notes", () => {
        render(
            <ProviderSupportPanel
                support={{
                    family: "LLM",
                    provider_type: "OPENAI",
                    display_name: "OpenAI",
                    support_level: "READY",
                    transport: "NATIVE_OPENAI",
                    summary: "Ready for the main runtime.",
                    suggested_setup: {
                        base_url: "https://api.openai.com/v1",
                        model: "gpt-4o-mini",
                    },
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: true,
                        supports_streaming: true,
                        supports_indexing: true,
                        supports_chat: true,
                        supports_optional_live_tests: true,
                        requires_explicit_model: false,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Direct OpenAI path."],
                }}
            />
        );

        expect(screen.getByText("Real support level")).toBeInTheDocument();
        expect(screen.getByText("Ready for main runtime")).toBeInTheDocument();
        expect(screen.getByText("Native OpenAI path")).toBeInTheDocument();
        expect(screen.getByText("https://api.openai.com/v1")).toBeInTheDocument();
        expect(screen.getByText("gpt-4o-mini")).toBeInTheDocument();
        expect(screen.getByText("Direct OpenAI path.")).toBeInTheDocument();
    });

    it("renders storage-specific capability labels for storage providers", () => {
        render(
            <ProviderSupportPanel
                support={{
                    family: "STORAGE",
                    provider_type: "GCP_STORAGE",
                    display_name: "Google Cloud Storage",
                    support_level: "READY",
                    transport: "NATIVE_GCS",
                    summary: "Ready for the main runtime.",
                    suggested_setup: {
                        project_id: "thingindex-dev",
                        bucket_name: "thingindex-docs",
                    },
                    capabilities: {
                        supports_credential_validation: true,
                        supports_connection_test: true,
                        supports_model_discovery: false,
                        supports_streaming: false,
                        supports_indexing: true,
                        supports_chat: false,
                        supports_optional_live_tests: true,
                        requires_explicit_model: false,
                        requires_explicit_base_url: false,
                        requires_openai_compatible_bridge: false,
                    },
                    notes: ["Supports service-account JSON or emulator_host."],
                }}
            />
        );

        expect(screen.getByText("Native Google Cloud Storage path")).toBeInTheDocument();
        expect(screen.getByText("Credential validation")).toBeInTheDocument();
        expect(screen.getByText("Project ID:")).toBeInTheDocument();
        expect(screen.getByText("thingindex-dev")).toBeInTheDocument();
    });
});
