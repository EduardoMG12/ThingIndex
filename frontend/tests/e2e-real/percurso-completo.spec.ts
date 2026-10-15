import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

// Percurso completo na stack real, sem mock. Cada etapa falha com mensagem
// propria. Precisa da stack de pe (./scripts/stack.sh dev).

const API = process.env.THINGINDEX_REAL_API ?? "http://localhost:8000/api/v1";
const SENHA = "SenhaForte#123";

// PDF do testdata do backend, estrutura conhecida (DECISAO / 1 RELATORIO / 2 FUNDAMENTACAO / 3 DISPOSITIVO).
const PDF_DE_TESTE = resolve(
    __dirname,
    "../../../api/internal/usecase/testdata/indexing-benchmark/court-order.pdf",
);

type Credenciais = { email: string; token: string };

async function registrarUsuario(): Promise<Credenciais> {
    const email = `e2ereal${Date.now()}@test.local`;

    const registro = await fetch(`${API}/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "E2E Real", email, password: SENHA }),
    });
    if (!registro.ok) {
        throw new Error(`etapa 1 (registro): API devolveu ${registro.status}. A stack esta de pe?`);
    }

    const login = await fetch(`${API}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password: SENHA }),
    });
    if (!login.ok) {
        throw new Error(`etapa 1 (login): API devolveu ${login.status}`);
    }

    const corpo = (await login.json()) as { data?: { token?: string } };
    const token = corpo.data?.token;
    if (!token) {
        throw new Error("etapa 1 (login): resposta sem token");
    }

    return { email, token };
}

async function enviarPDF(token: string, colecao: string): Promise<number> {
    const arquivo = readFileSync(PDF_DE_TESTE);
    const form = new FormData();
    // O titulo PRECISA carregar a extensao: resolveContentType faz sniffing a
    // partir dela, e sem o .pdf o upload e rejeitado com 400.
    form.append("file", new Blob([arquivo], { type: "application/pdf" }), "court-order.pdf");
    form.append("title", "court-order.pdf");
    form.append("collection", colecao);

    const resposta = await fetch(`${API}/documents/upload`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: form,
    });
    if (resposta.status !== 201) {
        throw new Error(`etapa 3 (upload): esperado 201, veio ${resposta.status}`);
    }

    const corpo = (await resposta.json()) as { data?: { id?: number } };
    const id = corpo.data?.id;
    if (!id) {
        throw new Error("etapa 3 (upload): resposta sem id de documento");
    }
    return id;
}

// Espera por condicao: o tempo de indexacao varia com a carga.
async function esperarIndexacao(token: string, documentoID: number): Promise<void> {
    for (let tentativa = 0; tentativa < 60; tentativa += 1) {
        const resposta = await fetch(`${API}/documents/${documentoID}`, {
            headers: { Authorization: `Bearer ${token}` },
        });
        if (resposta.ok) {
            const corpo = (await resposta.json()) as { data?: { status?: string } };
            const status = corpo.data?.status;
            if (status === "INDEXED") {
                return;
            }
            if (status === "FAILED") {
                throw new Error(`etapa 4 (indexacao): documento ${documentoID} terminou FAILED`);
            }
        }
        await new Promise((r) => setTimeout(r, 3000));
    }
    throw new Error(`etapa 4 (indexacao): documento ${documentoID} nao chegou a INDEXED em 180s`);
}

async function contarParagrafosComBox(token: string, colecao: string, documentoID: number): Promise<number> {
    const cabecalhos = { Authorization: `Bearer ${token}` };
    const lista = (await (await fetch(`${API}/documents/collections`, { headers: cabecalhos })).json()) as {
        data?: Array<{ id: number; name: string }>;
    };
    const alvo = lista.data?.find((c) => c.name === colecao);
    if (!alvo) {
        throw new Error(`etapa 6: collection ${colecao} nao esta na lista da API`);
    }
    const contexto = (await (await fetch(`${API}/documents/collections/${alvo.id}/context`, { headers: cabecalhos })).json()) as {
        data?: { paragraph_nodes?: Array<{ document_id: number; metadata?: { box?: unknown } }> };
    };
    return (contexto.data?.paragraph_nodes ?? []).filter((no) => no.document_id === documentoID && no.metadata?.box).length;
}

