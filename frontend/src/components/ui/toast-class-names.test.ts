import { describe, expect, it } from "vitest";

import { TOAST_CLASS_NAMES } from "@/components/ui/toast-class-names";

function classes(valor: string | undefined) {
    return (valor ?? "").split(/\s+/).filter(Boolean);
}

describe("classes do toast (F-16)", () => {
    it("nenhuma variante fixa texto branco, que some no tema claro", () => {
        for (const [variante, valor] of Object.entries(TOAST_CLASS_NAMES)) {
            const fixas = classes(valor).filter((c) => /text-white|text-zinc-/.test(c));
            expect(fixas, variante).toEqual([]);
        }
    });

    it("o toast base usa os tokens do tema para fundo e texto", () => {
        const base = classes(TOAST_CLASS_NAMES.toast);
        expect(base).toContain("bg-card");
        expect(base).toContain("text-card-foreground");
        expect(base.some((c) => c.startsWith("bg-white"))).toBe(false);
    });

    it("a largura minima de 440px so vale a partir de sm, para caber em 390px", () => {
        const base = classes(TOAST_CLASS_NAMES.toast);
        expect(base).toContain("min-w-0");
        expect(base).toContain("sm:min-w-[440px]");
        expect(base).not.toContain("min-w-[440px]");
    });
});
