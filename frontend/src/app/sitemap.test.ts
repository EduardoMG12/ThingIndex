import { describe, expect, it } from "vitest";

import robots from "@/app/robots";
import sitemap from "@/app/sitemap";
import { PROTECTED_PATHS } from "@/lib/route-access";

describe("sitemap e robots so expoem paginas publicas (F-17)", () => {
    it("o sitemap nao lista nenhuma rota autenticada em nenhum locale", () => {
        const urls = sitemap().map((entrada) => new URL(entrada.url).pathname);

        for (const url of urls) {
            const semLocale = url.replace(/^\/(pt-BR|en|es)(?=\/|$)/, "") || "/";
            const protegida = PROTECTED_PATHS.some((rota) => semLocale === rota || semLocale.startsWith(`${rota}/`));
            expect(protegida, url).toBe(false);
        }
    });

    it("o sitemap lista inicio, login, cadastro e documentacao nos tres locales", () => {
        const urls = sitemap().map((entrada) => new URL(entrada.url).pathname).sort();

        expect(urls).toEqual(
            ["pt-BR", "en", "es"]
                .flatMap((locale) => [`/${locale}`, `/${locale}/login`, `/${locale}/register`, `/${locale}/documentation`])
                .sort()
        );
    });

    it("o robots bloqueia as rotas autenticadas com e sem prefixo de locale", () => {
        const regras = robots().rules;
        const regra = Array.isArray(regras) ? regras[0]! : regras;
        const disallow = ([] as string[]).concat(regra.disallow ?? []);

        for (const rota of PROTECTED_PATHS) {
            expect(disallow).toContain(rota);
            for (const locale of ["pt-BR", "en", "es"]) {
                expect(disallow).toContain(`/${locale}${rota}`);
            }
        }
        expect(disallow).toContain("/api/");
        expect(regra.allow).toBe("/");
    });
});
