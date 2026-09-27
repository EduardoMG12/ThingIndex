import { describe, expect, it } from "vitest";

import { resolveSafeNextPath } from "@/lib/safe-next-path";

describe("resolveSafeNextPath (F-04, contra open redirect)", () => {
    it.each([
        "/en/documents",
        "/settings",
        "/pt-BR/workspace?collection=1#topo",
    ])("aceita caminho relativo local %s", (valor) => {
        expect(resolveSafeNextPath(valor)).toBe(valor);
    });

    it.each([
        ["vazio", ""],
        ["nulo", null],
        ["nao string", 42],
        ["url absoluta", "https://evil.example/x"],
        ["protocolo javascript", "javascript:alert(1)"],
        ["sem barra inicial", "workspace"],
        ["protocol-relative", "//evil.example"],
        ["barra invertida logo apos a barra", "/\\evil.example"],
        ["barra invertida no meio", "/foo\\..\\\\evil.example"],
        ["tab que o navegador descarta", "/\t/evil.example"],
        ["quebra de linha", "/\n/evil.example"],
    ])("recusa %s", (_nome, valor) => {
        expect(resolveSafeNextPath(valor)).toBeNull();
    });
});
