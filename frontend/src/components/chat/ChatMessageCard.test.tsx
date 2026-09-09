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
