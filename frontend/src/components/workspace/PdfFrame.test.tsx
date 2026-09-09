import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { alturaDoQuadro, PdfFrame, topoDaPagina } from "@/components/workspace/PdfFrame";
import { abrirDocumento } from "@/components/workspace/pdf";
import { createTestWrapper } from "@/test/test-utils";

// O I18nProvider le o locale do pathname; sem o mock cai em ingles.
vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

// pdf.js mockado: 30 paginas carta, render instantaneo, sem worker.
const documentoFalso = {
    numPages: 30,
    fingerprints: ["fake"],
    getPage: vi.fn(async () => ({
        getViewport: ({ scale }: { scale: number }) => ({ width: 612 * scale, height: 792 * scale }),
        render: () => ({ promise: Promise.resolve(), cancel: () => undefined }),
    })),
    destroy: vi.fn(),
};

vi.mock("@/components/workspace/pdf", () => ({
    abrirDocumento: vi.fn(async () => documentoFalso),
    fecharDocumento: vi.fn(),
    medirPaginas: vi.fn(async () => Array.from({ length: 30 }, () => ({ largura: 612, altura: 792 }))),
}));

function montar(paginasVivas: Set<number>, extra: Partial<React.ComponentProps<typeof PdfFrame>> = {}) {
    const onMedidas = vi.fn();
    render(
        <PdfFrame
            documentId={7}
            titulo="PPC"
            subtitulo="4382 trechos"
            visivel
            paginasVivas={paginasVivas}
            escalaDeRenderizacao={1}
            modoMiniatura={false}
            focado={false}
            onMedidas={onMedidas}
            onFocar={() => undefined}
            onPointerDownCabecalho={() => undefined}
            {...extra}
        />,
        { wrapper: createTestWrapper("pt-BR") }
    );
    return { onMedidas };
}

