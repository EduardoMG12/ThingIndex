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

// Cabecalho que carrega o IP do cliente ate a API. O Next chama a API pelo
// servidor, entao sem ele o ClientIP visto pelo Gin e sempre o do container do
// frontend e o limite por IP de /auth vira uma cota global: 10 logins por minuto
// para todos os usuarios somados (parecer A3). A API so confia neste cabecalho
// quando a conexao vem de TRUSTED_PROXIES (a rede interna do compose).
const FORWARDED_FOR_HEADER = "X-Forwarded-For";
// Um endereco IPv4/IPv6 por item, separados por virgula. Qualquer outra coisa
// (texto livre, quebra de linha) e descartada em vez de repassada.
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

// Le a cadeia X-Forwarded-For da requisicao recebida pelo Next. Em route
// handler a requisicao vem por parametro; em server action (login, cadastro,
// esqueci a senha) nao ha Request, entao le de next/headers. O proprio Next
// preenche x-forwarded-for com o endereco do socket quando o cliente nao envia.
// A cadeia e repassada inteira: quem decide qual item e o cliente real e o Gin,
// andando da direita para a esquerda e pulando os proxies confiaveis.
export async function resolveClientForwardedFor(request?: Request): Promise<string | null> {
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
