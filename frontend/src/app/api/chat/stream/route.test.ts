import { beforeEach, describe, expect, it, vi } from "vitest";

const { cookiesMock, createForwardHeadersMock } = vi.hoisted(() => ({
    cookiesMock: vi.fn(),
    createForwardHeadersMock: vi.fn(),
}));

vi.mock("next/headers", () => ({
    cookies: cookiesMock,
}));

vi.mock("@/app/api/_utils/locale", () => ({
    createForwardHeaders: createForwardHeadersMock,
}));

import { POST } from "@/app/api/chat/stream/route";

function criarUpstreamInfinito() {
    const estado = { cancelado: false, motivo: undefined as unknown };
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
        async pull(controller) {
            await new Promise((resolve) => setTimeout(resolve, 5));
            controller.enqueue(encoder.encode("data: token\n\n"));
        },
        cancel(motivo) {
            estado.cancelado = true;
            estado.motivo = motivo;
        },
    });
    return { body, estado };
}

describe("proxy do stream do chat", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        vi.stubGlobal("fetch", vi.fn());
        cookiesMock.mockResolvedValue({ get: () => ({ value: "jwt" }) });
        createForwardHeadersMock.mockResolvedValue(new Headers());
    });

    it("repassa o signal da requisicao do cliente ao fetch do upstream", async () => {
        const { body } = criarUpstreamInfinito();
        vi.mocked(fetch).mockResolvedValue(new Response(body, { status: 200 }));
        const cliente = new AbortController();

        await POST(new Request("http://localhost/api/chat/stream", {
            method: "POST",
            body: "{}",
            signal: cliente.signal,
        }));

        const init = vi.mocked(fetch).mock.calls[0]?.[1];
        expect(init?.signal).toBeDefined();
        cliente.abort();
        expect(init?.signal?.aborted).toBe(true);
    });

    it("cancela o upstream quando o cliente aborta no meio do stream (F-01)", async () => {
        const { body, estado } = criarUpstreamInfinito();
        vi.mocked(fetch).mockResolvedValue(new Response(body, { status: 200 }));

        const response = await POST(new Request("http://localhost/api/chat/stream", {
            method: "POST",
            body: "{}",
        }));

        const reader = response.body!.getReader();
        const primeiro = await reader.read();
        expect(primeiro.done).toBe(false);

        await reader.cancel("cliente saiu");
        await new Promise((resolve) => setTimeout(resolve, 50));

        expect(estado.cancelado).toBe(true);
    }, 2000);
});
