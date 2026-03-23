import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import SourceReference from "@/components/chat/SourceReference";
import { createTestWrapper } from "@/test/test-utils";

describe("SourceReference", () => {
    it("reveals the snippet when the source card is expanded", async () => {
        const user = userEvent.setup();

        render(
            <SourceReference
                source={{
                    id: "case-1",
                    docName: "Master Service Agreement",
                    page: 4,
                    snippet: "Invoices must be paid within 15 days.",
                }}
            />,
            { wrapper: createTestWrapper("en") }
        );

        expect(screen.getByText(/Invoices must be paid within 15 days/i)).not.toBeVisible();

        await user.click(screen.getByText("Master Service Agreement"));

        expect(screen.getByText(/Invoices must be paid within 15 days/i)).toBeInTheDocument();
        expect(screen.getByText("Verified source")).toBeInTheDocument();
    });
});
