import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ConnectionLayer } from "@/components/workspace/ConnectionLayer";
import type { Linha } from "@/lib/workspace/connections";
import { createTestWrapper } from "@/test/test-utils";

// O I18nProvider le o locale do pathname; sem o mock cai em ingles.
vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const opcoes = { wrapper: createTestWrapper("pt-BR") };

const LINHAS: Linha[] = [
    { id: "2:a", documentId: 2, nodeId: "a", titulo: "2 DAS PENALIDADES", score: 3, termos: ["multa", "fatura"], de: { x: 0, y: 0 }, para: { x: 100, y: 50 }, aproximada: false },
    { id: "2:b", documentId: 2, nodeId: "b", titulo: "3 PRAZOS", score: 2, termos: ["prazo"], de: { x: 0, y: 0 }, para: { x: 100, y: 200 }, aproximada: true },
];

describe("ConnectionLayer", () => {
    it("desenha uma linha por conexao com o score em termos e os termos no title", () => {
        render(<ConnectionLayer linhas={LINHAS} realcada={null} onRealcar={() => undefined} onEscolher={() => undefined} />, opcoes);
        const linhas = screen.getAllByTestId("linha-de-conexao");
        expect(linhas).toHaveLength(2);
        expect(screen.getAllByTestId("rotulo-da-linha").map((t) => t.textContent)).toEqual(["3 termos", "2 termos"]);
        expect(linhas[0].querySelector("title")?.textContent).toBe("2 DAS PENALIDADES: multa, fatura");
        // Linha aproximada (cai na borda do quadro) e tracejada.
        expect(linhas[1].querySelector("path")).toHaveAttribute("stroke-dasharray", "6 4");
        expect(linhas[0].querySelector("path")).not.toHaveAttribute("stroke-dasharray");
    });

    it("realca a linha sob o mouse e escolhe ao clicar", () => {
        const onRealcar = vi.fn();
        const onEscolher = vi.fn();
        render(<ConnectionLayer linhas={LINHAS} realcada="2:b" onRealcar={onRealcar} onEscolher={onEscolher} />, opcoes);
        const [a, b] = screen.getAllByTestId("linha-de-conexao");
        expect(a).toHaveAttribute("opacity", "0.25");
        expect(b).toHaveAttribute("opacity", "1");
        fireEvent.mouseEnter(a);
        expect(onRealcar).toHaveBeenCalledWith("2:a");
        fireEvent.click(b);
        expect(onEscolher).toHaveBeenCalledWith(LINHAS[1]);
    });

    it("sem linhas nao monta svg", () => {
        render(<ConnectionLayer linhas={[]} realcada={null} onRealcar={() => undefined} onEscolher={() => undefined} />, opcoes);
        expect(screen.queryByTestId("camada-de-conexoes")).toBeNull();
    });
});
