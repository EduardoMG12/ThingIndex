import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TableOfContents } from "@/components/documentation/TableOfContents";
import { createTestWrapper } from "@/test/test-utils";

describe("TableOfContents", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
    });

    it("highlights the section currently visible in the documentation scroll container", async () => {
        const positions = new Map<string, number>([
            ["main-content", 0],
            ["overview", 48],
            ["architecture", 420],
            ["references", 820],
        ]);

        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function getBoundingClientRect(this: HTMLElement) {
            const top = positions.get(this.id) ?? 0;

            return {
                top,
                right: 0,
                bottom: top + 80,
                left: 0,
                width: 0,
                height: 80,
                x: 0,
                y: top,
                toJSON: () => ({}),
            };
        });

        render(
            <div id="main-content">
                <TableOfContents
                    items={[
                        { id: "overview", label: "Visão geral" },
                        { id: "architecture", label: "Arquitetura" },
                        { id: "references", label: "Referências" },
                    ]}
                />
                <section id="overview">Visão geral</section>
                <section id="architecture">Arquitetura</section>
                <section id="references">Referências</section>
            </div>,
            { wrapper: createTestWrapper("pt-BR") }
        );

        await waitFor(() => {
            expect(screen.getByRole("link", { name: "Visão geral" })).toHaveAttribute("aria-current", "location");
        });

        positions.set("overview", -260);
        positions.set("architecture", 96);
        positions.set("references", 500);

        fireEvent.scroll(document.getElementById("main-content")!);

        await waitFor(() => {
            expect(screen.getByRole("link", { name: "Arquitetura" })).toHaveAttribute("aria-current", "location");
        });
    });
});
