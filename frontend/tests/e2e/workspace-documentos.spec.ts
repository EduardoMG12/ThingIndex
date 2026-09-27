import { expect, test, type BrowserContext } from "@playwright/test";
import { resolve } from "node:path";

/**
 * Aba Documentos da Area de trabalho: apagar tira o quadro do canvas e o
 * upload pelo painel faz o quadro novo aparecer quando a indexacao termina,
 * sem recarregar a pagina. Tudo com mock.
 */

const UMA_PAGINA = resolve(__dirname, "fixtures/uma-pagina.pdf");

function success(data: unknown) {
    return { status: 200, type: "success", message: "ok", data };
}

const COLECAO = { id: 7, name: "Processo XYZ", document_count: 2, metadata: {} };

function documento(id: number, title: string, status: string) {
    return { id, title, status, mime_type: "application/pdf", file_path: `docs/${id}.pdf` };
}

async function mockar(context: BrowserContext) {
    const estado = {
        documentos: [documento(3, "Termo assinado", "INDEXED"), documento(4, "Edital longo", "INDEXED")],
        apagados: [] as string[],
        leiturasDoContexto: 0,
        enviados: 0,
    };
    await context.route("**/api/user/me", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ id: 1, name: "Playwright", email: "p@x", avatarUrl: null, updatedAt: "2026-01-01T00:00:00Z" })) })
    );
    await context.route("**/api/user/providers/runtime", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({})) }));
    await context.route("**/api/chat/conversations**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success([])) }));
    await context.route("**/api/collections", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success([COLECAO])) }));
    await context.route("**/api/collections/7/context**", (route) => {
        estado.leiturasDoContexto += 1;
        return route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({ collection: COLECAO, documents: estado.documentos, outline_nodes: [], paragraph_nodes: [], table_sections: [], metadata_pairs: [], document_pages: [] })),
        });
    });
    await context.route("**/api/documents/*/preview", (route) => route.fulfill({ status: 200, contentType: "application/pdf", path: UMA_PAGINA }));
    await context.route("**/api/documents/collections/*/highlights**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ highlights: [] })) }));
    await context.route("**/api/documents/collections/*/legend", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ labels: {}, colors: [] })) }));
    await context.route("**/api/documents/collections/*/view-state", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ state: route.request().method() === "PUT" ? route.request().postDataJSON() : {} })) })
    );
    // Upload: o backend devolve o documento na fila; a proxima leitura do
    // contexto ja o traz como PROCESSING e a seguinte como INDEXED.
    await context.route("**/api/documents/upload", (route) => {
        estado.enviados += 1;
        estado.documentos.push(documento(9, "Calendario", "PROCESSING"));
        const leituraDoEnvio = estado.leiturasDoContexto;
        const promover = () => {
            if (estado.leiturasDoContexto > leituraDoEnvio) {
                estado.documentos = estado.documentos.map((d) => (d.id === 9 ? { ...d, status: "INDEXED" } : d));
            } else {
                setTimeout(promover, 200);
            }
        };
        setTimeout(promover, 200);
        return route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify(success({ id: 9, user_id: 1, title: "Calendario", status: "QUEUED", collection: { id: 7, name: "Processo XYZ" } })) });
    });
    await context.route("**/api/documents/9", (route) =>
        route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success({ id: 9, user_id: 1, title: "Calendario", status: estado.documentos.find((d) => d.id === 9)?.status ?? "QUEUED", collection: { id: 7, name: "Processo XYZ" } })) })
    );
    await context.route("**/api/documents/3", (route) => {
        if (route.request().method() !== "DELETE") {
            return route.fallback();
        }
        estado.apagados.push(route.request().url());
        estado.documentos = estado.documentos.filter((d) => d.id !== 3);
        return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(success(null)) });
    });
    return estado;
}

test.use({ locale: "pt-BR" });

test("apagar pela aba Documentos tira o quadro do canvas sem recarregar", async ({ baseURL, context, page }) => {
    const estado = await mockar(context);
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace");
    await expect(page.getByTestId("pdf-frame-3")).toBeAttached();

    await page.getByTestId("aba-documentos").click();
    const item = page.getByTestId("documento-item-3");
    await expect(item).toContainText("Indexado");
    await item.getByRole("button", { name: "Apagar" }).click();
    await page.getByRole("textbox", { name: "Valor de confirmação" }).fill("Termo assinado");
    await page.getByRole("button", { name: "Confirmar exclusão" }).click();

    await expect.poll(() => estado.apagados.length).toBe(1);
    await expect(page.getByTestId("pdf-frame-3")).toHaveCount(0);
    await expect(page.getByTestId("pdf-frame-4")).toBeAttached();
    await expect(page.getByTestId("documento-item-3")).toHaveCount(0);
});

test("enviar pelo painel faz o quadro novo aparecer e o status virar Indexado sozinho", async ({ baseURL, context, page }) => {
    const estado = await mockar(context);
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto("/pt-BR/workspace");
    await page.getByTestId("aba-documentos").click();
    await page.getByTestId("enviar-documento").click();

    const modal = page.getByTestId("modal-de-upload");
    await modal.locator('input[type="file"]').setInputFiles(UMA_PAGINA);
    await modal.getByRole("button", { name: /Enviar 1 documento/ }).click();
    await expect.poll(() => estado.enviados).toBe(1);

    await expect(page.getByTestId("pdf-frame-9")).toBeAttached({ timeout: 15_000 });
    await expect(page.getByTestId("documento-item-9")).toContainText("Indexado", { timeout: 15_000 });
});
