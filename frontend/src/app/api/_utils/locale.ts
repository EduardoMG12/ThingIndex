import { cookies, headers as incomingHeaders } from "next/headers";
import { REQUEST_ID_HEADER_NAME } from "@/lib/api/request-id";

import {
    DEFAULT_LOCALE,
    LOCALE_COOKIE_NAME,
    LOCALE_HEADER_NAME,
    normalizeLocale,
    type AppLocale,
} from "@/lib/i18n/config";

export async function getRequestLocale(request?: Request): Promise<AppLocale> {
    const cookieStore = await cookies();
    const localeFromRequest =
        request?.headers.get(LOCALE_HEADER_NAME) ??
        request?.headers.get("accept-language") ??
        cookieStore.get(LOCALE_COOKIE_NAME)?.value;

    return normalizeLocale(localeFromRequest ?? DEFAULT_LOCALE);
}

// Sem isso o Gin ve sempre o IP do container do frontend e o limite por IP
// de /auth vira uma cota global. A API so confia nele vindo de TRUSTED_PROXIES.
const FORWARDED_FOR_HEADER = "X-Forwarded-For";
// Lista de IPv4/IPv6 separados por virgula; qualquer outra coisa e descartada.
const FORWARDED_FOR_PATTERN = /^[0-9A-Fa-f:.]+(\s*,\s*[0-9A-Fa-f:.]+)*$/;

function sanitizeForwardedFor(value: string | null | undefined): string | null {
    const trimmed = value?.trim();
    if (!trimmed || trimmed.length > 512 || !FORWARDED_FOR_PATTERN.test(trimmed)) {
        return null;
    }
    return trimmed
        .split(",")
        .map((item) => item.trim())
        .join(", ");
}

// So repassa com um proxy reverso na frente sobrescrevendo X-Forwarded-For;
// sem ele o valor vem do navegador e daria um balde novo a cada tentativa.
// Server actions nao tem Request, dai o fallback para next/headers.
export async function resolveClientForwardedFor(request?: Request): Promise<string | null> {
    if (process.env.FRONTEND_TRUST_PROXY_HEADERS !== "true") {
        return null;
    }
    let source: Pick<Headers, "get"> | undefined = request?.headers;
    if (!source) {
        try {
            source = await incomingHeaders();
        } catch {
            // Fora de um escopo de requisicao (teste, build): nao ha cliente.
            return null;
        }
    }

    return sanitizeForwardedFor(source?.get("x-forwarded-for")) ?? sanitizeForwardedFor(source?.get("x-real-ip"));
}

export async function createForwardHeaders(
    request: Request | undefined,
    options?: {
        token?: string;
        contentType?: string;
        headers?: HeadersInit;
    }
) {
    const headers = new Headers(options?.headers);
    const locale = await getRequestLocale(request);

    headers.set("Accept-Language", locale);
    headers.set(LOCALE_HEADER_NAME, locale);
    const requestId = request?.headers.get(REQUEST_ID_HEADER_NAME);
    if (requestId) {
        headers.set(REQUEST_ID_HEADER_NAME, requestId);
    }

    const forwardedFor = await resolveClientForwardedFor(request);
    if (forwardedFor) {
        headers.set(FORWARDED_FOR_HEADER, forwardedFor);
    }

    if (options?.token) {
        headers.set("Authorization", `Bearer ${options.token}`);
    }

    if (options?.contentType) {
        headers.set("Content-Type", options.contentType);
    }

    return headers;
}
