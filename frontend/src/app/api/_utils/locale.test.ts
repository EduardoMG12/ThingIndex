import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { incomingHeadersMock } = vi.hoisted(() => ({
    incomingHeadersMock: vi.fn(),
}));

vi.mock("next/headers", () => ({
    cookies: async () => ({ get: () => undefined }),
    headers: incomingHeadersMock,
}));

import { createForwardHeaders } from "@/app/api/_utils/locale";

// Sem proxy reverso o x-forwarded-for vem do cliente e pode ser forjado,
// por isso o repasse e opt-in.
describe("createForwardHeaders: IP do cliente", () => {
    beforeEach(() => {
        incomingHeadersMock.mockReset();
        incomingHeadersMock.mockResolvedValue(new Headers());
    });

    afterEach(() => {
        vi.unstubAllEnvs();
    });

    describe("sem proxy confiavel (padrao)", () => {
        it("nao repassa o X-Forwarded-For que o cliente mandou", async () => {
            const request = new Request("http://frontend/api/x", {
                headers: { "x-forwarded-for": "203.0.113.7" },
            });

            const headers = await createForwardHeaders(request);

            expect(headers.get("X-Forwarded-For")).toBeNull();
        });

        it("nem o de next/headers em server action", async () => {
            incomingHeadersMock.mockResolvedValue(new Headers({ "x-forwarded-for": "2001:db8::1" }));

            const headers = await createForwardHeaders(undefined, { contentType: "application/json" });

            expect(headers.get("X-Forwarded-For")).toBeNull();
            expect(headers.get("Content-Type")).toBe("application/json");
        });
    });

    describe("atras de proxy reverso que sobrescreve o cabecalho (FRONTEND_TRUST_PROXY_HEADERS=true)", () => {
        beforeEach(() => {
            vi.stubEnv("FRONTEND_TRUST_PROXY_HEADERS", "true");
        });

        it("repassa o endereco que o proxy escreveu", async () => {
            const request = new Request("http://frontend/api/x", {
                headers: { "x-forwarded-for": "203.0.113.7" },
            });

            const headers = await createForwardHeaders(request);

            expect(headers.get("X-Forwarded-For")).toBe("203.0.113.7");
        });

        it("usa x-real-ip quando nao ha x-forwarded-for", async () => {
            const request = new Request("http://frontend/api/x", {
                headers: { "x-real-ip": "198.51.100.4" },
            });

            const headers = await createForwardHeaders(request);

            expect(headers.get("X-Forwarded-For")).toBe("198.51.100.4");
        });

        it("em server action le o IP de next/headers", async () => {
            incomingHeadersMock.mockResolvedValue(new Headers({ "x-forwarded-for": "2001:db8::1" }));

            const headers = await createForwardHeaders(undefined);

            expect(headers.get("X-Forwarded-For")).toBe("2001:db8::1");
        });

        it("descarta valor que nao e lista de IPs", async () => {
            const request = new Request("http://frontend/api/x", {
                headers: { "x-forwarded-for": "evil; drop table" },
            });

            const headers = await createForwardHeaders(request);

            expect(headers.get("X-Forwarded-For")).toBeNull();
        });

        it("fora de escopo de requisicao nao envia o cabecalho", async () => {
            incomingHeadersMock.mockRejectedValue(new Error("headers was called outside a request scope"));

            const headers = await createForwardHeaders(undefined);

            expect(headers.get("X-Forwarded-For")).toBeNull();
        });
    });
});