test.describe("percurso completo na stack real", () => {
    test("login, upload de PDF, indexacao, estrutura e conexoes", async ({ page, context }) => {
        const errosDeConsole: string[] = [];
        page.on("console", (msg) => {
            if (msg.type() === "error") {
                errosDeConsole.push(msg.text());
            }
        });

        // Etapa 1-2: usuario real e sessao pelo cookie que o middleware valida.
        //
        // A url do cookie e a baseURL fixa, e nao page.url(): antes da primeira
        // navegacao a pagina esta em branco e o Playwright recusa com
        // "Blank page can not have cookie".
        const { token } = await registrarUsuario();
        const baseURL = process.env.THINGINDEX_REAL_BASE_URL ?? "http://localhost:3988";
        await context.addCookies([{ name: "token", value: token, url: baseURL }]);

        // Etapa 3-4: upload e indexacao de verdade, pela API.
        const colecao = `E2EReal${Date.now()}`;
        const documentoID = await enviarPDF(token, colecao);
        await esperarIndexacao(token, documentoID);

        // Etapa 5: a aba Documentos do workspace lista o documento indexado.
        await page.goto("/pt-BR/workspace", { waitUntil: "domcontentloaded" });
        await page.locator("header select").first().selectOption({ label: colecao });
        await page.getByTestId("aba-documentos").click();
        await expect(
            page.getByTestId(`documento-item-${documentoID}`),
            "etapa 5: a aba Documentos nao listou o documento indexado",
        ).toContainText("Indexado", { timeout: 30_000 });

        // Etapa 6: grifos na tela = paragrafos com bounding box segundo a API.
        await expect(
            page.getByRole("heading", { level: 1 }),
            "etapa 6: o workspace nao montou",
        ).toBeVisible({ timeout: 30_000 });
        await expect(
            page.getByTestId(`pdf-frame-${documentoID}`),
            "etapa 6: o quadro do PDF nao apareceu",
        ).toHaveAttribute("data-paginas", /^[1-9]\d*$/, { timeout: 60_000 });

        const comBox = await contarParagrafosComBox(token, colecao, documentoID);
        expect(comBox, "etapa 6: a API nao devolveu paragrafo com box; o indexador tem pdftotext?").toBeGreaterThan(0);
        await expect(
            page.getByTestId("grifo"),
            "etapa 6: grifos na tela diferem dos paragrafos com box na API",
        ).toHaveCount(comBox, { timeout: 30_000 });

        // Etapa 6b: segundo PDF pelo painel aparece sem recarregar (via polling).
        await page.getByTestId("aba-documentos").click();
        await page.getByTestId("enviar-documento").click();
        const modal = page.getByTestId("modal-de-upload");
        await modal.locator('input[type="file"]').setInputFiles(PDF_DE_TESTE);
        await modal.getByRole("button", { name: /Enviar 1 documento/ }).click();
        await expect(
            page.locator('[data-testid^="pdf-frame-"]'),
            "etapa 6b: o segundo quadro nao apareceu depois do upload pelo painel",
        ).toHaveCount(2, { timeout: 120_000 });
        await modal.getByRole("button", { name: /Fechar/ }).click();
        // Etapa 7: nenhum erro de console no percurso.
        const relevantes = errosDeConsole.filter(
            (erro) => !erro.includes("favicon") && !erro.includes("Download the React DevTools"),
        );
        expect(relevantes, `etapa 7: erros de console no percurso: ${relevantes.join(" | ")}`)
            .toEqual([]);
    });

    test("a estrutura do PDF chega a API com nos e paginas", async () => {
        // Confere o conteudo do indice, nao so que a tela montou.
        const { token } = await registrarUsuario();
        const colecao = `E2EEstrutura${Date.now()}`;
        const documentoID = await enviarPDF(token, colecao);
        await esperarIndexacao(token, documentoID);

        const resposta = await fetch(`${API}/documents/${documentoID}/index`, {
            headers: { Authorization: `Bearer ${token}` },
        });
        expect(resposta.ok, `indice devolveu ${resposta.status}`).toBe(true);

        const corpo = (await resposta.json()) as {
            data?: {
                outline_nodes?: Array<{ node_id?: string; title?: string }>;
                document_pages?: Array<{ page_number?: number; content?: string }>;
            };
        };

        const nos = corpo.data?.outline_nodes ?? [];
        const paginas = corpo.data?.document_pages ?? [];

        expect(nos.length, "o PDF nao produziu nos de sumario").toBeGreaterThanOrEqual(3);
        expect(paginas.length, "o PDF nao produziu paginas").toBe(2);

        // node_id e o que permite pedir conexoes; ja sumiu uma vez.
        for (const no of nos) {
            expect(no.node_id, `no "${no.title}" veio sem node_id`).toBeTruthy();
        }

        // Pagina vazia em PDF com camada de texto significa extrator quebrado.
        for (const pagina of paginas) {
            expect(
                (pagina.content ?? "").trim().length,
                `pagina ${pagina.page_number} veio vazia`,
            ).toBeGreaterThan(0);
        }
    });
});
