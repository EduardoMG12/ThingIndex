import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { HighlightToolbar } from "@/components/workspace/HighlightToolbar";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const opcoes = { wrapper: createTestWrapper("pt-BR") };
const rect = { left: 72, top: 200, width: 400, height: 14 };

describe("HighlightToolbar", () => {
    it("mostra cinco chips com o rotulo da legenda ou o nome da cor, e marca a atual", () => {
        const onMarcar = vi.fn();
        render(
            <HighlightToolbar rect={rect} corAtual="amarelo" legenda={{ amarelo: "prazo" }} onMarcar={onMarcar} onVerConexoes={() => undefined} podeVerConexoes />,
            opcoes
        );
        expect(screen.getByRole("button", { name: "prazo" })).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByRole("button", { name: "Verde" })).toHaveAttribute("aria-pressed", "false");
        expect(screen.getAllByTestId(/^chip-(amarelo|verde|azul|vermelho|roxo)$/)).toHaveLength(5);

        fireEvent.click(screen.getByRole("button", { name: "Verde" }));
        expect(onMarcar).toHaveBeenCalledWith("verde");
        // Clicar na cor ja ativa desmarca.
        fireEvent.click(screen.getByRole("button", { name: "prazo" }));
        expect(onMarcar).toHaveBeenLastCalledWith(null);
        fireEvent.click(screen.getByTestId("chip-sem-cor"));
        expect(onMarcar).toHaveBeenLastCalledWith(null);
    });

    it("fica acima do paragrafo quando cabe e abaixo quando nao cabe; 'sem cor' e 'ver conexoes' respeitam o estado", () => {
        const onVer = vi.fn();
        const { rerender } = render(
            <HighlightToolbar rect={rect} corAtual={null} legenda={{}} onMarcar={() => undefined} onVerConexoes={onVer} podeVerConexoes={false} />,
            opcoes
        );
        expect(screen.getByTestId("barra-de-grifo")).toHaveStyle({ top: "164px" });
        expect(screen.getByTestId("chip-sem-cor")).toBeDisabled();
        expect(screen.getByRole("button", { name: /Ver conexões/ })).toBeDisabled();

        rerender(
            <HighlightToolbar rect={{ ...rect, top: 10 }} corAtual="roxo" legenda={{}} onMarcar={() => undefined} onVerConexoes={onVer} podeVerConexoes />
        );
        expect(screen.getByTestId("barra-de-grifo")).toHaveStyle({ top: "28px" });
        fireEvent.click(screen.getByRole("button", { name: /Ver conexões/ }));
        expect(onVer).toHaveBeenCalled();
    });
});
