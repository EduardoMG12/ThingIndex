"use client";

import { createContext, useContext, useMemo } from "react";
import { usePathname } from "next/navigation";

import type { AppLocale } from "@/lib/i18n/config";
import { getLocaleFromPathname, localizePath } from "@/lib/i18n/config";
import { getMessages, type TranslationTree, type TranslationValue } from "@/lib/i18n/messages";

type I18nContextValue = {
    locale: AppLocale;
    messages: TranslationTree;
};

const I18nContext = createContext<I18nContextValue | null>(null);

function getValue(messages: TranslationTree, key: string): TranslationValue | undefined {
    return key.split(".").reduce<TranslationValue | undefined>((acc, part) => {
        if (!acc || typeof acc === "string") {
            return undefined;
        }
        return acc[part];
    }, messages);
}

export function I18nProvider({
    locale,
    children,
}: {
    locale: AppLocale;
    children: React.ReactNode;
}) {
    const pathname = usePathname();
    const activeLocale = getLocaleFromPathname(pathname) ?? locale;
    const messages = useMemo(() => getMessages(activeLocale), [activeLocale]);
    const value = useMemo(() => ({ locale: activeLocale, messages }), [activeLocale, messages]);
    return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
    const context = useContext(I18nContext);
    if (!context) {
        throw new Error("useI18n must be used within I18nProvider");
    }

    function t(key: string, variables?: Record<string, string | number>) {
        const value = getValue(context.messages, key);
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

    function href(pathname: string) {
        return localizePath(pathname, context.locale);
    }

    return {
        locale: context.locale,
        t,
        href,
    };
}
