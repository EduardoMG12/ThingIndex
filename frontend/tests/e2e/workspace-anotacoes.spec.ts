import { expect, test, type BrowserContext, type Route } from "@playwright/test";
import { resolve } from "node:path";

/**
 * Anotacoes do workspace: marcas coloridas, legenda, folhear e 'Ir para p. N'.
 * Tudo com mock; a stack real e coberta em tests/e2e-real/anotacoes.spec.ts.
 *
 * Fixtures geradas por tests/e2e/fixtures/gerar.py; as caixas sao as que
 * pdftotext -bbox-layout mede em uma-pagina.pdf.
 */

const UMA_PAGINA = resolve(__dirname, "fixtures/uma-pagina.pdf");
const TRINTA_PAGINAS = resolve(__dirname, "fixtures/trinta-paginas.pdf");

function success(data: unknown) {
    return { status: 200, type: "success", message: "ok", data };
}

function box(xMin: number, xMax: number, yMin: number, yMax: number) {
    return { x_min: xMin, x_max: xMax, y_min: yMin, y_max: yMax, page_width: 612, page_height: 792 };
}

const CONTEXTO = {
    collection: { id: 7, name: "Processo XYZ", document_count: 2, metadata: {} },
    documents: [
        { id: 3, title: "Termo assinado", status: "INDEXED", mime_type: "application/pdf", file_path: "docs/termo.pdf" },
        { id: 4, title: "Edital longo", status: "INDEXED", mime_type: "application/pdf", file_path: "docs/edital.pdf" },
    ],
    outline_nodes: [
        { id: 31, node_id: "termo#penalidades", document_id: 3, sequence_order: 0, level: 1, title: "2 PENALIDADES", slug: "penalidades", start_page: 1, metadata: { paragraph_start: 2 } },
        { id: 41, node_id: "edital#multas", document_id: 4, sequence_order: 0, level: 1, title: "7 MULTAS", slug: "multas", start_page: 7, metadata: { paragraph_start: 0 } },
    ],
    paragraph_nodes: [
        { id: 301, document_id: 3, sequence_order: 0, page_number: 1, content: "1 DISPONIBILIDADE", metadata: { box: box(72, 205.042, 81.948, 94.898), source: "pdftotext", section_key: "disponibilidade" } },
        { id: 303, document_id: 3, sequence_order: 2, page_number: 1, content: "2 PENALIDADES", metadata: { box: box(72, 181.704, 161.948, 174.898), source: "pdftotext", section_key: "penalidades" } },
        { id: 304, document_id: 3, sequence_order: 3, page_number: 1, content: "Multa de 5 por cento sobre a fatura mensal por descumprimento.", metadata: { box: box(72, 388.085, 194.102, 204.277), source: "pdftotext", section_key: "penalidades" } },
        { id: 401, document_id: 4, sequence_order: 0, page_number: 7, content: "Pagina 7 de 30", metadata: { box: box(72, 205, 81.948, 94.898), source: "pdftotext", section_key: "multas" } },
    ],
    table_sections: [],
    metadata_pairs: [],
    document_pages: [],
};

type Gravado = { metodo: string; url: string; corpo: unknown };

async function mockar(context: BrowserContext, opcoes: { marcas?: unknown[]; legenda?: Record<string, string>; gravados?: Gravado[] } = {}) {
    const gravados = opcoes.gravados ?? [];
    // A Area de trabalho agora embute o chat: sem estes mocks as buscas de
    // runtime e de conversas recebem 401 da API real e a pagina cai no login.
    await context.route("**/api/user/providers/runtime", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ status: 200, type: "success", message: "ok", data: {} }) }));
    await context.route("**/api/chat/conversations**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ status: 200, type: "success", message: "ok", data: [] }) }));
    await context.route("**/api/user/me", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ id: 1, name: "Playwright", email: "p@x", avatarUrl: null, updatedAt: "2026-01-01T00:00:00Z" })) })
    );
    await context.route("**/api/collections", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success([CONTEXTO.collection])) })
    );
    await context.route("**/api/collections/7/context**", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(CONTEXTO)) })
    );
    await context.route("**/api/documents/3/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: UMA_PAGINA }));
    await context.route("**/api/documents/4/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: TRINTA_PAGINAS }));
    await context.route("**/api/documents/collections/7/highlights", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ highlights: opcoes.marcas ?? [] })) })
    );
    const registrar = async (route: Route, resposta: unknown) => {
        const req = route.request();
        gravados.push({ metodo: req.method(), url: req.url(), corpo: req.postDataJSON() });
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(resposta)) });
    };
    await context.route("**/api/documents/collections/7/highlights/*", (route) =>
        registrar(route, route.request().method() === "DELETE" ? null : { paragraph_node_id: 304, document_id: 3, color: "verde" })
    );
    await context.route("**/api/documents/collections/7/legend", (route) => {
        if (route.request().method() === "GET") {
            return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ labels: opcoes.legenda ?? {}, colors: [] })) });
        }
        return registrar(route, { labels: (route.request().postDataJSON() as { labels: unknown }).labels, colors: [] });
    });
    await context.route("**/api/documents/collections/7/view-state", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: {} })) })
    );
    await context.route("**/api/documents/collections/7/connections**", (route) =>
        route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    node_id: "termo#penalidades",
                    min_score: 2,
                    connections: [{ document_id: 4, node_id: "edital#multas", title: "7 MULTAS", summary: "", start_page: 7, score: 3, matched_terms: ["multa", "fatura", "mensal"] }],
                })
            ),
        })
    );
    await context.addCookies([{ name: "token", value: "playwright-token", url: "http://127.0.0.1:3000" }]);
    return gravados;
}