describe("PdfFrame", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("cria um placeholder por pagina com o tamanho real e avisa as medidas", async () => {
        const { onMedidas } = montar(new Set([1, 2]));
        await act(async () => {
            await Promise.resolve();
        });
        expect(onMedidas).toHaveBeenCalledWith(7, expect.arrayContaining([{ largura: 612, altura: 792 }]));
        const pagina5 = await screen.findByTestId("pdf-pagina-7-5");
        expect(pagina5).toHaveStyle({ width: "612px", height: "792px" });
        expect(screen.getByTestId("pdf-frame-7")).toHaveAttribute("data-paginas", "30");
    });

    it("so cria canvas para as paginas vivas, nunca para as 30", async () => {
        montar(new Set([3, 4, 5]));
        await screen.findByTestId("pdf-pagina-7-30");
        const canvases = document.querySelectorAll("canvas[data-pagina]");
        expect(canvases).toHaveLength(3);
        expect([...canvases].map((c) => c.getAttribute("data-pagina"))).toEqual(["3", "4", "5"]);
    });

    it("em modo miniatura nao cria canvas de pagina", async () => {
        montar(new Set([1]), { modoMiniatura: true });
        await act(async () => {
            await Promise.resolve();
            await Promise.resolve();
        });
        expect(document.querySelectorAll("canvas[data-pagina]")).toHaveLength(0);
        expect(screen.queryByTestId("pdf-pagina-7-1")).toBeNull();
    });

    it("em modo 'pagina' so a pagina atual esta no DOM e os botoes respeitam os limites", async () => {
        const onMudar = vi.fn();
        montar(new Set([1]), { apresentacao: { modo: "pagina", paginaAtual: 1 }, onMudarApresentacao: onMudar });
        await screen.findByTestId("pdf-pagina-7-1");
        expect(screen.queryByTestId("pdf-pagina-7-2")).toBeNull();
        expect(document.querySelectorAll('[data-testid^="pdf-pagina-7-"]')).toHaveLength(1);
        expect(screen.getByTestId("pagina-de")).toHaveTextContent("p. 1 de 30");
        expect(screen.getByRole("button", { name: "Página anterior" })).toBeDisabled();

        fireEvent.click(screen.getByRole("button", { name: "Próxima página" }));
        expect(onMudar).toHaveBeenCalledWith(7, { modo: "pagina", paginaAtual: 2 });

        // Na ultima pagina, avancar fica desabilitado; PageUp volta.
        const chamadas = vi.mocked(abrirDocumento).mock.calls.length;
        cleanup();
        montar(new Set([30]), { apresentacao: { modo: "pagina", paginaAtual: 30 }, onMudarApresentacao: onMudar });
        await screen.findByTestId("pdf-pagina-7-30");
        expect(screen.getByRole("button", { name: "Próxima página" })).toBeDisabled();
        fireEvent.keyDown(screen.getByTestId("pdf-frame-7"), { key: "PageUp" });
        expect(onMudar).toHaveBeenLastCalledWith(7, { modo: "pagina", paginaAtual: 29 });
        // Trocar de pagina nao reabre o documento: o cache de abrirDocumento
        // e chamado, mas a ideia e que o quadro nao dependa disso.
        expect(vi.mocked(abrirDocumento).mock.calls.length).toBeLessThanOrEqual(chamadas + 1);
    });

    it("no modo 'pagina' as faixas nas bordas da folha viram a pagina e somem nos limites", async () => {
        const onMudar = vi.fn();
        montar(new Set([1]), { apresentacao: { modo: "pagina", paginaAtual: 1 }, onMudarApresentacao: onMudar });
        await screen.findByTestId("pagina-de");
        // Primeira pagina: so a faixa de baixo.
        expect(screen.queryByTestId("virar-pagina-anterior")).toBeNull();
        fireEvent.click(screen.getByTestId("virar-pagina-proxima"));
        expect(onMudar).toHaveBeenLastCalledWith(7, { modo: "pagina", paginaAtual: 2 });
    });

    it("na ultima pagina so existe a faixa de cima, e ela volta uma pagina", async () => {
        const onMudar = vi.fn();
        montar(new Set([30]), { apresentacao: { modo: "pagina", paginaAtual: 30 }, onMudarApresentacao: onMudar });
        await screen.findByTestId("pagina-de");
        expect(screen.queryByTestId("virar-pagina-proxima")).toBeNull();
        fireEvent.click(screen.getByTestId("virar-pagina-anterior"));
        expect(onMudar).toHaveBeenLastCalledWith(7, { modo: "pagina", paginaAtual: 29 });
    });

    it("o seletor alterna entre folhas e uma pagina mantendo a pagina atual", async () => {
        const onMudar = vi.fn();
        montar(new Set([1]), { apresentacao: { modo: "folhas", paginaAtual: 4 }, onMudarApresentacao: onMudar });
        await screen.findByTestId("pdf-pagina-7-30");
        fireEvent.click(screen.getByTestId("alternar-modo"));
        expect(onMudar).toHaveBeenCalledWith(7, { modo: "pagina", paginaAtual: 4 });
    });
});

describe("geometria por modo", () => {
    const paginas = [
        { largura: 612, altura: 792 },
        { largura: 612, altura: 500 },
        { largura: 612, altura: 792 },
    ];
    it("em 'pagina' o quadro tem a altura da pagina atual e ela fica sob o cabecalho", () => {
        expect(alturaDoQuadro(paginas, { modo: "pagina", paginaAtual: 2 })).toBe(40 + 500);
        expect(topoDaPagina(paginas, 2, { modo: "pagina", paginaAtual: 2 })).toBe(40);
        expect(topoDaPagina(paginas, 3, { modo: "pagina", paginaAtual: 3 })).toBe(40);
    });
    it("em 'folhas' empilha todas com o espaco entre elas", () => {
        expect(alturaDoQuadro(paginas)).toBe(40 + 792 + 500 + 792 + 16 * 2);
        expect(topoDaPagina(paginas, 3)).toBe(40 + 792 + 16 + 500 + 16);
    });
});
