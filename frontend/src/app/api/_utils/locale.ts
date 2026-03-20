import { cookies } from "next/headers";
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

    if (options?.token) {
        headers.set("Authorization", `Bearer ${options.token}`);
    }

    if (options?.contentType) {
        headers.set("Content-Type", options.contentType);
    }

    return headers;
}
