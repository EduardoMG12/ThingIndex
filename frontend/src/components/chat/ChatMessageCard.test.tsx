import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ChatMessageCard } from "@/components/chat/ChatMessageCard";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const base = { id: "m1", role: "assistant" as const, content: "O prazo e de 30 dias.", createdAt: "2026-01-01T12:00:00Z", sources: [] };

describe("ChatMessageCard: consumo da resposta", () => {
    it("mostra tokens, modelo e custo quando o provedor informou", () => {
        render(
            <ChatMessageCard isStreaming={false} message={{ ...base, usage: { promptTokens: 1234, completionTokens: 56, usageReported: true, model: "gemma4:31b", estimatedCostUsd: 0.00027 } }} />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        const linha = screen.getByTestId("consumo-da-resposta");
        expect(linha).toHaveTextContent("1.290 tokens · gemma4:31b · US$ 0,00027");
        expect(linha).not.toHaveTextContent("≈");
        expect(linha).not.toHaveAttribute("title");
    });

    it("marca a estimativa com ≈ e explica no title; sem custo, sem valor", () => {
        render(
            <ChatMessageCard isStreaming={false} message={{ ...base, usage: { promptTokens: 100, completionTokens: 7, usageReported: false, model: "llama3" } }} />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        const linha = screen.getByTestId("consumo-da-resposta");
        expect(linha).toHaveTextContent("≈ 107 tokens · llama3");
        expect(linha).not.toHaveTextContent("US$");
        expect(linha).toHaveAttribute("title", "Estimativa: o provedor não informou o consumo.");
    });

    it("sem usage nao renderiza a linha", () => {
        render(<ChatMessageCard isStreaming={false} message={base} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.queryByTestId("consumo-da-resposta")).toBeNull();
    });
});

describe("ChatMessageCard: fontes navegaveis", () => {
    const fontes = [
        { id: "265:1:table-9-row-1", docName: "Matriz", page: 1, snippet: "Disciplina: SIS07302", documentId: 265 },
        { id: "260:126", docName: "PPC", page: 126, snippet: "conceito D", documentId: 260, paragraphId: 72 },
    ];

    it("numera as fontes, e o clique e as setas percorrem a lista marcando a ativa", async () => {
        const { fireEvent } = await import("@testing-library/react");
        const ver = vi.fn();
        render(<ChatMessageCard isStreaming={false} message={{ ...base, sources: fontes }} onVerNoDocumento={ver} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.getByTestId("fonte-posicao")).toHaveTextContent("–/2");
        expect(screen.getByTestId("fonte-anterior")).toBeDisabled();

        fireEvent.click(screen.getByTestId("fonte-proxima"));
        expect(ver).toHaveBeenLastCalledWith(expect.objectContaining({ id: "265:1:table-9-row-1" }));
        expect(screen.getByTestId("fonte-posicao")).toHaveTextContent("1/2");
        expect(screen.getAllByTestId("fonte")[0]).toHaveAttribute("data-ativa", "true");

        fireEvent.click(screen.getByTestId("fonte-proxima"));
        expect(ver).toHaveBeenLastCalledWith(expect.objectContaining({ paragraphId: 72 }));
        expect(screen.getByTestId("fonte-posicao")).toHaveTextContent("2/2");
        expect(screen.getByTestId("fonte-proxima")).toBeDisabled();

        // Clicar direto na fonte 1 volta a ativa para ela.
        fireEvent.click(screen.getAllByTestId("ver-no-documento")[0]);
        expect(screen.getByTestId("fonte-posicao")).toHaveTextContent("1/2");
    });

    it("sem onVerNoDocumento nao mostra as setas", () => {
        render(<ChatMessageCard isStreaming={false} message={{ ...base, sources: fontes }} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.queryByTestId("fonte-proxima")).toBeNull();
        expect(screen.getAllByTestId("fonte")).toHaveLength(2);
    });
});

describe("ChatMessageCard: fonte de indice antigo", () => {
    it("avisa quando o documento foi reindexado depois da resposta", () => {
        const fonte = { id: "260:126", docName: "PPC", page: 126, snippet: "x", documentId: 260, paragraphId: 72, indexedAt: "2026-09-01T10:00:00Z" };
        const indexadoEm = new Map([[260, "2026-09-09T10:00:00Z"]]);
        render(<ChatMessageCard isStreaming={false} message={{ ...base, sources: [fonte] }} indexadoEm={indexadoEm} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.getByTestId("fonte-reindexada")).toHaveTextContent("reindexado");
    });

    it("sem reindexacao nao avisa", () => {
        const fonte = { id: "260:126", docName: "PPC", page: 126, snippet: "x", documentId: 260, indexedAt: "2026-09-09T10:00:00Z" };
        render(<ChatMessageCard isStreaming={false} message={{ ...base, sources: [fonte] }} indexadoEm={new Map([[260, "2026-09-09T10:00:00Z"]])} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.queryByTestId("fonte-reindexada")).toBeNull();
    });
});

describe("ChatMessageCard: resposta com erro", () => {
    it("mostra o aviso com o detalhe do provedor e mantem as fontes", () => {
        const fonte = { id: "260:126", docName: "PPC", page: 126, snippet: "x", documentId: 260 };
        render(<ChatMessageCard isStreaming={false} message={{ ...base, content: "", status: "error", errorDetail: "monthly usage limit", sources: [fonte] }} />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.getByTestId("resposta-com-erro")).toHaveTextContent("A resposta falhou");
        expect(screen.getByTestId("resposta-com-erro")).toHaveTextContent("monthly usage limit");
        expect(screen.getAllByTestId("fonte")).toHaveLength(1);
    });
});

describe("ChatMessageCard: grade de fontes no painel estreito", () => {
    it("a grade cai para uma coluna pela largura do conteiner, nao da janela", () => {
        const fontes = [1, 2].map((n) => ({ id: `1:${n}`, docName: `Doc ${n}`, page: n, snippet: "x" }));
        render(<ChatMessageCard isStreaming={false} message={{ ...base, sources: fontes }} />, {
            wrapper: createTestWrapper("pt-BR"),
        });

        const grade = screen.getAllByTestId("fonte")[0]!.parentElement!;
        expect(grade).toHaveClass("grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]");
        expect(grade).not.toHaveClass("sm:grid-cols-2");
    });
});
