import { expect, test, type BrowserContext } from "@playwright/test";
import { resolve } from "node:path";

/**
 * Chat dentro da Area de trabalho: a fonte citada leva ao documento e a
 * pagina no canvas, e ?collection= escolhe a colecao. Tudo com mock.
 */

const UMA_PAGINA = resolve(__dirname, "fixtures/uma-pagina.pdf");
const TRINTA_PAGINAS = resolve(__dirname, "fixtures/trinta-paginas.pdf");

function success(data: unknown) {
    return { status: 200, type: "success", message: "ok", data };
}

function box(xMin: number, xMax: number, yMin: number, yMax: number) {
    return { x_min: xMin, x_max: xMax, y_min: yMin, y_max: yMax, page_width: 612, page_height: 792 };
}

const COLECOES = [
    { id: 7, name: "Processo XYZ", document_count: 2, metadata: {} },
    { id: 8, name: "Outra colecao", document_count: 0, metadata: {} },
];

const CONTEXTO = {
    collection: COLECOES[0],
    documents: [
        { id: 3, title: "Termo assinado", status: "INDEXED", mime_type: "application/pdf", file_path: "docs/termo.pdf" },
        { id: 4, title: "Edital longo", status: "INDEXED", mime_type: "application/pdf", file_path: "docs/edital.pdf" },
    ],
    outline_nodes: [
        { id: 42, node_id: "edital#prazos", document_id: 4, sequence_order: 1, level: 1, title: "12 PRAZOS", slug: "prazos", start_page: 12, metadata: { paragraph_start: 1 } },
    ],
    paragraph_nodes: [
        {
            id: 402,
            document_id: 4,
            sequence_order: 1,
            page_number: 12,
            content: "Pagina 12 de 30",
            metadata: {
                box: box(72, 205, 81.948, 94.898),
                source: "pdftotext",
                section_key: "prazos",
                // Duas linhas visuais dentro da caixa: a fonte do chat aponta a segunda.
                lines: [
                    { t: "linha um", b: [72, 81.948, 205, 88.4] },
                    { t: "linha dois", b: [72, 88.4, 205, 94.898] },
                ],
            },
        },
    ],
    table_sections: [],
    metadata_pairs: [],
    document_pages: [],
};

const CONTEXTO_VAZIO = { ...CONTEXTO, collection: COLECOES[1], documents: [], outline_nodes: [], paragraph_nodes: [] };

async function mockar(context: BrowserContext) {
    await context.route("**/api/user/me", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ id: 1, name: "Playwright", email: "p@x", avatarUrl: null, updatedAt: "2026-01-01T00:00:00Z" })) })
    );
    await context.route("**/api/user/providers/runtime", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ chat: { family: "LLM", activation_target: "CHAT", provider_type: "OPENAI", public_metadata: { model: "x" }, updated_at: "2026-01-01T00:00:00Z" } })) })
    );
    await context.route("**/api/collections", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(COLECOES)) }));
    await context.route("**/api/collections/7/context**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(CONTEXTO)) }));
    await context.route("**/api/collections/8/context**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(CONTEXTO_VAZIO)) }));
    await context.route("**/api/documents/3/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: UMA_PAGINA }));
    await context.route("**/api/documents/4/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: TRINTA_PAGINAS }));
    await context.route("**/api/documents/collections/*/highlights**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ highlights: [] })) }));
    await context.route("**/api/documents/collections/*/legend", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ labels: {}, colors: [] })) }));
    await context.route("**/api/documents/collections/*/view-state", (route) => {
        if (route.request().method() === "PUT") {
            return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: route.request().postDataJSON() })) });
        }
        // Edital folheado: "Ver no documento" tem de trocar a pagina.
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: { apresentacao: { "4": { modo: "pagina", paginaAtual: 1 } } } })) });
    });
    const criada = { id: 19, collection_id: 7, title: "Prazos", last_message_at: null, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" };
    await context.route("**/api/chat/conversations**", async (route) => {
        if (route.request().method() === "POST") {
            return route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify(success(criada)) });
        }
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success([])) });
    });
    // A conversa recem-criada e recarregada por id. Registrado DEPOIS da rota da
    // lista porque o Playwright da prioridade a ultima rota registrada.
    await context.route("**/api/chat/conversations/*", (route) =>
        route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(
                success({
                    conversation: criada,
                    // O que o backend persistiu: o historico usa snake_case.
                    messages: [
                        { id: 1, role: "user", content: "Qual e o prazo?", sources: [], created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" },
                        {
                            id: 2,
                            role: "assistant",
                            content: "O prazo e de 30 dias.",
                            sources: [
                                { id: "4:12:edital#prazos", doc_name: "Edital longo", page: 12, snippet: "Prazo de 30 dias.", document_id: 4, node_id: "edital#prazos", paragraph_id: 402, lines: [1] },
                                { id: "3:1", doc_name: "Termo assinado", page: 1, snippet: "Assinado." },
                            ],
                            created_at: "2026-01-01T00:00:01Z",
                            updated_at: "2026-01-01T00:00:01Z",
                        },
                    ],
                })
            ),
        })
    );
    await context.route("**/api/chat/stream", (route) =>
        route.fulfill({
            status: 200,
            contentType: "text/event-stream",
            body: [
                'event: stage\ndata: {"stage":"retrieving_context"}',
                // Uma fonte com documento e no (vai ao canvas) e uma antiga, sem documento (nao tem botao).
                'event: sources\ndata: {"sources":[{"id":"4:12:edital#prazos","docName":"Edital longo","page":12,"snippet":"Prazo de 30 dias.","documentId":4,"nodeId":"edital#prazos","paragraphId":402,"lines":[1]},{"id":"3:1","docName":"Termo assinado","page":1,"snippet":"Assinado."}]}',
                'event: delta\ndata: {"delta":"O prazo e de 30 dias."}',
                'event: done\ndata: {"stage":"completed","usage":{"promptTokens":1234,"completionTokens":56,"usageReported":true,"model":"gemma4:31b","estimatedCostUsd":0.00027}}',
                "",
            ].join("\n\n"),
        })
    );
}

