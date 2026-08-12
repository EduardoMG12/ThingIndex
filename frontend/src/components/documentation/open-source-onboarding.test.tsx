import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OpenSourceOnboardingSection } from "@/components/documentation/OpenSourceOnboardingSection";
import {
    getDocumentationResources,
    getDocumentationTracks,
    resolveApiReferenceBase,
    resolveHostedAPIBase,
} from "@/components/documentation/open-source-onboarding";
import { createTestWrapper } from "@/test/test-utils";

describe("documentation open-source onboarding helpers", () => {
    it("normalizes the public api base for docs endpoints", () => {
        expect(resolveApiReferenceBase("http://127.0.0.1:8000/api/v1")).toBe("http://127.0.0.1:8000");
        expect(resolveApiReferenceBase("http://localhost:8000/api/v1/")).toBe("http://localhost:8000");
        expect(resolveApiReferenceBase()).toBe("http://localhost:8000");
        expect(resolveHostedAPIBase("https://api.thingindex.dev/api/v1")).toBe("https://api.thingindex.dev/api/v1");
    });

    it("builds documentation resources with local api docs links", () => {
        const resources = getDocumentationResources("http://127.0.0.1:8000/api/v1");

        expect(resources).toEqual(
            expect.arrayContaining([
                expect.objectContaining({ title: "Scalar API reference", href: "http://127.0.0.1:8000/docs" }),
                expect.objectContaining({ title: "Swagger UI", href: "http://127.0.0.1:8000/swagger/index.html" }),
                expect.objectContaining({ title: "Storage support matrix", href: "http://127.0.0.1:8000/api/v1/providers/support?family=STORAGE" }),
                expect.objectContaining({ pathHint: "examples/http/thingindex-api.http" }),
                expect.objectContaining({ pathHint: "api/cmd/index-import" }),
                expect.objectContaining({ pathHint: "api/docs/swagger.yaml" }),
            ])
        );
    });

    it("builds separate tracks for self-hosted and hosted api consumption", () => {
        const tracks = getDocumentationTracks("https://api.thingindex.dev/api/v1");

        expect(tracks).toEqual(
            expect.arrayContaining([
                expect.objectContaining({ title: "Self-hosted runtime" }),
                expect.objectContaining({
                    title: "Hosted API consumption",
                    steps: expect.arrayContaining([
                        expect.objectContaining({ command: "https://api.thingindex.dev/docs" }),
                        expect.objectContaining({ command: "https://api.thingindex.dev/api/v1" }),
                    ]),
                }),
            ])
        );
    });

    it("renders the open-source onboarding section with both entry paths and references", () => {
        render(<OpenSourceOnboardingSection apiBaseUrl="http://127.0.0.1:8000/api/v1" />, {
            wrapper: createTestWrapper(),
        });

        expect(screen.getAllByRole("link", { name: /Open link/i })).toHaveLength(5);
        expect(screen.getByRole("heading", { level: 3, name: "Self-hosted runtime" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 3, name: "Hosted API consumption" })).toBeInTheDocument();
        expect(screen.getByText("./scripts/bootstrap.sh")).toBeInTheDocument();
        expect(screen.getAllByText("http://127.0.0.1:8000/docs").length).toBeGreaterThan(0);
        expect(screen.getAllByText("http://127.0.0.1:8000/api/v1").length).toBeGreaterThan(0);
        expect(screen.getByText("Storage support matrix")).toBeInTheDocument();
        expect(screen.getByText("LLM support matrix")).toBeInTheDocument();
        expect(screen.getByText("docs/open-source-onboarding.md")).toBeInTheDocument();
        expect(screen.getAllByText("examples/http/thingindex-api.http").length).toBeGreaterThan(0);
        expect(screen.getByText("api/cmd/index-import")).toBeInTheDocument();
        expect(screen.getAllByText("api/docs/swagger.yaml").length).toBeGreaterThan(0);
    });
});
