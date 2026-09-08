import { fireEvent, render, screen, within, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorkspaceCanvas } from "@/components/workspace/WorkspaceCanvas";
import type { CollectionContext } from "@/lib/api/contracts/document-context";

vi.mock("@/lib/api/collections", () => ({
    fetchNodeConnections: vi.fn(),
    fetchHighlights: vi.fn(async () => []),
    fetchLegend: vi.fn(async () => ({})),
    putHighlight: vi.fn(async () => null),
    deleteHighlight: vi.fn(async () => undefined),
    fetchViewState: vi.fn(async () => ({})),
    putViewState: vi.fn(async () => undefined),
}));

const navegacao = {
    replace: vi.fn(),
    searchParams: new URLSearchParams(),
};

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
    useRouter: () => ({ replace: navegacao.replace }),
    useSearchParams: () => navegacao.searchParams,
}));

import { createTestWrapper } from "@/test/test-utils";
import { deleteHighlight, putHighlight, putViewState } from "@/lib/api/collections";

// pdf.js mockado: documentos de tres paginas carta, para o overlay de grifos
// (e a barra de cor) existir no teste do canvas.
vi.mock("@/components/workspace/pdf", () => ({
    abrirDocumento: vi.fn(async () => ({
        numPages: 3,
        fingerprints: ["fake"],
        getPage: async () => ({
            getViewport: ({ scale }: { scale: number }) => ({ width: 612 * scale, height: 792 * scale }),
            render: () => ({ promise: Promise.resolve(), cancel: () => undefined }),
        }),
    })),
    fecharDocumento: vi.fn(),
    medirPaginas: vi.fn(async () => Array.from({ length: 3 }, () => ({ largura: 612, altura: 792 }))),
}));

const BOX = { x_min: 72, x_max: 540, y_min: 100, y_max: 120, page_width: 612, page_height: 792 };
const contextoComPdf = {
    collection: { id: 7, name: "Processo XYZ", metadata: {} },
    documents: [{ id: 3, title: "Termo", status: "INDEXED", mimeType: "application/pdf", filePath: "docs/termo.pdf" }],
    outlineNodes: [{ documentId: 3, nodeId: "termo#a", title: "1 A", level: 1, slug: "a", startPage: 1, metadata: { paragraph_start: 0 } }],
    paragraphNodes: [
        { id: 301, documentId: 3, sequenceOrder: 0, content: "Multa de 5 por cento.", pageNumber: 1, metadata: { box: BOX, section_key: "a" } },
    ],
    tableSections: [],
    metadataPairs: [],
} as unknown as CollectionContext;

const opcoes = { wrapper: createTestWrapper("pt-BR") };

import { fetchNodeConnections } from "@/lib/api/collections";

const contexto = {
    collection: { id: 7, name: "Processo XYZ", metadata: {} },
    documents: [
        { id: 1, title: "Contrato SLA", status: "INDEXED" },
        { id: 2, title: "Edital", status: "INDEXED" },
    ],
    outlineNodes: [
        {
            documentId: 1,
            nodeId: "sla#penalidades",
            title: "2 PENALIDADES",
            summary: "Multa de 5 por cento sobre a fatura mensal.",
            level: 1,
            startPage: 1,
            endPage: 1,
        },
        {
            documentId: 1,
            nodeId: "sla#disponibilidade",
            title: "1 DISPONIBILIDADE",
            summary: "Uptime de 99,5 por cento.",
            level: 1,
            startPage: 1,
            endPage: 1,
        },
        {
            documentId: 2,
            nodeId: "edital#penalidades",
            title: "2 DAS PENALIDADES",
            summary: "Multa por descumprimento.",
            level: 1,
            startPage: 1,
            endPage: 1,
        },
    ],
    documentPages: [],
    paragraphNodes: [],
    tableSections: [],
    metadataPairs: [],
    chunks: [],
} as unknown as CollectionContext;

