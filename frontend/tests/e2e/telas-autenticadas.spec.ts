import { expect, test, type BrowserContext } from "@playwright/test";

/**
 * Varredura de fumaca das telas autenticadas.
 *
 * Nasceu de uma caçada manual no Chromium: as telas foram percorridas uma a uma
 * procurando erro de console, requisição falhando em silêncio e conteúdo vazio.
 * A varredura manual não achou defeito de aplicação, mas o custo de repeti-la a
 * cada mudança é alto -- então virou teste.
 *
 * O que este teste pega que os outros não pegam: os testes E2E existentes
 * exercitam uma tela por vez com rotas mockadas. Aqui a questão é outra --
 * *toda* tela do menu ainda monta sem explodir depois de uma mudança
 * transversal (i18n, AppShell, contrato de API).
 */

const TELAS = [
    { nome: "dashboard", caminho: "/pt-BR/dashboard", titulo: /dashboard/i },
    { nome: "documentos", caminho: "/pt-BR/documents", titulo: /documentos/i },
    { nome: "area de trabalho", caminho: "/pt-BR/workspace", titulo: /área de trabalho/i },
    { nome: "configuracoes", caminho: "/pt-BR/settings", titulo: /configura/i },
    { nome: "custos", caminho: "/pt-BR/cost-visibility", titulo: /custos/i },
];

function sucesso(data: unknown) {
    return { status: 200, type: "success", message: "ok", data };
}

test.describe("varredura das telas autenticadas", () => {
    // O cookie precisa acompanhar a baseURL: fixar host/porta faz o cookie ser
    // descartado quando a suite roda em outra porta, e todo teste autenticado
    // cai na tela de login sem dizer por que.
    async function autenticar(context: BrowserContext, baseURL: string | undefined) {
        await context.addCookies([
            {
                name: "token",
                value: "token-de-teste",
                url: baseURL ?? "http://127.0.0.1:3000",
            },
        ]);
    }

    test.beforeEach(async ({ context, baseURL }) => {
        await autenticar(context, baseURL);

        // Respostas mínimas e válidas: o objetivo é provar que a tela monta,
        // não exercitar o backend.
        await context.route("**/api/user/me", (rota) =>
            rota.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify(sucesso({ id: 1, name: "Teste", email: "t@t.local" })),
            }),
        );

        await context.route("**/api/**", (rota) => {
            const url = rota.request().url();
            if (url.includes("/api/user/me")) {
                return rota.fallback();
            }
            return rota.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify(sucesso([])),
            });
        });
    });

    for (const tela of TELAS) {
        test(`${tela.nome} monta sem erro de console`, async ({ page }) => {
            const errosDeConsole: string[] = [];
            page.on("console", (msg) => {
                if (msg.type() === "error") {
                    errosDeConsole.push(msg.text());
                }
            });
            page.on("pageerror", (erro) => {
                errosDeConsole.push(String(erro));
            });

            await page.goto(tela.caminho, { waitUntil: "domcontentloaded" });

            // A tela precisa renderizar conteúdo de verdade. O limiar existe
            // porque uma página que falha ao montar ainda devolve 200 com o
            // esqueleto do layout -- só o corpo vem vazio.
            const texto = await page.locator("body").innerText();
            expect(texto.trim().length, `${tela.nome} renderizou praticamente nada`).toBeGreaterThan(200);

            expect(
                errosDeConsole,
                `${tela.nome} logou erro no console: ${errosDeConsole.join(" | ")}`,
            ).toHaveLength(0);
        });
    }

    test("o menu lateral leva a todas as telas registradas", async ({ page }) => {
        // Rota adicionada ao AppShell mas sem página, ou página sem entrada no
        // menu, são os dois lados do mesmo defeito -- e nenhum aparece em teste
        // de componente isolado.
        await page.goto("/pt-BR/dashboard", { waitUntil: "domcontentloaded" });

        const destinos = await page
            .locator("nav a[href], aside a[href]")
            .evaluateAll((elementos) =>
                elementos
                    .map((el) => el.getAttribute("href") || "")
                    .filter((href) => href.startsWith("/pt-BR/")),
            );

        expect(destinos.length, "nenhum link de navegação encontrado").toBeGreaterThan(0);

        for (const destino of new Set(destinos)) {
            const resposta = await page.goto(destino, { waitUntil: "domcontentloaded" });
            expect(resposta?.status(), `${destino} respondeu ${resposta?.status()}`).toBeLessThan(400);
        }
    });
});
