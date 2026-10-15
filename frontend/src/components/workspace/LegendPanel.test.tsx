import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LegendPanel } from "@/components/workspace/LegendPanel";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const opcoes = { wrapper: createTestWrapper("pt-BR") };

describe("LegendPanel", () => {
    beforeEach(() => {
        // Grava a chave em vez de limpar: o localStorage do jsdom muda entre
        // ambientes e na CI persiste entre testes.
        try {
            window.localStorage.setItem("thingindex.workspace.legenda.recolhida", "1");
        } catch {
            // sem storage, o painel ja comeca recolhida
        }
    });

    // A legenda comeca recolhida; os testes de edicao a abrem antes.
    function abrirLegenda() {
        fireEvent.click(screen.getByRole("button", { name: /Legenda/ }));
    }

    it("mostra o nome da cor como placeholder quando nao ha rotulo e salva a legenda inteira ao sair do campo", () => {
        const onSalvar = vi.fn();
        render(<LegendPanel legenda={{ amarelo: "prazo" }} onSalvar={onSalvar} />, opcoes);
        abrirLegenda();
        const verde = screen.getByTestId("legenda-verde") as HTMLInputElement;
        expect(verde.placeholder).toBe("Verde");
        expect((screen.getByTestId("legenda-amarelo") as HTMLInputElement).value).toBe("prazo");

        fireEvent.change(verde, { target: { value: "  requisito " } });
        fireEvent.blur(verde);
        expect(onSalvar).toHaveBeenCalledWith({ amarelo: "prazo", verde: "requisito" });

        // Sem mudanca, sair do campo nao salva de novo.
        fireEvent.blur(screen.getByTestId("legenda-amarelo"));
        expect(onSalvar).toHaveBeenCalledTimes(1);
    });

    it("Enter confirma, e a contagem por cor aparece quando informada", () => {
        const onSalvar = vi.fn();
        render(
            <LegendPanel legenda={{}} onSalvar={onSalvar} contagem={{ amarelo: 3, verde: 0, azul: 1, vermelho: 0, roxo: 0 }} />,
            opcoes
        );
        abrirLegenda();
        const azul = screen.getByTestId("legenda-azul");
        fireEvent.change(azul, { target: { value: "penalidade" } });
        fireEvent.keyDown(azul, { key: "Enter" });
        fireEvent.blur(azul);
        expect(onSalvar).toHaveBeenLastCalledWith({ azul: "penalidade" });
        expect(screen.getByTestId("contagem-amarelo")).toHaveTextContent("3");
        expect(screen.getByTestId("contagem-azul")).toHaveTextContent("1");
    });

    it("comeca recolhida (nao cobre o canto do PDF), expande e recolhe", () => {
        render(<LegendPanel legenda={{}} onSalvar={() => undefined} />, opcoes);
        expect(screen.queryByTestId("legenda-amarelo")).toBeNull();
        fireEvent.click(screen.getByRole("button", { name: /Legenda/ }));
        expect(screen.getByTestId("legenda-amarelo")).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: /Legenda/ }));
        expect(screen.queryByTestId("legenda-amarelo")).toBeNull();
    });
});