describe("WorkspaceCanvas", () => {
    beforeEach(() => {
        vi.mocked(fetchNodeConnections).mockReset();
        // jsdom nesta versao nao expoe localStorage.clear diretamente no
        // escopo do teste; substituir por um duplo em memoria mantem o teste
        // independente do ambiente e ainda exercita o caminho real de
        // persistencia do componente.
        const memoria = new Map<string, string>();
        Object.defineProperty(window, "localStorage", {
            configurable: true,
            value: {
                getItem: (chave: string) => memoria.get(chave) ?? null,
                setItem: (chave: string, valor: string) => {
                    memoria.set(chave, valor);
                },
                removeItem: (chave: string) => {
                    memoria.delete(chave);
                },
                clear: () => memoria.clear(),
                key: (indice: number) => Array.from(memoria.keys())[indice] ?? null,
                get length() {
                    return memoria.size;
                },
            },
        });
    });

    it("mostra um cartao por documento da colecao", () => {
        render(<WorkspaceCanvas context={contexto} />, opcoes);

        expect(screen.getByText("Contrato SLA")).toBeInTheDocument();
        expect(screen.getByText("Edital")).toBeInTheDocument();
    });

    it("abre a sidebar com o outline ao focar um documento", () => {
        render(<WorkspaceCanvas context={contexto} />, opcoes);

        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));

        // A sidebar mostra os nos DAQUELE documento, com os resumos que ja
        // existem no indice -- nenhuma chamada de LLM no caminho.
        expect(screen.getByText("2 PENALIDADES")).toBeInTheDocument();
        expect(screen.getByText("1 DISPONIBILIDADE")).toBeInTheDocument();
        expect(screen.getByText(/Multa de 5 por cento/)).toBeInTheDocument();

        // Nos de outro documento nao entram na sidebar do documento focado.
        expect(screen.queryByText("2 DAS PENALIDADES")).not.toBeInTheDocument();
    });

    it("busca conexoes ao pedir, e mostra a justificativa de cada uma", async () => {
        vi.mocked(fetchNodeConnections).mockResolvedValue({
            nodeId: "sla#penalidades",
            minScore: 2,
            connections: [
                {
                    documentId: 2,
                    nodeId: "edital#penalidades",
                    title: "2 DAS PENALIDADES",
                    summary: "Multa por descumprimento.",
                    startPage: 1,
                    score: 11,
                    matchedTerms: ["multa", "fatura", "descumprimento"],
                },
            ],
        });

        render(<WorkspaceCanvas context={contexto} />, opcoes);
        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));
        fireEvent.click(screen.getAllByRole("button", { name: /conex/i })[0]);

        await waitFor(() => {
            expect(screen.getByText("2 DAS PENALIDADES")).toBeInTheDocument();
        });

        // A aresta precisa dizer POR QUE existe: sem isso seria tao opaca
        // quanto a vizinhanca vetorial que este projeto evita.
        // A justificativa precisa estar na sidebar (a linha SVG tambem repete
        // os termos no <title>, entao o escopo e o painel de detalhes).
        const sidebar = within(screen.getByLabelText("Detalhes do documento"));
        expect(sidebar.getByText(/multa/)).toBeInTheDocument();
        expect(sidebar.getByText(/descumprimento/)).toBeInTheDocument();
    });

    it("declara ausencia de conexao em vez de mostrar area vazia", async () => {
        // Este e o teste que mais importa. Lista vazia significa "nenhuma
        // conexao supera o limiar", que e a resposta CORRETA quando nao ha
        // relacao. A interface precisa dizer isso com todas as letras -- uma
        // area em branco seria lida como falha de carregamento, e o sistema
        // perderia credito justamente pelo unico ponto em que sabe se calar.
        vi.mocked(fetchNodeConnections).mockResolvedValue({
            nodeId: "sla#disponibilidade",
            minScore: 2,
            connections: [],
        });

        render(<WorkspaceCanvas context={contexto} />, opcoes);
        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));
        fireEvent.click(screen.getAllByRole("button", { name: /conex/i })[0]);

        await waitFor(() => {
            expect(screen.getByText(/nenhuma conex/i)).toBeInTheDocument();
        });
    });

    it("persiste a posicao do cartao arrastado", async () => {
        const { unmount } = render(<WorkspaceCanvas context={contexto} />, opcoes);

        const cartao = screen.getByRole("button", { name: /Contrato SLA/i });
        fireEvent.pointerDown(cartao, { clientX: 0, clientY: 0, pointerId: 1 });
        fireEvent.pointerMove(window, { clientX: 120, clientY: 80, pointerId: 1 });
        fireEvent.pointerUp(window, { pointerId: 1 });

        // A posicao vai para o servidor (debounce de 800 ms), nao mais para o localStorage.
        await waitFor(() => expect(putViewState).toHaveBeenCalled(), { timeout: 2500 });
        const enviado = vi.mocked(putViewState).mock.calls.at(-1)?.[1];
        expect(enviado?.quadros).toHaveProperty("1");
        expect(window.localStorage.getItem("thingindex.workspace.positions.7")).toBeNull();

        unmount();

        // Ao remontar, a posicao salva e reaplicada.
        render(<WorkspaceCanvas context={contexto} />, opcoes);
        const remontado = screen.getByRole("button", { name: /Contrato SLA/i });
        expect(remontado.parentElement?.style.transform).toContain("translate");
    });

    it("nao quebra quando a colecao nao tem nos indexados", () => {
        const vazio = {
            ...contexto,
            outlineNodes: [],
        } as unknown as CollectionContext;

        render(<WorkspaceCanvas context={vazio} />, opcoes);
        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));

        expect(screen.getByText(/sem estrutura indexada/i)).toBeInTheDocument();
    });

    it("empilha painel e sidebar em telas estreitas", () => {
        // Regressao medida no navegador: com `flex` e sidebar de largura fixa
        // (w-80), num viewport de 780px o canvas era espremido para 72px e os
        // cartoes ficavam ilegiveis atras da sidebar.
        //
        // Duas garantias: a coluna so aparece a partir de lg, e o canvas tem
        // min-w-0 -- sem ele um item flex assume a largura do conteudo e volta
        // a estourar o container, que foi exatamente a causa do bug.
        const { container } = render(<WorkspaceCanvas context={contexto} />, opcoes);

        const raiz = container.firstElementChild as HTMLElement;
        expect(raiz.className).toContain("flex-col");
        expect(raiz.className).toContain("lg:flex-row");

        const canvas = screen.getByTestId("workspace-canvas");
        expect(canvas.className).toContain("min-w-0");

        const sidebar = screen.getByLabelText("Detalhes do documento");
        expect(sidebar.className).toContain("w-full");
        expect(sidebar.className).toContain("lg:w-72");
        expect(sidebar.className).toContain("overflow-y-auto");
        expect(sidebar.className).toContain("lg:max-h-[calc(100vh-9rem)]");
    });

    it("le o foco de ?doc= ao montar e escreve na URL ao focar outro documento", () => {
        navegacao.replace.mockClear();
        navegacao.searchParams = new URLSearchParams("doc=2");
        render(<WorkspaceCanvas context={contexto} />, opcoes);
        // Documento 2 ja aparece focado na sidebar, sem clique.
        expect(within(screen.getByLabelText("Detalhes do documento")).getByText("Edital")).toBeInTheDocument();
        expect(navegacao.replace).not.toHaveBeenCalled();

        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));
        expect(navegacao.replace).toHaveBeenCalledWith("/pt-BR/workspace?doc=1", { scroll: false });
        navegacao.searchParams = new URLSearchParams();
    });

    it("marca com cor de forma otimista e desfaz quando a API falha", async () => {
        vi.mocked(putHighlight).mockRejectedValueOnce(new Error("Cor invalida"));
        // O quadro so abre quando entra no viewport, e o viewport vem do
        // ResizeObserver; o stub do jsdom nunca dispara, entao este avisa um
        // tamanho de tela de verdade.
        const original = globalThis.ResizeObserver;
        globalThis.ResizeObserver = class {
            private cb: ResizeObserverCallback;
            constructor(cb: ResizeObserverCallback) {
                this.cb = cb;
            }
            observe() {
                this.cb([{ contentRect: { width: 1400, height: 900 } } as ResizeObserverEntry], this as unknown as ResizeObserver);
            }
            unobserve() {}
            disconnect() {}
        } as unknown as typeof ResizeObserver;
        try {
        render(<WorkspaceCanvas context={contextoComPdf} />, opcoes);
        const grifo = await screen.findByTestId("grifo", {}, { timeout: 3000 });
        fireEvent.click(grifo);
        const chip = await screen.findByTestId("chip-verde");
        fireEvent.click(chip);
        // Otimista: a cor entra antes da resposta.
        expect(screen.getByTestId("grifo")).toHaveAttribute("data-cor", "verde");
        expect(putHighlight).toHaveBeenCalledWith(7, 301, "verde");
        // Rollback: a API falhou, a cor sai e o erro aparece na sidebar.
        await waitFor(() => expect(screen.getByTestId("grifo")).not.toHaveAttribute("data-cor"));
        expect(screen.getByTestId("erro-de-marca")).toHaveTextContent("Cor invalida");

        // Sem falha: marca e, ao clicar de novo na mesma cor, remove com DELETE.
        fireEvent.click(screen.getByTestId("chip-verde"));
        await waitFor(() => expect(screen.getByTestId("grifo")).toHaveAttribute("data-cor", "verde"));
        fireEvent.click(screen.getByTestId("chip-verde"));
        await waitFor(() => expect(screen.getByTestId("grifo")).not.toHaveAttribute("data-cor"));
        expect(deleteHighlight).toHaveBeenCalledWith(7, 301);
        } finally {
            globalThis.ResizeObserver = original;
        }
    });

    it("em 'uma pagina', conexao para outra pagina mostra 'Ir para p. N' e o botao troca a pagina do quadro destino", async () => {
        const original = globalThis.ResizeObserver;
        globalThis.ResizeObserver = class {
            private cb: ResizeObserverCallback;
            constructor(cb: ResizeObserverCallback) {
                this.cb = cb;
            }
            observe() {
                this.cb([{ contentRect: { width: 1400, height: 900 } } as ResizeObserverEntry], this as unknown as ResizeObserver);
            }
            unobserve() {}
            disconnect() {}
        } as unknown as typeof ResizeObserver;
        // Destino: documento 4, secao que comeca na pagina 3. Quadro em 'uma pagina' na 1 (estado salvo).
        vi.mocked(fetchNodeConnections).mockResolvedValue({
            nodeId: "termo#a",
            minScore: 2,
            connections: [{ documentId: 4, nodeId: "outro#c", title: "3 PRAZOS", summary: "", startPage: 3, score: 4, matchedTerms: ["prazo"] }],
        });
        const contexto = {
            ...contextoComPdf,
            documents: [...contextoComPdf.documents, { id: 4, title: "Outro", status: "INDEXED", mimeType: "application/pdf", filePath: "docs/outro.pdf" }],
            outlineNodes: [...contextoComPdf.outlineNodes, { documentId: 4, nodeId: "outro#c", title: "3 PRAZOS", level: 1, slug: "c", startPage: 3 }],
        } as unknown as CollectionContext;
        try {
            render(<WorkspaceCanvas context={contexto} estadoInicial={{ apresentacao: { "4": { modo: "pagina", paginaAtual: 1 } } }} />, opcoes);
            fireEvent.click(await screen.findByTestId("grifo", {}, { timeout: 3000 }));
            // A barra flutuante e a sidebar tem 'Ver conexoes'; aqui e a da barra.
            fireEvent.click(within(await screen.findByTestId("barra-de-grifo")).getByRole("button", { name: /Ver conexões/ }));
            const botao = await screen.findByTestId("ir-para-pagina");
            expect(botao).toHaveTextContent("Ir para p. 3");
            fireEvent.click(botao);
            await waitFor(() => expect(screen.getByTestId("pdf-frame-4")).toHaveAttribute("data-modo", "pagina"));
            await waitFor(() => expect(within(screen.getByTestId("pdf-frame-4")).getByTestId("pagina-de")).toHaveTextContent("p. 3 de 3"));
        } finally {
            globalThis.ResizeObserver = original;
        }
    });

    it("abre ja no estado salvo: zoom, pagina do quadro e marcadores escondidos, sem esperar efeito", async () => {
        const original = globalThis.ResizeObserver;
        globalThis.ResizeObserver = class {
            private cb: ResizeObserverCallback;
            constructor(cb: ResizeObserverCallback) {
                this.cb = cb;
            }
            observe() {
                this.cb([{ contentRect: { width: 1400, height: 900 } } as ResizeObserverEntry], this as unknown as ResizeObserver);
            }
            unobserve() {}
            disconnect() {}
        } as unknown as typeof ResizeObserver;
        try {
            vi.mocked(putViewState).mockClear();
            render(
                <WorkspaceCanvas
                    context={contextoComPdf}
                    estadoInicial={{ viewport: { x: 0, y: 0, escala: 0.75 }, apresentacao: { "3": { modo: "pagina", paginaAtual: 2 } }, marcadoresVisiveis: false }}
                />,
                opcoes
            );
            // Primeiro render: 75% e modo pagina ja aplicados.
            expect(screen.getByRole("toolbar", { name: "Zoom do canvas" })).toHaveTextContent("75%");
            expect(screen.getByTestId("pdf-frame-3")).toHaveAttribute("data-modo", "pagina");
            await screen.findByTestId("pagina-de");
            expect(screen.getByTestId("pagina-de")).toHaveTextContent("p. 2 de 3");
            expect(screen.getByTestId("alternar-marcadores")).toHaveAttribute("aria-pressed", "false");
            // Nada mudou: nada e salvo de volta.
            await new Promise((r) => setTimeout(r, 1000));
            expect(putViewState).not.toHaveBeenCalled();
            // Mostrar marcadores muda o estado e salva.
            fireEvent.click(screen.getByTestId("alternar-marcadores"));
            await waitFor(() => expect(putViewState).toHaveBeenCalled(), { timeout: 2500 });
            expect(vi.mocked(putViewState).mock.calls.at(-1)?.[1]?.marcadoresVisiveis).toBe(true);
        } finally {
            globalThis.ResizeObserver = original;
        }
    });

    it("migra posicoes e apresentacao do localStorage uma vez e apaga as chaves", async () => {
        vi.mocked(putViewState).mockClear();
        window.localStorage.setItem("thingindex.workspace.positions.7", JSON.stringify({ "1": { x: 300, y: 40 } }));
        window.localStorage.setItem("thingindex.workspace.apresentacao.7", JSON.stringify({ "1": { modo: "pagina", paginaAtual: 2 } }));
        render(<WorkspaceCanvas context={contexto} />, opcoes);
        const cartao = screen.getByRole("button", { name: /Contrato SLA/i });
        expect(cartao.parentElement?.style.transform).toContain("translate(300px, 40px)");
        expect(window.localStorage.getItem("thingindex.workspace.positions.7")).toBeNull();
        await waitFor(() => expect(putViewState).toHaveBeenCalled(), { timeout: 2500 });
        const enviado = vi.mocked(putViewState).mock.calls.at(-1)?.[1];
        expect(enviado?.quadros).toEqual({ "1": { x: 300, y: 40 } });
        expect(enviado?.apresentacao).toEqual({ "1": { modo: "pagina", paginaAtual: 2 } });
    });
});
