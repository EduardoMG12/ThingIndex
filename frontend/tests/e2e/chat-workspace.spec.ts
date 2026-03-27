import { expect, test, type BrowserContext } from "@playwright/test";

function success(data: unknown) {
    return {
        status: 200,
        type: "success",
        message: "ok",
        data,
    };
}

async function authenticate(context: BrowserContext) {
    await context.addCookies([
        {
            name: "token",
            value: "playwright-token",
            url: "http://127.0.0.1:3000",
        },
    ]);
}

test.use({ locale: "en-US" });

test("chat workspace keeps conversation history, creates a fresh draft, and expands verified sources", async ({ context, page }) => {
    await authenticate(context);

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

    await page.goto("/en/chat");

    const workspace = page.locator("#main-content");

    await expect(page.getByRole("heading", { level: 1, name: "ThingIndex Chat" })).toBeVisible();
    await expect(workspace.getByRole("button", { name: "New chat" })).toBeVisible();
    await expect(workspace.getByRole("heading", { level: 2, name: "Start a contextual query" })).toBeVisible();
    await expect(workspace.getByText("No saved conversations")).toBeVisible();

    await workspace.getByRole("button", { name: /Deadlines and dates/i }).click();
    await workspace.getByRole("textbox", { name: "Chat message" }).press("Enter");

    await expect(workspace.getByRole("heading", { level: 4, name: "Next steps" })).toBeVisible();
    await expect(workspace.getByText("Escalate unresolved incidents within 24 hours.", { exact: true })).toBeVisible();
    await expect(workspace.getByRole("button", { name: /Deadlines and dates/i })).toBeVisible();
    await expect(workspace.getByText("Operations handbook.pdf", { exact: true })).toBeVisible();

    await workspace.getByLabel("Delete conversation").first().click();
    await page.getByLabel("Confirmation value").fill("Deadlines and dates");
    await page.getByRole("button", { name: "Confirm deletion" }).click();

    await expect(workspace.getByText("No saved conversations")).toBeVisible();
});