test.use({ locale: "pt-BR" });

// A tela de abertura (SplashScreen) cobre a pagina por alguns segundos na
// primeira visita e engole o arrasto do mouse; marcar como ja vista a pula.
test.beforeEach(async ({ context }) => {
    await context.addInitScript(() => {
        window.localStorage.setItem("thingindex_last_splash", Date.now().toString());
    });
});

test("perguntar no painel e clicar em 'Ver no documento' leva o Edital a pagina citada", async ({ baseURL, context, page }) => {
    await mockar(context);
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace");

    const painel = page.getByTestId("painel-chat");
    await expect(painel).toBeVisible();
    // Com o painel aberto o Edital fica fora da tela e ainda nao foi medido:
    // "Ver no documento" tem de centralizar E trocar a pagina mesmo assim.
    const edital = page.getByTestId("pdf-frame-4");
    await expect(edital).toBeAttached();
    await painel.getByRole("textbox", { name: "Mensagem do chat" }).fill("Qual e o prazo?");
    await painel.getByRole("textbox", { name: "Mensagem do chat" }).press("Enter");
    await expect(painel.getByText("O prazo e de 30 dias.")).toBeVisible();
    // Tokens, modelo e custo embaixo da resposta (o artigo compara custo por consulta).
    await expect(painel.getByTestId("consumo-da-resposta").first()).toContainText("1.290 tokens · gemma4:31b · US$ 0,00027");
    // So a fonte com documentId tem o botao (a do Termo, sem document_id, nao).
    const botao = painel.getByTestId("ver-no-documento").first();
    await expect(botao).toBeVisible();
    await expect(painel.locator("details", { hasText: "Termo assinado" }).first().getByTestId("ver-no-documento")).toHaveCount(0);

    await botao.click();
    await expect(edital.getByTestId("pagina-de")).toHaveText("p. 12 de 30", { timeout: 30_000 });
    await expect(page.getByRole("complementary", { name: "Detalhes do documento" })).toContainText("Edital longo");
    // A fonte clicada fica marcada como ativa e o contador diz onde o usuario esta na lista.
    await expect(painel.getByTestId("fonte-posicao").first()).toHaveText("1/2");
    await expect(painel.getByTestId("fonte").first()).toHaveAttribute("data-ativa", "true");
    // O paragrafo citado fica selecionado e so a linha apontada pela fonte recebe o grifo forte.
    await expect(edital.locator('[data-paragrafo="402"]')).toHaveAttribute("aria-pressed", "true");
    await expect(edital.getByTestId("grifo-linha")).toHaveCount(1);
    await expect(edital.getByTestId("grifo-linha")).toHaveAttribute("title", "linha dois");
});

test("?collection= escolhe a colecao e o botao esconde o painel", async ({ baseURL, context, page }) => {
    await mockar(context);
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace?collection=8");

    const seletor = page.locator("header select").first();
    await expect(seletor).toHaveValue("8");
    // Trocar no seletor escreve a URL, para o link ser copiavel.
    await seletor.selectOption("7");
    await expect(page).toHaveURL(/collection=7/);
    await expect(page.getByTestId("pdf-frame-3")).toBeAttached();
    await expect(page.getByTestId("painel-chat")).toBeVisible();
    // Arrastar a divisa muda a largura do painel (280 a 640 px). O aria-pressed
    // do botao so fica true depois da hidratacao: e o sinal de que os handlers
    // do ponteiro ja existem.
    await expect(page.getByTestId("alternar-chat")).toHaveAttribute("aria-pressed", "true");
    const antes = (await page.getByTestId("painel-chat").boundingBox())!.width;
    const divisa = (await page.getByTestId("redimensionar-painel").boundingBox())!;
    await page.mouse.move(divisa.x + divisa.width / 2, divisa.y + 200);
    await page.mouse.down();
    await page.mouse.move(divisa.x + divisa.width / 2 + 120, divisa.y + 200, { steps: 6 });
    await page.mouse.up();
    const depois = (await page.getByTestId("painel-chat").boundingBox())!.width;
    expect(Math.round(depois - antes)).toBe(120);
    // O botao no proprio painel fecha; o do cabecalho reabre.
    await page.getByTestId("fechar-painel").click();
    await expect(page.getByTestId("painel-chat")).toHaveCount(0);
    await page.getByTestId("alternar-chat").click();
    await expect(page.getByTestId("painel-chat")).toBeVisible();
    await page.getByTestId("alternar-chat").click();
    await expect(page.getByTestId("painel-chat")).toHaveCount(0);
});
