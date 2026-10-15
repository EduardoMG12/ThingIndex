import {
    DEFAULT_LOCALE,
    LOCALE_COOKIE_NAME,
    LOCALE_HEADER_NAME,
    normalizeLocale,
} from "@/lib/i18n/config";
import { createRequestId, REQUEST_ID_HEADER_NAME } from "@/lib/api/request-id";

function getClientLocale() {
    if (typeof document === "undefined") {
        return DEFAULT_LOCALE;
    }

    const cookieMatch = document.cookie.match(new RegExp(`(?:^|; )${LOCALE_COOKIE_NAME}=([^;]+)`));
    const cookieLocale = cookieMatch?.[1];
    const htmlLocale = document.documentElement.lang;
    return normalizeLocale(cookieLocale ?? htmlLocale ?? DEFAULT_LOCALE);
}

export class UnauthorizedRequestError extends Error {
    constructor(message = "Usuario nao autenticado.") {
        super(message);
        this.name = "UnauthorizedRequestError";
    }
}

function redirectToLogin() {
    if (typeof window === "undefined") {
        return;
    }

    const locale = getClientLocale();
    const loginPath = `/${locale}/login`;

    if (window.location.pathname === loginPath) {
        return;
    }

    window.location.assign(loginPath);
}

interface FetchWithAuthRedirectOptions {
    /**
    * false: 401 so lanca UnauthorizedRequestError, sem ir para o login (telas
    * publicas). Quem protege rota e o proxy, via PROTECTED_PATHS.
    */
    redirectOnUnauthorized?: boolean;
}

export async function fetchWithAuthRedirect(
    input: RequestInfo | URL,
    init?: RequestInit,
    options?: FetchWithAuthRedirectOptions
) {
    const locale = getClientLocale();
    const headers = new Headers(init?.headers);
    headers.set("Accept-Language", locale);
    headers.set(LOCALE_HEADER_NAME, locale);
    if (!headers.has(REQUEST_ID_HEADER_NAME)) {
        headers.set(REQUEST_ID_HEADER_NAME, createRequestId());
    }

    const response = await fetch(input, {
        ...init,
        headers,
    });

    if (response.status === 401) {
        if (options?.redirectOnUnauthorized !== false) {
            redirectToLogin();
        }
        throw new UnauthorizedRequestError();
    }

    return response;
}
