import { beforeEach, describe, expect, it, vi } from "vitest";

const { cookieStore, fetchApiUpstreamMock, redirectMock } = vi.hoisted(() => ({
    cookieStore: {
        locale: "en" as string | undefined,
        get(name: string) {
            return name === "thingindex-locale" && this.locale ? { value: this.locale } : undefined;
        },
        set: vi.fn(),
        delete: vi.fn(),
    },
    fetchApiUpstreamMock: vi.fn(),
    redirectMock: vi.fn((url: string) => {
        throw new Error(`NEXT_REDIRECT:${url}`);
    }),
}));

vi.mock("next/headers", () => ({
    cookies: async () => cookieStore,
}));

vi.mock("next/navigation", () => ({
    redirect: redirectMock,
}));

vi.mock("@/app/api/_utils/upstream", () => ({
    fetchApiUpstream: fetchApiUpstreamMock,
}));

vi.mock("@/app/api/_utils/locale", () => ({
    createForwardHeaders: async () => new Headers(),
}));

import { login, register } from "@/components/auth/actions";
import { LOCALE_COOKIE_NAME } from "@/lib/i18n/config";

function formulario(campos: Record<string, string>) {
    const data = new FormData();
    for (const [nome, valor] of Object.entries(campos)) {
        data.set(nome, valor);
    }
    return data;
}

describe("mensagens de erro de login e cadastro seguem o locale (F-03)", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        expect(LOCALE_COOKIE_NAME).toBe("thingindex-locale");
    });

    it("login sem campos devolve a mensagem em ingles quando o locale e en", async () => {
        cookieStore.locale = "en";
        const res = await login(formulario({}));
        expect(res?.error).toBe("Email and password are required");
    });

    it("login sem campos devolve a mensagem em espanhol quando o locale e es", async () => {
        cookieStore.locale = "es";
        const res = await login(formulario({}));
        expect(res?.error).toBe("El correo y la contraseña son obligatorios");
    });

    it("login sem campos mantem o portugues no locale padrao", async () => {
        cookieStore.locale = "pt-BR";
        const res = await login(formulario({}));
        expect(res?.error).toBe("Email e senha são obrigatórios");
    });

    it("falha de conexao no login usa o texto do locale", async () => {
        cookieStore.locale = "en";
        fetchApiUpstreamMock.mockRejectedValue(new Error("offline"));
        vi.spyOn(console, "error").mockImplementation(() => undefined);
        const res = await login(formulario({ email: "a@b.c", password: "12345678" }));
        expect(res?.error).toBe("Could not connect to the server");
    });

    it("login recusado sem mensagem do backend cai no fallback traduzido", async () => {
        cookieStore.locale = "es";
        fetchApiUpstreamMock.mockResolvedValue(new Response("{}", { status: 401 }));
        const res = await login(formulario({ email: "a@b.c", password: "errada" }));
        expect(res?.error).toBe("Credenciales inválidas");
    });

    it("login recusado preserva a mensagem ja localizada pelo backend", async () => {
        cookieStore.locale = "en";
        fetchApiUpstreamMock.mockResolvedValue(
            new Response(JSON.stringify({ message: "Invalid credentials from API" }), { status: 401 })
        );
        const res = await login(formulario({ email: "a@b.c", password: "errada" }));
        expect(res?.error).toBe("Invalid credentials from API");
    });

    it("cadastro sem token na resposta usa o texto do locale", async () => {
        cookieStore.locale = "en";
        fetchApiUpstreamMock.mockResolvedValue(new Response(JSON.stringify({ data: {} }), { status: 200 }));
        const res = await register(formulario({ name: "A", email: "a@b.c", password: "12345678" }));
        expect(res?.error).toBe("Token not received from the server");
    });

    it("cadastro sem campos devolve a mensagem em ingles", async () => {
        cookieStore.locale = "en";
        const res = await register(formulario({}));
        expect(res?.error).toBe("All fields are required");
    });
});

describe("login respeita o destino ?next= (F-04)", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        cookieStore.locale = "en";
        fetchApiUpstreamMock.mockResolvedValue(
            new Response(JSON.stringify({ data: { token: "jwt" } }), { status: 200 })
        );
    });

    it("redireciona para o caminho local pedido depois do login", async () => {
        await expect(
            login(formulario({ email: "a@b.c", password: "12345678", next: "/en/documents" }))
        ).rejects.toThrow("NEXT_REDIRECT:/en/documents");
    });

    it("ignora destino externo e cai no workspace do locale", async () => {
        await expect(
            login(formulario({ email: "a@b.c", password: "12345678", next: "//evil.example" }))
        ).rejects.toThrow("NEXT_REDIRECT:/en/workspace");
    });

    it("sem next continua indo para o workspace", async () => {
        await expect(
            login(formulario({ email: "a@b.c", password: "12345678" }))
        ).rejects.toThrow("NEXT_REDIRECT:/en/workspace");
    });
});
