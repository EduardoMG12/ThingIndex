import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DocumentosDaColecao } from "@/components/workspace/DocumentosDaColecao";
import { createTestWrapper } from "@/test/test-utils";

// O provider de i18n le o locale do pathname.
vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const documentos = [
    { id: 3, collectionId: 7, title: "Termo assinado", filePath: "a.pdf", mimeType: "application/pdf", status: "INDEXED" },
    { id: 4, collectionId: 7, title: "Edital longo", filePath: "b.pdf", mimeType: "application/pdf", status: "PROCESSING" },
];

describe("DocumentosDaColecao", () => {
    it("lista com status, foca ao clicar no titulo e reindexa pelo botao", async () => {
        const user = userEvent.setup();
        const onFocar = vi.fn();
        const onReindexar = vi.fn().mockResolvedValue(undefined);
        render(
            <DocumentosDaColecao documentos={documentos} onFocar={onFocar} onReindexar={onReindexar} onApagar={vi.fn()} />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        expect(screen.getAllByTestId("status-do-documento").map((el) => el.textContent)).toEqual(["Indexado", "Indexando"]);
        await user.click(screen.getByRole("button", { name: /Edital longo/ }));
        expect(onFocar).toHaveBeenCalledWith(4);
        await user.click(screen.getAllByRole("button", { name: "Reindexar" })[0]);
        expect(onReindexar).toHaveBeenCalledWith(3);
    });

    it("apagar so chama a API depois de digitar o titulo no modal", async () => {
        const user = userEvent.setup();
        const onApagar = vi.fn().mockResolvedValue(undefined);
        render(
            <DocumentosDaColecao documentos={documentos} onFocar={vi.fn()} onReindexar={vi.fn()} onApagar={onApagar} />,
            { wrapper: createTestWrapper("pt-BR") }
        );
        await user.click(screen.getAllByRole("button", { name: "Apagar" })[0]);
        const confirmar = screen.getByRole("button", { name: "Confirmar exclusão" });
        expect(confirmar).toBeDisabled();
        expect(onApagar).not.toHaveBeenCalled();
        await user.type(screen.getByRole("textbox", { name: "Valor de confirmação" }), "Termo assinado");
        await user.click(confirmar);
        expect(onApagar).toHaveBeenCalledWith(3);
    });
});
