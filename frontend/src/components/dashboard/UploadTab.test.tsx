import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import UploadTab from "@/components/dashboard/UploadTab";
import { createTestWrapper } from "@/test/test-utils";

vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/workspace",
}));

const addFiles = vi.fn(() => ({ accepted: [], rejections: [] }));
vi.mock("@/hooks/use-document-upload", () => ({
    useDocumentUpload: () => ({ files: [], isProcessing: false, queuedCount: 0, addFiles, removeFile: vi.fn(), processQueue: vi.fn() }),
}));
vi.mock("@/components/dashboard/CollectionCombobox", () => ({ default: () => <div data-testid="combobox-colecao" /> }));
vi.mock("@/components/dashboard/CategoryCombobox", () => ({ default: () => <div data-testid="combobox-categoria" /> }));

describe("UploadTab com colecao fixa", () => {
    it("nao mostra o combobox de colecao e envia os arquivos com a colecao fixa", async () => {
        const user = userEvent.setup();
        const { container } = render(<UploadTab colecaoFixa="Processo XYZ" />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.queryByTestId("combobox-colecao")).toBeNull();

        const arquivo = new File(["%PDF-1.4"], "termo.pdf", { type: "application/pdf" });
        await user.upload(container.querySelector('input[type="file"]') as HTMLInputElement, arquivo);
        expect(addFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "termo.pdf" })], { category: "", collection: "Processo XYZ" });
    });

    it("sem colecao fixa mantem o combobox", () => {
        render(<UploadTab />, { wrapper: createTestWrapper("pt-BR") });
        expect(screen.getByTestId("combobox-colecao")).toBeInTheDocument();
    });
});