test.use({ locale: "pt-BR" });

test("marcar com cor salva com PUT, 'sem cor' apaga com DELETE, e a marca carregada da API aparece colorida", async ({
    baseURL,
    context,
    page,
}) => {
    const gravados = await mockar(context);
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.goto("/pt-BR/workspace");
    // O painel de chat aberto tira o segundo quadro da tela; estes testes contam grifos nos dois.
    await page.getByTestId("alternar-chat").click();
    const grifos = page.getByTestId("grifo");
    await expect(grifos).toHaveCount(4);

    await grifos.nth(2).click();
    const barra = page.getByTestId("barra-de-grifo");
    await expect(barra).toBeVisible();
    await barra.getByTestId("chip-verde").click();
    await expect(grifos.nth(2)).toHaveAttribute("data-cor", "verde");
    await expect(grifos.nth(2)).toHaveAttribute("aria-label", /grifo Verde$/);
    await expect.poll(() => gravados.length).toBe(1);
    expect(gravados[0]).toMatchObject({ metodo: "PUT", corpo: { color: "verde" } });
    expect(gravados[0].url).toMatch(/\/highlights\/304$/);

    await barra.getByTestId("chip-sem-cor").click();
    await expect(grifos.nth(2)).not.toHaveAttribute("data-cor");
    await expect.poll(() => gravados.length).toBe(2);
    expect(gravados[1].metodo).toBe("DELETE");
});

test("marca vinda da API ja abre colorida e a legenda editada entra no aria-label", async ({ baseURL, context, page }) => {
    const gravados = await mockar(context, { marcas: [{ paragraph_node_id: 304, document_id: 3, color: "roxo" }], legenda: { roxo: "risco" } });
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.goto("/pt-BR/workspace");
    // O painel de chat aberto tira o segundo quadro da tela; estes testes contam grifos nos dois.
    await page.getByTestId("alternar-chat").click();
    const grifo = page.getByTestId("grifo").nth(2);
    await expect(grifo).toHaveAttribute("data-cor", "roxo");
    await expect(grifo).toHaveAttribute("aria-label", /grifo risco$/);

    // Editar o rotulo salva a legenda inteira e o aria-label muda sem recarregar.
    const campo = page.getByTestId("legenda-roxo");
    await expect(campo).toHaveValue("risco");
    await campo.fill("penalidade");
    await campo.press("Enter");
    await expect.poll(() => gravados.length).toBe(1);
    expect(gravados[0]).toMatchObject({ metodo: "PUT", corpo: { labels: { roxo: "penalidade" } } });
    await expect(grifo).toHaveAttribute("aria-label", /grifo penalidade$/);
});

test("'uma pagina' mostra so a pagina atual, e 'Ir para p. 7' troca a pagina do quadro destino", async ({ baseURL, context, page }) => {
    await mockar(context);
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace");
    // O painel de chat aberto tira o segundo quadro da tela; estes testes contam grifos nos dois.
    await page.getByTestId("alternar-chat").click();
    await page.getByRole("button", { name: "Ajustar à tela" }).click();
    const edital = page.getByTestId("pdf-frame-4");
    await expect(edital).toHaveAttribute("data-paginas", "30", { timeout: 30_000 });

    await edital.getByTestId("alternar-modo").click();
    await expect(edital).toHaveAttribute("data-modo", "pagina");
    await expect(edital.locator('[data-testid^="pdf-pagina-4-"]')).toHaveCount(1);
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 1 de 30");
    await expect(edital.getByRole("button", { name: "Página anterior" })).toBeDisabled();
    await edital.getByRole("button", { name: "Próxima página" }).click();
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 2 de 30");

    // Conexao do termo (p.1) para o edital, secao que comeca na p.7.
    const termo = page.getByTestId("pdf-frame-3");
    await termo.getByTestId("grifo").nth(2).click();
    await page.getByTestId("barra-de-grifo").getByRole("button", { name: /Ver conexões/ }).click();
    await expect(page.getByTestId("linha-de-conexao")).toHaveCount(1);
    const irPara = page.getByTestId("ir-para-pagina");
    await expect(irPara).toHaveText("Ir para p. 7");
    await irPara.click();
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 7 de 30");
    await expect(edital.getByTestId("pdf-pagina-4-7")).toBeVisible();
    // Na p.7 o paragrafo da secao destino esta em cor secundaria.
    await expect(edital.getByTestId("grifo")).toHaveCount(1);
});
