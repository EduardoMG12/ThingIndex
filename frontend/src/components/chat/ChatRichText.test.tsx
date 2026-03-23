import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ChatRichText } from "@/components/chat/ChatRichText";

describe("ChatRichText", () => {
    it("renders bullet, numbered, and quoted blocks", () => {
        render(
            <ChatRichText
                content={[
                    "## Findings",
                    "",
                    "- First point",
                    "- Second point",
                    "",
                    "1. Step one",
                    "2. Step two",
                    "",
                    "> quoted evidence",
                ].join("\n")}
            />
        );

        expect(screen.getByRole("heading", { level: 4, name: "Findings" })).toBeInTheDocument();
        expect(screen.getAllByRole("list")).toHaveLength(2);
        expect(screen.getByText("First point")).toBeInTheDocument();
        expect(screen.getByText("Step two")).toBeInTheDocument();
        expect(screen.getByText("quoted evidence")).toBeInTheDocument();
    });

    it("preserves bold inline segments inside paragraphs", () => {
        render(<ChatRichText content={"Use **clause 7.2** before filing."} />);

        expect(screen.getByText("clause 7.2")).toBeInTheDocument();
        expect(screen.getByText("clause 7.2").tagName).toBe("STRONG");
    });
});
