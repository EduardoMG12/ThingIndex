import { beforeEach, describe, expect, it, vi } from "vitest";

import { fetchRuntimeProfile } from "@/lib/api/runtime-profile";

describe("runtime profile api", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
    });

    it("loads and normalizes the public runtime profile", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response(
                JSON.stringify({
                    status: 200,
                    type: "success",
                    message: "ok",
                    data: {
                        product_name: "ThingIndex",
                        tagline: "Relational document intelligence for auditable operational knowledge.",
                        primary_use_case: "Organize collections with auditable context.",
                        core_problem: "Disconnected document tooling breaks provenance.",
                        value_proposition: "Relational-first context recovery.",
                        target_users: [
                            { name: "Operations and compliance teams", description: "Need provenance." },
                            { name: "Academic and research environments", description: "Need reproducibility." },
                            { name: "", description: "skip" },
                        ],
                        current_scope: ["Document workspace", " ", null],
                        differentiators: ["Relational-first"],
                        non_goals: ["No generic copilot"],
                        research_tracks: ["Vector benchmark"],
                        artifact_refs: [
                            {
                                label: "Product focus RFC",
                                path: "rfc/0001-product-focus.md",
                                kind: "RFC",
                                summary: "Backlog alignment.",
                            },
                            {
                                label: "",
                                path: "skip",
                                kind: "RFC",
                                summary: "skip",
                            },
                        ],
                        last_reviewed_on: "2026-03-22",
                    },
                }),
                { status: 200, headers: { "Content-Type": "application/json" } }
            )
        );

        const profile = await fetchRuntimeProfile();

        expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe("/api/runtime/profile");
        expect(profile).toEqual({
            product_name: "ThingIndex",
            tagline: "Relational document intelligence for auditable operational knowledge.",
            primary_use_case: "Organize collections with auditable context.",
            core_problem: "Disconnected document tooling breaks provenance.",
            value_proposition: "Relational-first context recovery.",
            target_users: [
                { name: "Operations and compliance teams", description: "Need provenance." },
                { name: "Academic and research environments", description: "Need reproducibility." },
            ],
            current_scope: ["Document workspace"],
            differentiators: ["Relational-first"],
            non_goals: ["No generic copilot"],
            research_tracks: ["Vector benchmark"],
            artifact_refs: [
                {
                    label: "Product focus RFC",
                    path: "rfc/0001-product-focus.md",
                    kind: "RFC",
                    summary: "Backlog alignment.",
                },
            ],
            last_reviewed_on: "2026-03-22",
        });
    });
});
