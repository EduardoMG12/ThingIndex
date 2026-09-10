import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import SourceReference from "@/components/chat/SourceReference";
import { createTestWrapper } from "@/test/test-utils";

const fonte = {
    id: "case-1",
    docName: "Master Service Agreement",
    page: 4,
    snippet: "Invoices must be paid within 15 days.",
};

describe("SourceReference", () => {
    it("reveals the snippet when the source card is expanded", async () => {
        const user = userEvent.setup();

        render(<SourceReference source={fonte} />, { wrapper: createTestWrapper("en") });

        expect(screen.getByText(/Invoices must be paid within 15 days/i)).not.toBeVisible();

        await user.click(screen.getByText("Master Service Agreement"));

        expect(screen.getByText(/Invoices must be paid within 15 days/i)).toBeInTheDocument();
        expect(screen.getByText("Verified source")).toBeInTheDocument();
    });

    it("so mostra 'Ver no documento' quando a fonte tem documentId e ha callback; o clique nao abre o card", async () => {
        const user = userEvent.setup();
        const onVer = vi.fn();
        const { rerender } = render(<SourceReference source={fonte} onVerNoDocumento={onVer} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.queryByTestId("ver-no-documento")).toBeNull();

        rerender(<SourceReference source={{ ...fonte, documentId: 42, nodeId: "n-1" }} onVerNoDocumento={onVer} />);
        await user.click(screen.getByTestId("ver-no-documento"));
        expect(onVer).toHaveBeenCalledWith(expect.objectContaining({ documentId: 42, nodeId: "n-1", page: 4 }));
        expect(screen.getByText(/Invoices must be paid/i)).not.toBeVisible();

        rerender(<SourceReference source={{ ...fonte, documentId: 42 }} />);
        expect(screen.queryByTestId("ver-no-documento")).toBeNull();
    });
});
