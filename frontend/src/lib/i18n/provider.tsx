"use client";

import { createContext, useCallback, useContext, useMemo } from "react";
import { usePathname } from "next/navigation";

import type { AppLocale } from "@/lib/i18n/config";
import { getLocaleFromPathname, localizePath } from "@/lib/i18n/config";
import { getMessages, type TranslationTree } from "@/lib/i18n/messages";
import { translateFromMessages, type TranslationVariables } from "@/lib/i18n/translate";

type I18nContextValue = {
    locale: AppLocale;
    messages: TranslationTree;
};

const I18nContext = createContext<I18nContextValue | null>(null);

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

    const t = useCallback((key: string, variables?: TranslationVariables) => {
        return translateFromMessages(context.messages, key, variables);
    }, [context.messages]);

    const href = useCallback((pathname: string) => {
        return localizePath(pathname, context.locale);
    }, [context.locale]);

    return {
        locale: context.locale,
        t,
        href,
    };
}
