import { cookies } from "next/headers";

import {
    DEFAULT_LOCALE,
    getPreferredLocale,
    type AppLocale,
    LOCALE_COOKIE_NAME,
    normalizeLocale,
} from "@/lib/i18n/config";
import { getMessages } from "@/lib/i18n/messages";

export async function getServerLocale(): Promise<AppLocale> {
    const cookieStore = await cookies();
    const cookieLocale = cookieStore.get(LOCALE_COOKIE_NAME)?.value;
    return normalizeLocale(cookieLocale ?? DEFAULT_LOCALE);
}

export async function getServerMessages() {
    const locale = await getServerLocale();
    return {
        locale,
        messages: getMessages(locale),
    };
}

export function resolveLocaleFromHeaders(acceptLanguage?: string | null): AppLocale {
    return getPreferredLocale(acceptLanguage);
}
