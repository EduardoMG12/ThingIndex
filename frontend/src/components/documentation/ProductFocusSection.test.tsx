import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const { fetchRuntimeProfileMock } = vi.hoisted(() => ({
    fetchRuntimeProfileMock: vi.fn(),
}));

vi.mock("@/lib/api/runtime-profile", () => ({
    fetchRuntimeProfile: fetchRuntimeProfileMock,
}));

import { ProductFocusSection } from "@/components/documentation/ProductFocusSection";

describe("ProductFocusSection", () => {
    it("renders the runtime profile payload", async () => {
        fetchRuntimeProfileMock.mockResolvedValue({
            product_name: "ThingIndex",
            tagline: "Relational document intelligence for auditable operational knowledge.",
            primary_use_case: "Organize collections of operational, compliance, legal, and research documents.",
            core_problem: "Disconnected tools break provenance.",
            value_proposition: "Relational-first context recovery.",
            target_users: [
                { name: "Operations and compliance teams", description: "Need provenance." },
                { name: "Academic and research environments", description: "Need reproducibility." },
            ],
            current_scope: ["Document workspace", "Grounded chat"],
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

        render(<ProductFocusSection />);

        await waitFor(() => {
            expect(screen.getByText("ThingIndex")).toBeInTheDocument();
        });

        expect(screen.getByText(/Operations and compliance teams/i)).toBeInTheDocument();
        expect(screen.getByText("Relational-first context recovery.")).toBeInTheDocument();
        expect(screen.getByText("rfc/0001-product-focus.md")).toBeInTheDocument();
        expect(screen.getByText("/api/v1/meta/runtime-profile")).toBeInTheDocument();
    });
});
