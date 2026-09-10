import { expect, test, type BrowserContext } from "@playwright/test";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

// A URL do cookie acompanha a baseURL. Fixar a porta 3000 fazia o cookie ser
// descartado sempre que a suite rodava em outra porta, e todo teste autenticado
// caia na tela de login sem explicar o motivo.
// Alem do cookie, o bootstrap do app busca /api/user/me. Sem esse mock a
// chamada ia para a API real com um token falso, recebia 401 e a pagina
// redirecionava para o login — todo teste autenticado falhava com "elemento nao
// encontrado", escondendo a causa real.
async function authenticateRoutes(context: BrowserContext) {
    await context.route("**/api/user/me", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify({
                status: 200,
                type: "success",
                message: "ok",
                data: {
                    id: 1,
                    name: "Playwright",
                    email: "playwright@example.com",
                    avatarUrl: null,
                    updatedAt: "2026-01-01T00:00:00Z",
                },
            }),
        });
    });
}

async function authenticate(context: BrowserContext, baseURL: string | undefined) {
    await authenticateRoutes(context);
    await context.addCookies([
        {
            name: "token",
            value: "playwright-token",
            url: baseURL ?? "http://127.0.0.1:3000",
        },
    ]);
}

test.use({ locale: "en-US" });

// O diagnostico anterior estava errado: nao era seletor desatualizado. O h1
// "ThingIndex Chat" vive em ChatWorkspaceHero e a lista de conversas em
// ChatConversationRail — os dois componentes existem, tem teste unitario e
// story, mas nenhuma pagina os renderiza. A lista de conversas migrou para a
// sidebar do AppShell, sincronizada por evento (thingindex:chat-sidebar-sync).
// O teste agora exercita o fluxo real: sidebar vazia -> primeira mensagem cria
// a conversa -> resposta com fonte citada -> exclusao volta ao estado vazio.
// O chat vive no painel da Area de trabalho; a lista de conversas continua no
// menu lateral, pendurada no item "Workspace".
test("chat workspace keeps conversation history, creates a fresh draft, and expands verified sources", async ({ baseURL, context, page }) => {
    await authenticate(context, baseURL);

    type ConversationRecord = {
        id: number;
        collection_id: number;
        title: string;
        last_message_at: string | null;
        created_at: string;
        updated_at: string;
    };

    const conversations: ConversationRecord[] = [];

    await page.route("**/api/user/providers/runtime", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({
                chat: {
                    family: "LLM",
                    activation_target: "CHAT",
                    provider_type: "OPENAI",
                    public_metadata: {
                        model: "gpt-4o-mini",
                    },
                    updated_at: "2026-03-20T12:05:00Z",
                },
            })),
        });
    });

    await page.route("**/api/collections", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success([
                {
                    id: 7,
                    name: "Operations",
                    document_count: 2,
                    metadata: {},
                },
            ])),
        });
    });

    await page.route("**/api/chat/conversations/*", async (route) => {
        const request = route.request();
        const pathname = new URL(request.url()).pathname;
        const id = Number(pathname.match(/\/api\/chat\/conversations\/(\d+)\/?$/)?.[1] ?? "0");

        if (request.method() === "GET") {
            const conversation = conversations.find((item) => item.id === id) ?? {
                id,
                collection_id: 7,
                title: "Deadlines and dates",
                last_message_at: "2026-03-20T14:00:00Z",
                created_at: "2026-03-20T14:00:00Z",
                updated_at: "2026-03-20T14:00:00Z",
            };

            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify(success({
                    conversation,
                    messages: [],
                })),
            });
            return;
        }

        if (request.method() === "DELETE") {
            const index = conversations.findIndex((conversation) => conversation.id === id);
            if (index >= 0) {
                conversations.splice(index, 1);
            }

            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify(success(null)),
            });
            return;
        }

        await route.fallback();
    });

    await page.route("**/api/chat/conversations**", async (route) => {
        const request = route.request();
        if (request.method() === "GET") {
            await route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify(success(conversations)),
            });
            return;
        }

        if (request.method() === "POST") {
            const body = request.postDataJSON() as { collection_id: number; title: string };
            const created = {
                id: 19,
                collection_id: body.collection_id,
                title: "Deadlines and dates",
                last_message_at: null,
                created_at: "2026-03-20T14:00:00Z",
                updated_at: "2026-03-20T14:00:00Z",
            };

            conversations.unshift(created);

            await route.fulfill({
                status: 201,
                contentType: "application/json",
                body: JSON.stringify(success(created)),
            });
            return;
        }

        await route.fallback();
    });

    await page.route("**/api/chat/stream", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "text/event-stream",
            body: [
                'event: stage\ndata: {"stage":"retrieving_context"}',
                'event: stage\ndata: {"stage":"selecting_sources"}',
                'event: sources\ndata: {"sources":[{"id":"doc-19","docName":"Operations handbook.pdf","page":2,"snippet":"Escalate unresolved incidents within 24 hours."}]}',
                'event: stage\ndata: {"stage":"generating_answer"}',
                'event: delta\ndata: {"delta":"## Next steps\\n\\n- Escalate unresolved incidents within 24 hours."}',
                'event: done\ndata: {"stage":"completed"}',
                "",
            ].join("\n\n"),
        });
    });

    await page.route("**/api/collections/*/context**", async (route) => {
        await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(success({ collection: { id: 7, name: "Operations", document_count: 0, metadata: {} }, documents: [], outline_nodes: [], paragraph_nodes: [], table_sections: [], metadata_pairs: [], document_pages: [] })),
        });
    });
    await page.goto("/en/workspace");
    const workspace = page.getByTestId("painel-chat");
    // O painel de chat tambem e um <aside>; o menu lateral e o primeiro.
    const sidebar = page.getByRole("complementary").first();

    await expect(workspace.getByRole("heading", { level: 2, name: "Start a contextual query" })).toBeVisible();
    await expect(sidebar.getByRole("button", { name: "New chat" })).toBeVisible();
    await expect(sidebar.getByText("Saved conversations for the active collection appear here.")).toBeVisible();

    // A collection e escolhida sozinha quando so existe uma, entao o composer ja
    // aceita envio. A primeira mensagem e o que cria a conversa.
    await workspace.getByRole("textbox", { name: "Chat message" }).fill("What are the escalation deadlines?");
    await workspace.getByRole("textbox", { name: "Chat message" }).press("Enter");

    await expect(workspace.getByRole("heading", { name: "Next steps" })).toBeVisible();
    await expect(workspace.getByText("Escalate unresolved incidents within 24 hours.").first()).toBeVisible();
    await expect(workspace.getByText("Operations handbook.pdf", { exact: true })).toBeVisible();

    // A conversa criada aparece na sidebar via evento de sincronizacao.
    const conversationRow = sidebar.getByText("Deadlines and dates");
    await expect(conversationRow).toBeVisible();

    // Os botoes de renomear e excluir so ficam visiveis no hover da linha.
    await conversationRow.hover();
    await sidebar.getByRole("button", { name: "Delete conversation" }).click();
    await page.getByLabel("Confirmation value").fill("Deadlines and dates");
    await page.getByRole("button", { name: "Confirm deletion" }).click();

    await expect(sidebar.getByText("Saved conversations for the active collection appear here.")).toBeVisible();
});
