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

describe("SourceReference: termos casados", () => {
    it("mostra os termos da pergunta que a fonte contem", async () => {
        const { render, screen } = await import("@testing-library/react");
        const { createTestWrapper } = await import("@/test/test-utils");
        const SourceReference = (await import("@/components/chat/SourceReference")).default;
        render(<SourceReference source={{ id: "1:2", docName: "PPC", page: 2, snippet: "x", matchedTerms: ["estagio", "horas"] }} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.getByTestId("termos-da-fonte")).toHaveTextContent("estagio");
        expect(screen.getByTestId("termos-da-fonte")).toHaveTextContent("horas");
    });
});

it("fonte cuja frase quebra na virada mostra as duas paginas", () => {
    render(<SourceReference source={{ id: "1:186", docName: "PPC", page: 186, snippet: "x", continuesOnPage: 187 }} />, { wrapper: createTestWrapper("pt-BR") });
    expect(screen.getByTestId("fonte")).toHaveTextContent("186–187");
});

describe("SourceReference legivel em painel estreito", () => {
    const nomeLongo = "Normas Complementares de Estagio Supervisionado do Curso de Sistemas de Informacao 2026.pdf";

    it("o nome do documento fica numa linha com reticencias e o nome completo acessivel no title", () => {
        render(<SourceReference source={{ id: "1:4", docName: nomeLongo, page: 4, snippet: "x", continuesOnPage: 5 }} />, {
            wrapper: createTestWrapper("pt-BR"),
        });

        const nome = screen.getByText(nomeLongo);
        expect(nome).toHaveAttribute("title", nomeLongo);
        expect(nome).toHaveClass("truncate");
        expect(nome.closest("div")).toHaveClass("min-w-0");
    });

    it("a pagina com continuacao vira um unico texto que nao quebra", () => {
        render(<SourceReference source={{ id: "1:4", docName: nomeLongo, page: 4, snippet: "x", continuesOnPage: 5 }} />, {
            wrapper: createTestWrapper("pt-BR"),
        });

        // O locale vem da rota (mock global em /en), entao o rotulo e "Page".
        const pagina = screen.getByText("Page 4–5");
        expect(pagina.childNodes).toHaveLength(1);
        expect(pagina).toHaveClass("whitespace-nowrap");
    });
});
