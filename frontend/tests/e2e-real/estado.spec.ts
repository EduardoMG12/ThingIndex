import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

/**
 * Estado de visualizacao na stack REAL: folhear ate a p. 2, fechar o contexto
 * do navegador, abrir outro com o mesmo usuario e reabrir em p. 2.
 */

const API = process.env.THINGINDEX_REAL_API ?? "http://localhost:8000/api/v1";
const SENHA = "SenhaForte#123";
const PDF = resolve(__dirname, "../../../api/internal/usecase/testdata/indexing-benchmark/court-order.pdf");

async function registrar(prefixo: string) {
    const email = `${prefixo}${Date.now()}@test.local`;
    const r = await fetch(`${API}/auth/register`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: prefixo, email, password: SENHA }) });
    if (!r.ok) throw new Error(`registro ${r.status}`);
    const login = (await (await fetch(`${API}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password: SENHA }) })).json()) as { data?: { token?: string } };
    if (!login.data?.token) throw new Error("login sem token");
    return login.data.token;
}

async function enviarEEsperar(token: string, colecao: string) {
    const form = new FormData();
    form.append("file", new Blob([readFileSync(PDF)], { type: "application/pdf" }), "court-order.pdf");
    form.append("title", "court-order.pdf");
    form.append("collection", colecao);
    const up = (await (await fetch(`${API}/documents/upload`, { method: "POST", headers: { Authorization: `Bearer ${token}` }, body: form })).json()) as { data?: { id?: number } };
    const id = up.data?.id;
    if (!id) throw new Error("upload sem id");
    for (let i = 0; i < 60; i += 1) {
        const doc = (await (await fetch(`${API}/documents/${id}`, { headers: { Authorization: `Bearer ${token}` } })).json()) as { data?: { status?: string } };
        if (doc.data?.status === "INDEXED") return id;
        if (doc.data?.status === "FAILED") throw new Error("indexacao FAILED");
        await new Promise((r) => setTimeout(r, 3000));
    }
    throw new Error("nao indexou em 180s");
}

test("folhear ate a p. 2 e reabrir em outro contexto volta na p. 2", async ({ browser }) => {
    test.setTimeout(240_000);
    const token = await registrar("estado");
    const colecao = `Estado${Date.now()}`;
    const documentoID = await enviarEEsperar(token, colecao);
    const baseURL = process.env.THINGINDEX_REAL_BASE_URL ?? "http://localhost:3988";

    const primeiro = await browser.newContext();
    await primeiro.addCookies([{ name: "token", value: token, url: baseURL }]);
    const page = await primeiro.newPage();
    await page.goto(`${baseURL}/pt-BR/workspace`, { waitUntil: "domcontentloaded" });
    await page.selectOption("select", { label: colecao });
    const quadro = page.getByTestId(`pdf-frame-${documentoID}`);
    await expect(quadro).toHaveAttribute("data-paginas", /^[1-9]\d*$/, { timeout: 60_000 });
    await quadro.getByTestId("alternar-modo").click();
    await quadro.getByRole("button", { name: "Próxima página" }).click();
    await expect(quadro.getByTestId("pagina-de")).toHaveText(/^p\. 2 de \d+$/);
    // Debounce de 800 ms + folga; o pagehide tambem envia, mas nao dependemos dele.
    await page.waitForTimeout(1500);
    await primeiro.close();

    const segundo = await browser.newContext();
    await segundo.addCookies([{ name: "token", value: token, url: baseURL }]);
    const outra = await segundo.newPage();
    await outra.goto(`${baseURL}/pt-BR/workspace`, { waitUntil: "domcontentloaded" });
    await outra.selectOption("select", { label: colecao });
    const reaberto = outra.getByTestId(`pdf-frame-${documentoID}`);
    await expect(reaberto).toHaveAttribute("data-modo", "pagina", { timeout: 60_000 });
    await expect(reaberto.getByTestId("pagina-de")).toHaveText(/^p\. 2 de \d+$/, { timeout: 60_000 });
    await segundo.close();
});
