import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PageOverlay } from "@/components/workspace/PageOverlay";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const box = { xMin: 20, yMin: 300, xMax: 560, yMax: 480, pageWidth: 595, pageHeight: 842 };

function montar(realcada: number | null) {
    render(
        <PageOverlay
            numero={1}
            tamanho={{ largura: 595, altura: 842 }}
            paragrafos={[]}
            secoes={[]}
            ancoraDaSecao={() => null}
            origem={null}
            selecionadoId={null}
            secundarios={new Set()}
            onSelecionar={() => undefined}
            tabelas={[{ id: 9, titulo: "Grade 07", box }]}
            tabelaRealcada={realcada}
        />,
        { wrapper: createTestWrapper("pt-BR") }
    );
}

describe("PageOverlay: caixa da tabela", () => {
    it("desenha a caixa tracejada so da tabela realcada, com o titulo", () => {
        montar(9);
        const caixa = screen.getByTestId("caixa-de-tabela");
        expect(caixa).toHaveAttribute("aria-label", "Tabela: Grade 07");
        expect(caixa).toHaveStyle({ top: "296px" });
    });

    it("sem tabela realcada nao desenha nada", () => {
        montar(null);
        expect(screen.queryByTestId("caixa-de-tabela")).toBeNull();
    });
});
