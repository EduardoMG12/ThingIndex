import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PdfFrame } from "@/components/workspace/PdfFrame";
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
});
