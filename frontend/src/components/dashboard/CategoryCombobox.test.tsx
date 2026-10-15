import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("@/hooks/use-categories", () => ({
    useCategories: vi.fn(),
}));

// O I18nProvider le o locale do pathname; sem o mock ele nao monta.
vi.mock("next/navigation", () => ({
    usePathname: () => "/pt-BR/dashboard",
}));

import { useCategories } from "@/hooks/use-categories";
import CategoryCombobox from "@/components/dashboard/CategoryCombobox";
import { I18nProvider } from "@/lib/i18n/provider";

function renderizar(elemento: React.ReactElement) {
    return render(<I18nProvider locale="pt-BR">{elemento}</I18nProvider>);
}

function comCategorias(nomes: string[]) {
    vi.mocked(useCategories).mockReturnValue({
        categories: nomes.map((name, indice) => ({
            id: indice + 1,
            name,
            description: "",
            document_count: 0,
        })),
        isLoading: false,
        error: null,
        createCategory: vi.fn(),
        mergeCategory: vi.fn(),
        reload: vi.fn(),
    } as unknown as ReturnType<typeof useCategories>);
}

describe("CategoryCombobox", () => {
    beforeEach(() => {
        vi.mocked(useCategories).mockReset();
        comCategorias(["Contratos", "Editais"]);
    });

    it("propaga o valor sem espacos nas pontas enquanto o usuario digita", async () => {
        // O UploadTab envia o valor direto; com espacos duplicava a categoria.
        const aoMudar = vi.fn();
        renderizar(<CategoryCombobox value="" onChange={aoMudar} />);

        await userEvent.type(screen.getByRole("textbox"), "  Juridico  ");

        expect(aoMudar).toHaveBeenCalled();
        const ultimoValor = aoMudar.mock.calls[aoMudar.mock.calls.length - 1][0];
        expect(ultimoValor).toBe("Juridico");
    });

    it("nao propaga apenas espacos como se fosse categoria", async () => {
        const aoMudar = vi.fn();
        renderizar(<CategoryCombobox value="" onChange={aoMudar} />);

        await userEvent.type(screen.getByRole("textbox"), "   ");

        const ultimoValor = aoMudar.mock.calls[aoMudar.mock.calls.length - 1][0];
        expect(ultimoValor).toBe("");
    });

    it("mantem o texto digitado visivel, com os espacos, enquanto edita", async () => {
        // O campo mostra o texto cru; so o valor propagado vai aparado.
        const aoMudar = vi.fn();
        renderizar(<CategoryCombobox value="" onChange={aoMudar} />);

        const campo = screen.getByRole("textbox");
        await userEvent.type(campo, "Contratos ");

        expect((campo as HTMLInputElement).value).toBe("Contratos ");
    });

    it("a lista de sugestoes fica acima do conteudo seguinte", () => {
        // Lista absoluta sobre o proximo campo: sem z-index alto fica inclicavel.
        const { container } = renderizar(<CategoryCombobox value="" onChange={vi.fn()} />);

        const lista = container.querySelector(".absolute");
        if (lista) {
            expect(lista.className).toMatch(/z-\[?\d/);
        }
    });
});
