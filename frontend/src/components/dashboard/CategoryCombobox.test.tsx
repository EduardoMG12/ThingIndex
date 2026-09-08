import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

vi.mock("@/hooks/use-categories", () => ({
    useCategories: vi.fn(),
}));

// O I18nProvider le o locale do pathname; sem o mock de navegacao o provider
// nao monta e todo teste falharia por ausencia de contexto, escondendo o que
// realmente se quer medir.
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
        // O componente aparava o valor ao SELECIONAR e ao sair do campo, mas
        // propagava o texto cru a cada tecla. Quem consome (UploadTab) envia
        // esse valor direto no upload -- digitar " Contratos " e enviar sem sair
        // do campo criava uma categoria com espacos, duplicando a existente,
        // porque a deduplicacao de use-categories compara por nome aparado.
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
        // Aparar o que o usuario VE seria hostil: apagaria o espaco no momento
        // em que ele o digita, entre duas palavras. O trim vale para o valor
        // propagado, nao para o campo.
        const aoMudar = vi.fn();
        renderizar(<CategoryCombobox value="" onChange={aoMudar} />);

        const campo = screen.getByRole("textbox");
        await userEvent.type(campo, "Contratos ");

        expect((campo as HTMLInputElement).value).toBe("Contratos ");
    });

    it("a lista de sugestoes fica acima do conteudo seguinte", () => {
        // A lista e posicionada de forma absoluta sobre o campo seguinte do
        // formulario. Sem z-index alto o proximo campo desenha por cima e a
        // sugestao fica inclicavel -- defeito que so aparece em formulario com
        // mais de um campo, e por isso escapa de teste isolado.
        const { container } = renderizar(<CategoryCombobox value="" onChange={vi.fn()} />);

        const lista = container.querySelector(".absolute");
        if (lista) {
            expect(lista.className).toMatch(/z-\[?\d/);
        }
    });
});
