import { expect, test, type BrowserContext } from "@playwright/test";
import { resolve } from "node:path";

/**
 * Navegar entre conexoes a partir do grifo colorido, estado de visualizacao
 * restaurado do servidor e marcadores escondidos. Tudo com mock; a stack real
 * esta em tests/e2e-real/estado.spec.ts.
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
        { id: 42, node_id: "edital#prazos", document_id: 4, sequence_order: 1, level: 1, title: "12 PRAZOS", slug: "prazos", start_page: 12, metadata: { paragraph_start: 1 } },
    ],
    paragraph_nodes: [
        { id: 303, document_id: 3, sequence_order: 2, page_number: 1, content: "2 PENALIDADES", metadata: { box: box(72, 181.704, 161.948, 174.898), source: "pdftotext", section_key: "penalidades" } },
        { id: 304, document_id: 3, sequence_order: 3, page_number: 1, content: "Multa de 5 por cento sobre a fatura mensal por descumprimento.", metadata: { box: box(72, 388.085, 194.102, 204.277), source: "pdftotext", section_key: "penalidades" } },
        { id: 401, document_id: 4, sequence_order: 0, page_number: 7, content: "Pagina 7 de 30", metadata: { box: box(72, 205, 81.948, 94.898), source: "pdftotext", section_key: "multas" } },
        { id: 402, document_id: 4, sequence_order: 1, page_number: 12, content: "Pagina 12 de 30", metadata: { box: box(72, 205, 81.948, 94.898), source: "pdftotext", section_key: "prazos" } },
    ],
    table_sections: [],
    metadata_pairs: [],
    document_pages: [],
};

type Gravado = { metodo: string; url: string; corpo: unknown };

async function mockar(context: BrowserContext, opcoes: { estado?: unknown; gravados?: Gravado[]; conexoes?: string[] } = {}) {
    const gravados = opcoes.gravados ?? [];
    const conexoes = opcoes.conexoes ?? [];
    await context.route("**/api/user/me", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ id: 1, name: "Playwright", email: "p@x", avatarUrl: null, updatedAt: "2026-01-01T00:00:00Z" })) })
    );
    await context.route("**/api/collections", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success([CONTEXTO.collection])) }));
    await context.route("**/api/collections/7/context**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(CONTEXTO)) }));
    await context.route("**/api/documents/3/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: UMA_PAGINA }));
    await context.route("**/api/documents/4/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: TRINTA_PAGINAS }));
    await context.route("**/api/documents/collections/7/highlights", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ highlights: [{ paragraph_node_id: 304, document_id: 3, color: "verde" }] })) })
    );
    await context.route("**/api/documents/collections/7/legend", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ labels: {}, colors: [] })) }));
    await context.route("**/api/documents/collections/7/view-state", async (route) => {
        if (route.request().method() === "PUT") {
            gravados.push({ metodo: "PUT", url: route.request().url(), corpo: route.request().postDataJSON() });
            return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: route.request().postDataJSON() })) });
        }
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: opcoes.estado ?? {} })) });
    });
    await context.route("**/api/documents/collections/7/connections**", (route) => {
        conexoes.push(route.request().url());
        return route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    node_id: "termo#penalidades",
                    min_score: 2,
                    connections: [
                        { document_id: 4, node_id: "edital#multas", title: "7 MULTAS", summary: "", start_page: 7, score: 3, matched_terms: ["multa"] },
                        { document_id: 4, node_id: "edital#prazos", title: "12 PRAZOS", summary: "", start_page: 12, score: 2, matched_terms: ["prazo"] },
                    ],
                })
            ),
        });
    });
    return { gravados, conexoes };
}

test.use({ locale: "pt-BR" });

test("clicar no grifo colorido busca as conexoes sozinho e o navegador leva ate cada destino", async ({ baseURL, context, page }) => {
    const { conexoes } = await mockar(context, { estado: { apresentacao: { "4": { modo: "pagina", paginaAtual: 1 } } } });
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace");

    const edital = page.getByTestId("pdf-frame-4");
    await expect(edital).toHaveAttribute("data-modo", "pagina");
    const grifoVerde = page.locator('[data-testid="grifo"][data-cor="verde"]');
    await expect(grifoVerde).toHaveCount(1);
    await grifoVerde.click();

    // Sem clicar em "Ver conexoes".
    await expect.poll(() => conexoes.length).toBe(1);
    expect(conexoes[0]).toContain("node_id=termo%23penalidades");
    await expect(page.getByTestId("conexao-i-de-n")).toHaveText("1 de 2");

    // Primeiro passo pela barra flutuante (esta a vista, junto do grifo).
    await page.getByTestId("barra-de-grifo").getByRole("button", { name: "Próxima conexão" }).click();
    await expect(page.getByTestId("conexao-i-de-n")).toHaveText("2 de 2");
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 12 de 30", { timeout: 30_000 });
    const atual = page.getByTestId("conexao-atual");
    await expect(atual).toContainText("12 PRAZOS");
    // Depois de centralizar o destino, a barra saiu da tela: os mesmos ◀ ▶
    // existem na sidebar, que esta sempre a vista.
    await expect(atual.getByRole("button", { name: "Próxima conexão" })).toBeDisabled();
    await atual.getByRole("button", { name: "Conexão anterior" }).click();
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 7 de 30");
    await expect(atual).toContainText("7 MULTAS");
});

test("o estado salvo no servidor e aplicado ao abrir, e esconder marcadores salva de volta", async ({ baseURL, context, page }) => {
    const { gravados } = await mockar(context, {
        estado: { viewport: { x: 30, y: 30, escala: 0.75 }, apresentacao: { "4": { modo: "pagina", paginaAtual: 4 } }, marcadoresVisiveis: true },
    });
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace");

    await expect(page.getByRole("toolbar", { name: "Zoom do canvas" })).toContainText("75%");
    const edital = page.getByTestId("pdf-frame-4");
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 4 de 30", { timeout: 30_000 });
    await expect(page.getByTestId("marcador-secao").first()).toBeVisible();

    await page.getByTestId("alternar-marcadores").click();
    await expect(page.getByTestId("marcador-secao")).toHaveCount(0);
    await expect.poll(() => gravados.length, { timeout: 5000 }).toBeGreaterThan(0);
    const ultimo = gravados.at(-1)!.corpo as { state: { marcadoresVisiveis: boolean; apresentacao: Record<string, unknown> } };
    expect(ultimo.state.marcadoresVisiveis).toBe(false);
    expect(ultimo.state.apresentacao["4"]).toEqual({ modo: "pagina", paginaAtual: 4 });
});
