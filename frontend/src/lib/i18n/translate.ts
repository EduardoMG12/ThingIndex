import {
    DEFAULT_LOCALE,
    LOCALE_COOKIE_NAME,
    getLocaleFromPathname,
    normalizeLocale,
    type AppLocale,
} from "@/lib/i18n/config";
import { getMessages, type TranslationTree, type TranslationValue } from "@/lib/i18n/messages";

export type TranslationVariables = Record<string, string | number | boolean>;

function getTranslationValue(messages: TranslationTree, key: string): TranslationValue | undefined {
    return key.split(".").reduce<TranslationValue | undefined>((acc, part) => {
        if (!acc || typeof acc === "string") {
            return undefined;
        }

        return acc[part];
    }, messages);
}

export function translateFromMessages(
    messages: TranslationTree,
    key: string,
    variables?: TranslationVariables
) {
    const value = getTranslationValue(messages, key);
    if (typeof value !== "string") {
        return key;
    }

    if (!variables) {
        return value;
    }

    return Object.entries(variables).reduce((result, [name, replacement]) => {
        return result.replaceAll(`{${name}}`, String(replacement));
    }, value);
}

export function translate(locale: AppLocale, key: string, variables?: TranslationVariables) {
    return translateFromMessages(getMessages(locale), key, variables);
}

function readLocaleCookie() {
    if (typeof document === "undefined") {
        return undefined;
    }

    const localeCookie = document.cookie
        .split(";")
        .map((item) => item.trim())
        .find((item) => item.startsWith(`${LOCALE_COOKIE_NAME}=`));

    return localeCookie ? localeCookie.slice(`${LOCALE_COOKIE_NAME}=`.length) : undefined;
}

function resolveRuntimeLocale(): AppLocale {
    if (typeof window !== "undefined") {
        const pathnameLocale = getLocaleFromPathname(window.location.pathname);
        if (pathnameLocale) {
            return pathnameLocale;
        }
    }

    const localeFromCookie = readLocaleCookie();
    if (localeFromCookie) {
        return normalizeLocale(localeFromCookie);
    }

    if (typeof document !== "undefined") {
        const localeFromDocument = document.documentElement.lang;
        if (localeFromDocument) {
            return normalizeLocale(localeFromDocument);
        }
    }

    if (typeof navigator !== "undefined") {
        return normalizeLocale(navigator.language);
    }

    return DEFAULT_LOCALE;
}

export function translateRuntime(
    key: string,
    variables?: TranslationVariables,
    locale = resolveRuntimeLocale()
) {
    return translate(locale, key, variables);
}
