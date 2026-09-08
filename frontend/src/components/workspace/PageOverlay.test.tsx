import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { OutlineNode, ParagraphNode } from "@/lib/api/contracts/document-context";
import { AvisoSemPosicao, distribuirMarcadores, origemMajoritaria, PageOverlay } from "@/components/workspace/PageOverlay";
import { createTestWrapper } from "@/test/test-utils";

// O I18nProvider le o locale do pathname; sem o mock cai em ingles.
vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const opcoes = { wrapper: createTestWrapper("pt-BR") };

vi.mock("@/lib/api/documents", () => ({
    retryDocumentIndexing: vi.fn(async () => ({ id: 9, status: "QUEUED" })),
}));

const BOX = { x_min: 72, x_max: 540, y_min: 100, y_max: 120, page_width: 612, page_height: 792 };

function paragrafo(id: number, extra: Partial<ParagraphNode> = {}): ParagraphNode {
    return { id, documentId: 1, sequenceOrder: id, content: `texto ${id}`, pageNumber: 1, metadata: { box: BOX, source: "pdftotext", section_key: "1-relatorio" }, ...extra };
}

const SECAO: OutlineNode = { id: 1, nodeId: "node-001-1-relatorio", documentId: 1, sequenceOrder: 0, level: 1, title: "1 RELATORIO", slug: "1-relatorio", startPage: 1 };

describe("PageOverlay", () => {
    it("desenha um grifo por paragrafo com box, na posicao do box, e nenhum para os sem box", () => {
        const onSelecionar = vi.fn();
        render(
            <PageOverlay
                numero={1}
                tamanho={{ largura: 612, altura: 792 }}
                paragrafos={[
                    { no: paragrafo(1), secao: SECAO },
                    { no: paragrafo(2, { metadata: { box: { ...BOX, y_min: 200, y_max: 230 } } }), secao: null },
                    { no: paragrafo(3, { metadata: { source: "pdf_text" } }), secao: null },
                ]}
                secoes={[SECAO]}
                ancoraDaSecao={() => paragrafo(1)}
                origem="pdftotext"
                selecionadoId={2}
                secundarios={new Set()}
                onSelecionar={onSelecionar}
            />,
            opcoes
        );
        const grifos = screen.getAllByTestId("grifo");
        expect(grifos).toHaveLength(2);
        expect(grifos[0]).toHaveStyle({ left: "72px", top: "100px", width: "468px", height: "20px" });
        expect(grifos[0]).toHaveAttribute("aria-label", "Parágrafo 2, página 1, seção 1 RELATORIO");
        expect(grifos[1]).toHaveAttribute("aria-pressed", "true");

        fireEvent.click(grifos[0]);
        expect(onSelecionar).toHaveBeenCalledWith(expect.objectContaining({ id: 1 }), SECAO);
    });

    it("poe o marcador da secao na altura do primeiro paragrafo e o cabecalho com a origem", () => {
        render(
            <PageOverlay
                numero={3}
                tamanho={{ largura: 612, altura: 792 }}
                paragrafos={[]}
                secoes={[SECAO, { ...SECAO, id: 2, nodeId: "node-002-x", level: 3, title: "1.1.1 nivel fundo" }]}
                ancoraDaSecao={() => paragrafo(1)}
                origem="ocr"
                selecionadoId={null}
                secundarios={new Set()}
                onSelecionar={() => undefined}
            />,
            opcoes
        );
        // O no de nivel 3 nao vira marcador: so ate o nivel 2.
        expect(screen.getAllByTestId("marcador-secao")).toHaveLength(1);
        expect(screen.getByTestId("marcador-secao")).toHaveStyle({ top: "100px" });
        expect(screen.getByTestId("marcador-secao")).toHaveTextContent("1 RELATORIO");
        expect(screen.getByTestId("cabecalho-pagina")).toHaveTextContent("p. 3 · ocr");
    });
});

describe("PageOverlay: cores", () => {
    it("grifo marcado usa a classe da cor e o aria-label ganha o rotulo; secundario segue a cor de origem", () => {
        render(
            <PageOverlay
                numero={1}
                tamanho={{ largura: 612, altura: 792 }}
                paragrafos={[
                    { no: paragrafo(1), secao: SECAO },
                    { no: paragrafo(2, { metadata: { box: { ...BOX, y_min: 200, y_max: 230 }, section_key: "1-relatorio" } }), secao: SECAO },
                ]}
                secoes={[]}
                ancoraDaSecao={() => null}
                origem={null}
                selecionadoId={null}
                secundarios={new Set([2])}
                onSelecionar={() => undefined}
                marcas={new Map([[1, "verde"]])}
                legenda={{ verde: "requisito" }}
                corSecundaria="verde"
                onMarcar={() => undefined}
            />,
            opcoes
        );
        const [marcado, secundario] = screen.getAllByTestId("grifo");
        expect(marcado).toHaveAttribute("data-cor", "verde");
        expect(marcado.className).toContain("bg-green-300/40");
        expect(marcado).toHaveAttribute("aria-label", expect.stringContaining("grifo requisito"));
        expect(secundario.className).toContain("bg-green-200/30");
        expect(secundario).not.toHaveAttribute("data-cor");
    });
});

describe("distribuirMarcadores", () => {
    it("empurra para baixo marcadores que colidiriam e mantem a ordem pela ancora", () => {
        const saida = distribuirMarcadores([
            { item: "b", top: 105 },
            { item: "a", top: 100 },
            { item: "c", top: 200 },
        ]);
        expect(saida.map((m) => m.item)).toEqual(["a", "b", "c"]);
        expect(saida.map((m) => m.top)).toEqual([100, 114, 200]);
    });
});

describe("origemMajoritaria", () => {
    it("devolve a origem mais frequente e null sem origem", () => {
        expect(origemMajoritaria([{ metadata: { source: "ocr" } }, { metadata: { source: "pdftotext" } }, { metadata: { source: "pdftotext" } }])).toBe("pdftotext");
        expect(origemMajoritaria([{ metadata: {} }])).toBeNull();
    });
});

describe("AvisoSemPosicao", () => {
    it("so reindexa ao clicar e mostra que entrou na fila", async () => {
        const { retryDocumentIndexing } = await import("@/lib/api/documents");
        render(<AvisoSemPosicao documentId={9} totalParagrafos={47} />, opcoes);
        expect(retryDocumentIndexing).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole("button", { name: "Reindexar" }));
        expect(retryDocumentIndexing).toHaveBeenCalledWith(9);
        expect(await screen.findByText("Na fila de indexação")).toBeVisible();
    });
});
