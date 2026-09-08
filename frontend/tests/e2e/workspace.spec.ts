import { expect, test, type BrowserContext } from "@playwright/test";
import { resolve } from "node:path";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

async function authenticate(context: BrowserContext, baseURL: string | undefined) {
    await context.route("**/api/user/me", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    id: 1,
                    name: "Playwright",
                    email: "playwright@example.com",
                    avatarUrl: null,
                    updatedAt: "2026-01-01T00:00:00Z",
                })
            ),
        });
    });

    await context.addCookies([
        {
            name: "token",
            value: "playwright-token",
            url: baseURL ?? "http://127.0.0.1:3000",
        },
    ]);
}

const COLECOES = [
    { id: 7, name: "Processo XYZ", document_count: 2, metadata: {} },
];

const CONTEXTO = {
    collection: COLECOES[0],
    documents: [
        { id: 1, title: "Contrato SLA", status: "INDEXED" },
        { id: 2, title: "Edital de Pregao", status: "INDEXED" },
    ],
    outline_nodes: [
        {
            id: 11,
            node_id: "sla#penalidades",
            document_id: 1,
            sequence_order: 1,
            level: 1,
            title: "2 PENALIDADES",
            summary: "Multa de 5 por cento sobre a fatura mensal.",
        },
        {
            id: 12,
            node_id: "sla#disponibilidade",
            document_id: 1,
            sequence_order: 0,
            level: 1,
            title: "1 DISPONIBILIDADE",
            summary: "Uptime de 99,5 por cento.",
        },
        {
            id: 21,
            node_id: "edital#penalidades",
            document_id: 2,
            sequence_order: 1,
            level: 1,
            title: "2 DAS PENALIDADES",
            summary: "Multa por descumprimento.",
        },
    ],
    paragraph_nodes: [],
    table_sections: [],
    metadata_pairs: [],
    document_pages: [],
    chunks: [],
};

/** Marcas e legenda vazias: sem isto as rotas batem na API real com token falso. */
async function mockarAnotacoesVazias(context: BrowserContext) {
    await context.route("**/api/documents/collections/*/highlights**", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ highlights: [] })) });
    });
    await context.route("**/api/documents/collections/*/legend", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ labels: {}, colors: [] })) });
    });
    await context.route("**/api/documents/collections/*/view-state", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: {} })) });
    });
}

async function mockarColecao(context: BrowserContext) {
    await context.route("**/api/collections", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success(COLECOES)),
        });
    });

    await context.route("**/api/collections/7/context**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success(CONTEXTO)),
        });
    });
    await mockarAnotacoesVazias(context);
}

test.use({ locale: "pt-BR" });

test("workspace mostra a estrutura do documento e justifica cada conexao @smoke", async ({
    baseURL,
    context,
    page,
}) => {
    await authenticate(context, baseURL);
    await mockarColecao(context);

    await context.route("**/api/documents/collections/7/connections**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    node_id: "sla#penalidades",
                    min_score: 2,
                    connections: [
                        {
                            document_id: 2,
                            node_id: "edital#penalidades",
                            title: "2 DAS PENALIDADES",
                            summary: "Multa por descumprimento.",
                            start_page: 1,
                            score: 8,
                            matched_terms: ["multa", "fatura", "descumprimento"],
                        },
                    ],
                })
            ),
        });
    });

    await page.goto("/pt-BR/workspace");

    // Um cartao por documento da colecao.
    await expect(page.getByRole("button", { name: /Contrato SLA/ })).toBeVisible();
    await expect(page.getByRole("button", { name: /Edital de Pregao/ })).toBeVisible();

    // Focar o documento abre a sidebar com a estrutura JA INDEXADA -- sem
    // nenhuma chamada de modelo de linguagem no caminho.
    await page.getByRole("button", { name: /Contrato SLA/ }).click();
    await expect(page.getByText("2 PENALIDADES")).toBeVisible();
    await expect(page.getByText(/Multa de 5 por cento/)).toBeVisible();

    await page.getByRole("button", { name: /conex/i }).first().click();

    // A conexao precisa vir acompanhada dos termos que a justificam: uma aresta
    // sem explicacao seria tao opaca quanto a vizinhanca vetorial que este
    // projeto evita.
    // Escopo na sidebar: a linha SVG tambem repete titulo e termos no <title>.
    const sidebar = page.getByLabel("Detalhes do documento");
    await expect(sidebar.getByText("2 DAS PENALIDADES")).toBeVisible();
    await expect(sidebar.getByText(/multa/)).toBeVisible();
    // E a conexao existe como linha desenhada, com o score em termos.
    await expect(page.getByTestId("linha-de-conexao")).toHaveCount(1);
    await expect(page.getByTestId("rotulo-da-linha")).toHaveText("8 termos");
});

