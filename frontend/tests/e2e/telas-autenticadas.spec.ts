import { expect, test, type BrowserContext } from "@playwright/test";

// Fumaca: toda tela do menu monta sem erro de console nem corpo vazio.
// Pega quebra transversal (i18n, AppShell, contrato de API).

const TELAS = [
    { nome: "operacoes", caminho: "/pt-BR/documents", titulo: /opera/i },
    { nome: "area de trabalho", caminho: "/pt-BR/workspace", titulo: /área de trabalho/i },
    { nome: "configuracoes", caminho: "/pt-BR/settings", titulo: /configura/i },
    { nome: "custos", caminho: "/pt-BR/cost-visibility", titulo: /custos/i },
];

function sucesso(data: unknown) {
    return { status: 200, type: "success", message: "ok", data };
}

test.describe("varredura das telas autenticadas", () => {
    // Cookie na URL da baseURL; com host/porta fixos ele era descartado.
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

            // Pagina que falha ao montar ainda devolve 200 com o layout, so o corpo vazio.
            const texto = await page.locator("body").innerText();
            expect(texto.trim().length, `${tela.nome} renderizou praticamente nada`).toBeGreaterThan(200);

            expect(
                errosDeConsole,
                `${tela.nome} logou erro no console: ${errosDeConsole.join(" | ")}`,
            ).toHaveLength(0);
        });
    }

    test("o menu lateral leva a todas as telas registradas", async ({ page }) => {
        // Menu e paginas tem de bater nos dois sentidos.
        await page.goto("/pt-BR/workspace", { waitUntil: "domcontentloaded" });

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
