import { expect, test, type BrowserContext } from "@playwright/test";
import { resolve } from "node:path";

/**
 * Orcamento de renderizacao: com tres PDFs de 30 paginas no canvas, nunca ha
 * mais de 12 <canvas> de pagina vivos, em qualquer posicao de rolagem. E o que
 * permite varios PDFs no workspace sem o navegador engasgar.
 *
 * Fixture trinta-paginas.pdf gerada por tests/e2e/fixtures/gerar.py.
 */

const FIXTURE = resolve(__dirname, "fixtures/trinta-paginas.pdf");
const TETO = 12;

function success(data: unknown) {
    return { status: 200, type: "success", message: "ok", data };
}

async function authenticate(context: BrowserContext, baseURL: string | undefined) {
    await context.route("**/api/user/me", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({ id: 1, name: "Playwright", email: "playwright@example.com", avatarUrl: null, updatedAt: "2026-01-01T00:00:00Z" })),
        });
    });
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
}

const DOCUMENTOS = [11, 12, 13];

test.use({ locale: "pt-BR" });

test("nunca ha mais de 12 canvases de pagina vivos, mesmo rolando tres PDFs de 30 paginas", async ({
    baseURL,
    context,
    page,
}) => {
    await authenticate(context, baseURL);
    await context.route("**/api/collections", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success([{ id: 9, name: "Tres PPCs", document_count: 3, metadata: {} }])) });
    });
    await context.route("**/api/collections/9/context**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    collection: { id: 9, name: "Tres PPCs", document_count: 3, metadata: {} },
                    documents: DOCUMENTOS.map((id) => ({ id, title: `PPC ${id}`, status: "INDEXED", mime_type: "application/pdf" })),
                    outline_nodes: [],
                    paragraph_nodes: [],
                    table_sections: [],
                    metadata_pairs: [],
                    document_pages: [],
                })
            ),
        });
    });
    await context.route("**/api/documents/collections/*/highlights**", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ highlights: [] })) });
    });
    await context.route("**/api/documents/collections/*/legend", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ labels: {} })) });
    });
    await context.route("**/api/documents/*/preview", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/pdf", path: FIXTURE });
    });

    // Viewport alto de proposito: com 1000 px so cabem ~9 paginas visiveis e o
    // teto nunca aperta -- o teste passava com teto 100 (provado por reversao).
    await page.setViewportSize({ width: 1600, height: 2000 });
    await page.goto("/pt-BR/workspace");
    // Os quadros so abrem quando entram no viewport; "Ajustar a tela" poe os
    // tres lado a lado (~0,46x, acima do piso de miniatura de 0,35x).
    await expect(page.getByTestId("pdf-frame-11")).toBeVisible();
    await page.getByRole("button", { name: "Ajustar à tela" }).click();
    for (const id of DOCUMENTOS) {
        await expect(page.getByTestId(`pdf-frame-${id}`)).toHaveAttribute("data-paginas", "30", { timeout: 30_000 });
    }
    // Abaixo de 35% o quadro mostra so a miniatura (sem placeholders de
    // pagina). Sobe o zoom ate passar do piso, mantendo varias paginas por
    // coluna visiveis -- e onde o teto realmente aperta.
    const zoomAtual = async () => Number((await page.getByRole("toolbar").locator("span").innerText()).replace("%", ""));
    for (let i = 0; i < 6 && (await zoomAtual()) < 40; i += 1) {
        await page.getByRole("button", { name: "Aproximar" }).click();
    }
    expect(await zoomAtual()).toBeGreaterThanOrEqual(40);
    // 90 placeholders de pagina, todos com o tamanho real.
    await expect(page.locator('[data-testid^="pdf-pagina-"]')).toHaveCount(90);

    const canvases = page.locator("canvas[data-pagina]");
    await expect.poll(async () => canvases.count()).toBeGreaterThan(0);

    const canvas = page.getByTestId("workspace-canvas");
    const caixa = await canvas.boundingBox();
    expect(caixa).not.toBeNull();
    let maximo = 0;
    for (let passo = 0; passo < 8; passo += 1) {
        // Pan pelo fundo: arrasta para cima para rolar os quadros.
        await page.mouse.move(caixa!.x + caixa!.width - 40, caixa!.y + caixa!.height - 80);
        await page.mouse.down();
        await page.mouse.move(caixa!.x + caixa!.width - 40, caixa!.y + 80, { steps: 6 });
        await page.mouse.up();
        await page.waitForTimeout(350);
        const vivos = await canvases.count();
        maximo = Math.max(maximo, vivos);
        expect(vivos, `passo ${passo}`).toBeLessThanOrEqual(TETO);
        expect(vivos, `passo ${passo}`).toBeGreaterThan(0);
    }
    // O teto foi exercitado de verdade: ha mais paginas visiveis do que o
    // orcamento permite, e o maximo observado bate exatamente nele.
    expect(maximo).toBe(TETO);
});