test("workspace declara ausencia de conexao em vez de area vazia @smoke", async ({
    baseURL,
    context,
    page,
}) => {
    await authenticate(context, baseURL);
    await mockarColecao(context);

    // Nenhuma conexao supera o limiar. Esta e a resposta CORRETA quando nao ha
    // relacao real -- e o oposto do falso positivo de 1,000 medido nos seis
    // bracos do experimento, onde nenhum sabia se calar.
    await context.route("**/api/documents/collections/7/connections**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    node_id: "sla#disponibilidade",
                    min_score: 2,
                    connections: [],
                })
            ),
        });
    });

    await page.goto("/pt-BR/workspace");
    await page.getByRole("button", { name: /Contrato SLA/ }).click();
    await page.getByRole("button", { name: /conex/i }).first().click();

    // Area em branco seria lida como falha de carregamento; a ausencia precisa
    // ser dita com todas as letras.
    await expect(page.getByText(/Nenhuma conexão acima do limiar/i)).toBeVisible();
});

test("workspace nao esmaga o painel em viewport estreito", async ({
    baseURL,
    context,
    page,
}) => {
    // Regressao medida no navegador: com sidebar de largura fixa ao lado de um
    // canvas flex-1, num viewport de 780px o canvas ficava com 72px e os
    // cartoes viravam tarjas ilegiveis atras da sidebar.
    await authenticate(context, baseURL);
    await mockarColecao(context);
    await page.setViewportSize({ width: 780, height: 900 });

    await page.goto("/pt-BR/workspace");

    const canvas = page.getByTestId("workspace-canvas");
    await expect(canvas).toBeVisible();

    const caixa = await canvas.boundingBox();
    expect(caixa).not.toBeNull();
    // O limite de 300px e folgado de proposito: o valor exato depende de
    // padding do shell. O que este teste impede e o colapso para dezenas de
    // pixels, que foi o defeito real.
    expect(caixa!.width).toBeGreaterThan(300);
});

// ---------------------------------------------------------------------------
// Quadros de PDF: paginas reais, grifos por bounding box, marcadores e aviso.
//
// A fixture uma-pagina.pdf foi gerada por tests/e2e/fixtures/gerar.py e as
// caixas abaixo sao as que `pdftotext -bbox-layout` mede nela. Nao sao
// inventadas: o teste de alinhamento compara o grifo com a pagina renderizada.
// ---------------------------------------------------------------------------

const FIXTURE_UMA_PAGINA = resolve(__dirname, "fixtures/uma-pagina.pdf");

function box(xMin: number, xMax: number, yMin: number, yMax: number) {
    return { x_min: xMin, x_max: xMax, y_min: yMin, y_max: yMax, page_width: 612, page_height: 792 };
}

const CAIXAS = [
    box(72, 205.042, 81.948, 94.898),
    box(72, 343.48, 114.102, 124.277),
    box(72, 181.704, 161.948, 174.898),
    box(72, 388.085, 194.102, 204.277),
];

