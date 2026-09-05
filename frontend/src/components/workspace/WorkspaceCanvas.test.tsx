import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorkspaceCanvas } from "@/components/workspace/WorkspaceCanvas";
import type { CollectionContext } from "@/lib/api/contracts/document-context";

vi.mock("@/lib/api/collections", () => ({
    fetchNodeConnections: vi.fn(),
}));

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
        render(<WorkspaceCanvas context={contexto} />);

        expect(screen.getByText("Contrato SLA")).toBeInTheDocument();
        expect(screen.getByText("Edital")).toBeInTheDocument();
    });

    it("abre a sidebar com o outline ao focar um documento", () => {
        render(<WorkspaceCanvas context={contexto} />);

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

        render(<WorkspaceCanvas context={contexto} />);
        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));
        fireEvent.click(screen.getAllByRole("button", { name: /conex/i })[0]);

        await waitFor(() => {
            expect(screen.getByText("2 DAS PENALIDADES")).toBeInTheDocument();
        });

        // A aresta precisa dizer POR QUE existe: sem isso seria tao opaca
        // quanto a vizinhanca vetorial que este projeto evita.
        expect(screen.getByText(/multa/)).toBeInTheDocument();
        expect(screen.getByText(/descumprimento/)).toBeInTheDocument();
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

        render(<WorkspaceCanvas context={contexto} />);
        fireEvent.click(screen.getByRole("button", { name: /Contrato SLA/i }));
        fireEvent.click(screen.getAllByRole("button", { name: /conex/i })[0]);

        await waitFor(() => {
            expect(screen.getByText(/nenhuma conex/i)).toBeInTheDocument();
        });
    });

    it("persiste a posicao do cartao arrastado", () => {
        const { unmount } = render(<WorkspaceCanvas context={contexto} />);

        const cartao = screen.getByRole("button", { name: /Contrato SLA/i });
        fireEvent.pointerDown(cartao, { clientX: 0, clientY: 0, pointerId: 1 });
        fireEvent.pointerMove(window, { clientX: 120, clientY: 80, pointerId: 1 });
        fireEvent.pointerUp(window, { pointerId: 1 });

        const salvo = window.localStorage.getItem("thingindex.workspace.positions.7");
        expect(salvo).toBeTruthy();
        expect(JSON.parse(salvo as string)).toHaveProperty("1");

        unmount();

        // Ao remontar, a posicao salva e reaplicada.
        render(<WorkspaceCanvas context={contexto} />);
        const remontado = screen.getByRole("button", { name: /Contrato SLA/i });
        expect(remontado.parentElement?.style.transform).toContain("translate");
    });

    it("nao quebra quando a colecao nao tem nos indexados", () => {
        const vazio = {
            ...contexto,
            outlineNodes: [],
        } as unknown as CollectionContext;

        render(<WorkspaceCanvas context={vazio} />);
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
        const { container } = render(<WorkspaceCanvas context={contexto} />);

        const raiz = container.firstElementChild as HTMLElement;
        expect(raiz.className).toContain("flex-col");
        expect(raiz.className).toContain("lg:flex-row");

        const canvas = screen.getByTestId("workspace-canvas");
        expect(canvas.className).toContain("min-w-0");

        const sidebar = screen.getByLabelText("Detalhes do documento");
        expect(sidebar.className).toContain("w-full");
        expect(sidebar.className).toContain("lg:w-80");
    });
});
