import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

/**
 * Marcas coloridas na stack REAL: marcar, recarregar, continuar marcado; e o
 * isolamento entre usuarios provado pela API (o segundo usuario nao ve nada).
 * Pre-requisito: stack de pe.
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

async function idDaColecao(token: string, nome: string) {
    const lista = (await (await fetch(`${API}/documents/collections`, { headers: { Authorization: `Bearer ${token}` } })).json()) as { data?: Array<{ id: number; name: string }> };
    const alvo = lista.data?.find((c) => c.name === nome);
    if (!alvo) throw new Error(`colecao ${nome} nao encontrada`);
    return alvo.id;
}

test("marca amarela sobrevive ao reload e outro usuario nao a ve", async ({ context, page }) => {
    test.setTimeout(240_000);
    const tokenA = await registrar("anotaA");
    const colecao = `Marcas${Date.now()}`;
    await enviarEEsperar(tokenA, colecao);
    const baseURL = process.env.THINGINDEX_REAL_BASE_URL ?? "http://localhost:3988";
    await context.addCookies([{ name: "token", value: tokenA, url: baseURL }]);

    await page.goto("/pt-BR/workspace", { waitUntil: "domcontentloaded" });
    // O painel de chat aberto tira o quadro da tela e as paginas so sao
    // renderizadas quando visiveis; o teste conta grifos, entao fecha o painel.
    await page.getByTestId("alternar-chat").click();
    await page.locator("header select").first().selectOption({ label: colecao });
    const grifos = page.getByTestId("grifo");
    await expect(grifos.first()).toBeVisible({ timeout: 60_000 });
    await grifos.first().click();
    await page.getByTestId("barra-de-grifo").getByTestId("chip-amarelo").click();
    await expect(grifos.first()).toHaveAttribute("data-cor", "amarelo");

    await page.reload({ waitUntil: "domcontentloaded" });
    await page.getByTestId("alternar-chat").click();
    await page.locator("header select").first().selectOption({ label: colecao });
    await expect(page.locator('[data-testid="grifo"][data-cor="amarelo"]')).toHaveCount(1, { timeout: 60_000 });

    // Isolamento: B nao lista a marca de A, nem pela colecao de A.
    const tokenB = await registrar("anotaB");
    const colecaoID = await idDaColecao(tokenA, colecao);
    const deA = (await (await fetch(`${API}/documents/collections/${colecaoID}/highlights`, { headers: { Authorization: `Bearer ${tokenA}` } })).json()) as { data?: { highlights?: unknown[] } };
    expect(deA.data?.highlights).toHaveLength(1);
    const deB = await fetch(`${API}/documents/collections/${colecaoID}/highlights`, { headers: { Authorization: `Bearer ${tokenB}` } });
    expect(deB.status).toBe(404);
});
