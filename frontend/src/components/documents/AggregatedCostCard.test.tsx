import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const { pathnameState } = vi.hoisted(() => ({ pathnameState: { value: "/pt-BR/documents" } }));

vi.mock("next/navigation", () => ({
    usePathname: () => pathnameState.value,
}));

import { AggregatedCostCard } from "@/components/documents/AggregatedCostCard";
import { createTestWrapper } from "@/test/test-utils";

const ollama = {
    documents_with_llm: 12,
    pricing_aware_documents: 0,
    total_estimated_cost_usd: 0,
    total_tokens: 1753745,
    total_llm_requests: 299,
};

describe("AggregatedCostCard: custo nao informado nao vira US$ 0", () => {
    it("sem preco para o modelo mostra 'Não informado' em vez de US$ 0,000000", () => {
        pathnameState.value = "/pt-BR/documents";
        render(<AggregatedCostCard summary={ollama} />, { wrapper: createTestWrapper("pt-BR") });

        const cartao = screen.getByTestId("custo-agregado");
        expect(cartao).toHaveTextContent("Não informado");
        expect(cartao).not.toHaveTextContent("US$");
        expect(cartao).toHaveTextContent("1.753.745 tokens estimados · 299 requests");
    });

    it("traduz 'não informado' em ingles e espanhol", () => {
        pathnameState.value = "/en/documents";
        const { unmount } = render(<AggregatedCostCard summary={ollama} />, { wrapper: createTestWrapper("en") });
        expect(screen.getByTestId("custo-agregado")).toHaveTextContent("Not reported");
        unmount();

        pathnameState.value = "/es/documents";
        render(<AggregatedCostCard summary={ollama} />, { wrapper: createTestWrapper("es") });
        expect(screen.getByTestId("custo-agregado")).toHaveTextContent("No informado");
    });

    it("com preco em todos os documentos mostra o valor em dolar", () => {
        pathnameState.value = "/pt-BR/documents";
        render(
            <AggregatedCostCard summary={{ ...ollama, pricing_aware_documents: 12, total_estimated_cost_usd: 0.0134 }} />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        const cartao = screen.getByTestId("custo-agregado");
        expect(cartao).toHaveTextContent("US$ 0,013400");
        expect(cartao).not.toHaveTextContent("Não informado");
    });

    it("com preco so em parte dos documentos marca o valor como parcial", () => {
        pathnameState.value = "/pt-BR/documents";
        render(
            <AggregatedCostCard summary={{ ...ollama, pricing_aware_documents: 3, total_estimated_cost_usd: 0.01 }} />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        const cartao = screen.getByTestId("custo-agregado");
        expect(cartao).toHaveTextContent("US$ 0,010000");
        expect(cartao).toHaveTextContent("preço conhecido em 3 de 12 documentos");
    });
});
