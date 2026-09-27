import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// O provider prioriza o locale da rota, entao o pathname define o idioma.
const { pathnameState } = vi.hoisted(() => ({ pathnameState: { value: "/en/rota-inexistente" } }));

vi.mock("next/navigation", () => ({
    usePathname: () => pathnameState.value,
    useRouter: () => ({ push: vi.fn(), replace: vi.fn(), refresh: vi.fn(), prefetch: vi.fn(), back: vi.fn() }),
}));

import { NotFoundContent } from "@/components/ui/NotFoundContent";
import { RegisterForm } from "@/components/auth/RegisterForm";
import { SkipToContentLink } from "@/components/ui/SkipToContentLink";
import { createTestWrapper } from "@/test/test-utils";

describe("strings visiveis passam pelo i18n (F-12)", () => {
    it("pagina 404 mostra titulo, descricao e botao no idioma ativo", () => {
        pathnameState.value = "/en/rota-inexistente";
        render(<NotFoundContent />, { wrapper: createTestWrapper("en") });

        expect(screen.getByRole("heading", { name: "Page not found" })).toBeInTheDocument();
        expect(screen.getByText(/does not exist or has been moved/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Back to home" })).toBeInTheDocument();
        expect(screen.queryByText("Página não encontrada")).not.toBeInTheDocument();
    });

    it("placeholders do cadastro seguem o idioma ativo", () => {
        pathnameState.value = "/es/register";
        render(<RegisterForm />, { wrapper: createTestWrapper("es") });

        expect(screen.getByPlaceholderText("Tu nombre")).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Mínimo 8 caracteres")).toBeInTheDocument();
        expect(screen.queryByPlaceholderText("Seu nome")).not.toBeInTheDocument();
    });

    it("skip link usa o texto do idioma ativo", () => {
        pathnameState.value = "/pt-BR/workspace";
        render(<SkipToContentLink />, { wrapper: createTestWrapper("pt-BR") });

        expect(screen.getByRole("link", { name: "Pular para o conteúdo principal" })).toHaveAttribute(
            "href",
            "#main-content"
        );
    });
});
