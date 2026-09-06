import { existsSync } from "node:fs";
import { defineConfig } from "@playwright/test";

// Configuracao para o E2E SEM MOCK, contra a stack real do docker-compose.
//
// A suite padrao (playwright.config.ts) sobe o proprio servidor e intercepta as
// rotas de API com page.route: prova que a interface reage corretamente a uma
// resposta conhecida, o que e util e rapido. Nao prova que o sistema inteiro
// funciona junto.
//
// Esta suite percorre o caminho que o usuario faz de verdade -- login, upload
// de PDF, indexacao real, estrutura extraida, conexoes derivadas -- e foi assim
// que dois defeitos de layout apareceram nesta rodada: o painel esmagado a
// 72px e a conexao justificada por palavra vazia. Nenhum dos dois seria visto
// por teste de componente.
//
// Nao tem webServer: a stack precisa estar de pe antes. E deliberado -- subir
// docker-compose de dentro do Playwright esconderia falha de infraestrutura
// dentro de um timeout de teste.
const baseURL = process.env.THINGINDEX_REAL_BASE_URL ?? "http://localhost:3988";
const chromiumExecutablePath = [
    process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
    "/snap/bin/chromium",
].find((candidate) => typeof candidate === "string" && candidate.length > 0 && existsSync(candidate));

export default defineConfig({
    testDir: "./tests/e2e-real",
    fullyParallel: false,
    // Generoso: indexacao real de PDF passa por fila, extracao e persistencia.
    // O caminho com mock responde em milissegundos; este leva dezenas de
    // segundos, e um timeout curto reportaria lentidao legitima como falha.
    timeout: 240_000,
    expect: {
        timeout: 30_000,
    },
    // Sem retentativa: aqui uma falha e sinal de defeito real no sistema, e
    // repetir mascararia intermitencia que vale investigar.
    retries: 0,
    workers: 1,
    reporter: "list",
    use: {
        baseURL,
        trace: "retain-on-failure",
        screenshot: "only-on-failure",
        video: "retain-on-failure",
        ...(chromiumExecutablePath ? { launchOptions: { executablePath: chromiumExecutablePath } } : {}),
    },
    projects: [
        {
            name: "chromium-stack-real",
            use: { browserName: "chromium" },
        },
    ],
});
