import { expect, test } from "@playwright/test";

/**
 * Dashboard e Chat viraram a Area de trabalho: as rotas antigas redirecionam
 * e a query (colecao, conversa) sobrevive ao redirect.
 */

test.use({ locale: "pt-BR" });

test("/dashboard e /chat redirecionam para a Area de trabalho preservando a query", async ({ baseURL, context, page }) => {
    await context.addCookies([{ name: "token", value: "playwright-token", url: baseURL ?? "http://127.0.0.1:3000" }]);
    await context.route("**/api/**", (route) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ status: 200, type: "success", message: "ok", data: [] }) }));

    await page.goto("/pt-BR/dashboard");
    await expect(page).toHaveURL(/\/pt-BR\/workspace$/);

    await page.goto("/pt-BR/chat?collection=3&conversation=9");
    await expect(page).toHaveURL(/\/pt-BR\/workspace\?collection=3&conversation=9$/);
});