const CONTEXTO_COM_PDF = {
    ...CONTEXTO,
    // O PDF vem primeiro: os quadros abrem em coluna e so o que esta no viewport
    // e aberto (carregamento preguicoso). No terceiro lugar ele ficaria fora de
    // uma tela de 1280 px e nunca abriria -- que e o comportamento certo.
    documents: [
        { id: 3, title: "Termo assinado", status: "INDEXED", mime_type: "application/pdf", file_path: "docs/termo.pdf" },
        ...CONTEXTO.documents,
    ],
    outline_nodes: [
        ...CONTEXTO.outline_nodes,
        { id: 31, node_id: "termo#disponibilidade", document_id: 3, sequence_order: 0, level: 1, title: "1 DISPONIBILIDADE", slug: "disponibilidade", start_page: 1, metadata: { paragraph_start: 0 } },
        { id: 32, node_id: "termo#penalidades", document_id: 3, sequence_order: 1, level: 1, title: "2 PENALIDADES", slug: "penalidades", start_page: 1, metadata: { paragraph_start: 2 } },
    ],
    paragraph_nodes: [
        { id: 301, document_id: 3, sequence_order: 0, page_number: 1, content: "1 DISPONIBILIDADE", metadata: { box: CAIXAS[0], source: "pdftotext", section_key: "disponibilidade" } },
        { id: 302, document_id: 3, sequence_order: 1, page_number: 1, content: "O fornecedor garante uptime de 99,5 por cento ao mes.", metadata: { box: CAIXAS[1], source: "pdftotext", section_key: "disponibilidade" } },
        { id: 303, document_id: 3, sequence_order: 2, page_number: 1, content: "2 PENALIDADES", metadata: { box: CAIXAS[2], source: "pdftotext", section_key: "penalidades" } },
        { id: 304, document_id: 3, sequence_order: 3, page_number: 1, content: "Multa de 5 por cento sobre a fatura mensal por descumprimento.", metadata: { box: CAIXAS[3], source: "pdftotext", section_key: "penalidades" } },
    ],
};

async function mockarColecaoComPdf(context: BrowserContext, contexto: unknown = CONTEXTO_COM_PDF) {
    await context.route("**/api/collections", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(COLECOES)) });
    });
    await context.route("**/api/collections/7/context**", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(contexto)) });
    });
    await mockarAnotacoesVazias(context);
    await context.route("**/api/documents/*/preview", async (route) => {
        await route.fulfill({ status: 200, contentType: "application/pdf", path: FIXTURE_UMA_PAGINA });
    });
}

test("quadro de PDF abre com a pagina real, grifos alinhados ao texto e marcadores de secao", async ({
    baseURL,
    context,
    page,
}) => {
    await authenticate(context, baseURL);
    await mockarColecaoComPdf(context);

    await page.goto("/pt-BR/workspace");
    const quadro = page.getByTestId("pdf-frame-3");
    await expect(quadro).toBeVisible();
    await expect(quadro).toHaveAttribute("data-paginas", "1", { timeout: 30_000 });

    const grifos = page.getByTestId("grifo");
    await expect(grifos).toHaveCount(4);

    // "Alinhado" e medida, nao adjetivo: o grifo fica em box * (tamanho da
    // pagina na tela / tamanho da pagina em pontos), com 1 px de tolerancia.
    const pagina = await page.getByTestId("pdf-pagina-3-1").boundingBox();
    const grifo = await grifos.first().boundingBox();
    expect(pagina).not.toBeNull();
    expect(grifo).not.toBeNull();
    const fx = pagina!.width / 612;
    const fy = pagina!.height / 792;
    expect(Math.abs(grifo!.x - pagina!.x - CAIXAS[0].x_min * fx)).toBeLessThanOrEqual(1);
    expect(Math.abs(grifo!.y - pagina!.y - CAIXAS[0].y_min * fy)).toBeLessThanOrEqual(1);
    expect(Math.abs(grifo!.width - (CAIXAS[0].x_max - CAIXAS[0].x_min) * fx)).toBeLessThanOrEqual(1);
    expect(Math.abs(grifo!.height - (CAIXAS[0].y_max - CAIXAS[0].y_min) * fy)).toBeLessThanOrEqual(1);

    // Um marcador por secao de nivel 1, ancorado no paragrafo em que ela comeca.
    const marcadores = page.getByTestId("marcador-secao");
    await expect(marcadores).toHaveCount(2);
    await expect(marcadores.nth(1)).toHaveText("2 PENALIDADES");
    const marcador = await marcadores.nth(1).boundingBox();
    expect(Math.abs(marcador!.y - pagina!.y - CAIXAS[2].y_min * fy)).toBeLessThanOrEqual(1);

    await expect(page.getByTestId("cabecalho-pagina")).toHaveText("p. 1 · pdftotext");
    await expect(grifos.nth(3)).toHaveAttribute("aria-label", "Parágrafo 4, página 1, seção 2 PENALIDADES");
});

