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

describe("PageOverlay: grifo fino por linha", () => {
    it("desenha so as linhas citadas do paragrafo apontado", () => {
        const no = {
            id: 5,
            documentId: 7,
            sequenceOrder: 0,
            pageNumber: 1,
            content: "a b c",
            metadata: {
                box: { x_min: 50, y_min: 100, x_max: 500, y_max: 160, page_width: 595, page_height: 842 },
                lines: [
                    { t: "primeira", b: [50, 100, 500, 120] },
                    { t: "segunda", b: [50, 120, 500, 140] },
                    { t: "terceira", b: [50, 140, 500, 160] },
                ],
            },
        } as unknown as import("@/lib/api/contracts/document-context").ParagraphNode;
        render(
            <PageOverlay
                numero={1}
                tamanho={{ largura: 595, altura: 842 }}
                paragrafos={[{ no, secao: null }]}
                secoes={[]}
                ancoraDaSecao={() => null}
                origem={null}
                selecionadoId={null}
                secundarios={new Set()}
                onSelecionar={() => undefined}
                destaque={{ paragraphId: 5, lines: [1] }}
            />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        const linhas = screen.getAllByTestId("grifo-linha");
        expect(linhas).toHaveLength(1);
        expect(linhas[0]).toHaveAttribute("title", "segunda");
        expect(linhas[0]).toHaveStyle({ top: "119px" });
    });
});

describe("PageOverlay: citacao gravada", () => {
    it("desenha a caixa e as linhas pela geometria gravada, sem paragrafo no contexto", () => {
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
                citacao={{ box, lineBoxes: [[20, 300, 560, 320], [20, 320, 560, 340]] }}
            />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        expect(screen.getByTestId("citacao-gravada")).toHaveStyle({ top: "298px" });
        expect(screen.getAllByTestId("grifo-linha")).toHaveLength(2);
    });
});
