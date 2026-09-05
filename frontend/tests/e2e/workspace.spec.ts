import { expect, test, type BrowserContext } from "@playwright/test";

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
    await expect(page.getByText("2 DAS PENALIDADES")).toBeVisible();
    await expect(page.getByText(/multa/)).toBeVisible();
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
