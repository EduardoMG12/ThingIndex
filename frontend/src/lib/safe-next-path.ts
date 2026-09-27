/**
 * Valida o destino pos-login vindo de ?next= para evitar open redirect.
 * So aceita caminho relativo local: comeca com "/", nao com "//" nem "/\",
 * e sem barra invertida ou caractere de controle (o navegador descarta tab e
 * quebra de linha, o que transformaria "/\t/host" em "//host").
 */
export function resolveSafeNextPath(value: unknown): string | null {
    if (typeof value !== "string" || value.length === 0) {
        return null;
    }
    if (!value.startsWith("/") || value.startsWith("//")) {
        return null;
    }
    if (value.includes("\\")) {
        return null;
    }
    if (/[\u0000-\u001f\u007f]/.test(value)) {
        return null;
    }
    return value;
}