test("clicar num grifo seleciona o trecho e 'Ver conexoes' desenha a linha ate o outro documento", async ({
    baseURL,
    context,
    page,
}) => {
    await authenticate(context, baseURL);
    await mockarColecaoComPdf(context);
    const pedidos: string[] = [];
    await context.route("**/api/documents/collections/7/connections**", async (route) => {
        pedidos.push(route.request().url());
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    node_id: "termo#penalidades",
                    min_score: 2,
                    connections: [
                        {
                            document_id: 1,
                            node_id: "sla#penalidades",
                            title: "2 PENALIDADES",
                            summary: "Multa de 5 por cento sobre a fatura mensal.",
                            start_page: 1,
                            score: 8,
                            matched_terms: ["multa", "fatura", "descumprimento"],
                        },
                    ],
                })
            ),
        });
    });

    await page.goto("/pt-BR/workspace");
    await expect(page.getByTestId("grifo")).toHaveCount(4);
    await page.getByTestId("grifo").nth(3).click();

    const sidebar = page.getByLabel("Detalhes do documento");
    await expect(sidebar.getByTestId("trecho-selecionado")).toContainText("Multa de 5 por cento");
    await expect(sidebar.getByTestId("trecho-selecionado")).toContainText("2 PENALIDADES");
    // O foco compartilhavel foi para a URL.
    await expect(page).toHaveURL(/doc=3/);

    await sidebar.getByTestId("trecho-selecionado").getByRole("button", { name: /Ver conexões/ }).click();

    // A consulta usou o no de sumario da secao do paragrafo, nao o paragrafo.
    expect(pedidos.some((url) => url.includes("node_id=termo%23penalidades"))).toBe(true);
    const linha = page.getByTestId("linha-de-conexao");
    await expect(linha).toHaveCount(1);
    await expect(page.getByTestId("rotulo-da-linha")).toHaveText("8 termos");
    // O destino e um cartao (nao PDF): a linha chega na borda dele, tracejada.
    await expect(linha.locator("path")).toHaveAttribute("stroke-dasharray", "6 4");
    await expect(sidebar.getByText(/multa, fatura, descumprimento/)).toBeVisible();
});

test("PDF indexado sem bounding box mostra o aviso e so reindexa ao clicar", async ({
    baseURL,
    context,
    page,
}) => {
    await authenticate(context, baseURL);
    const semBox = {
        ...CONTEXTO_COM_PDF,
        paragraph_nodes: CONTEXTO_COM_PDF.paragraph_nodes.map((no) => ({
            ...no,
            metadata: { source: "pdf_text", section_key: no.metadata.section_key },
        })),
    };
    await mockarColecaoComPdf(context, semBox);
    const tentativas: string[] = [];
    await context.route("**/api/documents/3/retry", async (route) => {
        tentativas.push(route.request().method());
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ id: 3, status: "QUEUED" })) });
    });

    await page.goto("/pt-BR/workspace");
    const aviso = page.getByTestId("aviso-sem-posicao");
    await expect(aviso).toContainText("4 trechos indexados sem posição");
    await expect(page.getByTestId("grifo")).toHaveCount(0);
    expect(tentativas).toEqual([]);

    await aviso.getByRole("button", { name: "Reindexar" }).click();
    await expect(aviso).toContainText("Na fila de indexação");
    expect(tentativas).toEqual(["POST"]);
});
