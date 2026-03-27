export const SUPPORTED_LOCALES = ["pt-BR", "en", "es"] as const;

export type AppLocale = (typeof SUPPORTED_LOCALES)[number];

export const DEFAULT_LOCALE: AppLocale = "en";
export const LOCALE_COOKIE_NAME = "thingindex-locale";
export const LOCALE_HEADER_NAME = "x-locale";

const LOCALE_ALIASES: Record<string, AppLocale> = {
    pt: "pt-BR",
    "pt-br": "pt-BR",
    en: "en",
    "en-us": "en",
    "en-gb": "en",
    es: "es",
    "es-es": "es",
    "es-419": "es",
};

export function isSupportedLocale(value: string): value is AppLocale {
    return SUPPORTED_LOCALES.includes(value as AppLocale);
}

export function normalizeLocale(value?: string | null): AppLocale {
    if (!value) {
        return DEFAULT_LOCALE;
    }

    if (isSupportedLocale(value)) {
        return value;
    }

    const normalized = value.trim().toLowerCase();
    if (normalized in LOCALE_ALIASES) {
        return LOCALE_ALIASES[normalized];
    }

    const language = normalized.split("-")[0];
    return LOCALE_ALIASES[language] ?? DEFAULT_LOCALE;
}

export function getPreferredLocale(acceptLanguage?: string | null): AppLocale {
    if (!acceptLanguage) {
        return DEFAULT_LOCALE;
    }

    const candidates = acceptLanguage
        .split(",")
        .map((item) => item.split(";")[0]?.trim())
        .filter(Boolean);

    for (const candidate of candidates) {
        const locale = normalizeLocale(candidate);
        if (locale) {
            return locale;
        }
    }

    return DEFAULT_LOCALE;
}

export function stripLocaleFromPathname(pathname: string): string {
    const segments = pathname.split("/");
    const maybeLocale = segments[1];

    if (!maybeLocale || !isSupportedLocale(maybeLocale)) {
        return pathname || "/";
    }

    const stripped = `/${segments.slice(2).join("/")}`.replace(/\/+/g, "/");
    return stripped === "/" ? "/" : stripped.replace(/\/$/, "") || "/";
}

export function getLocaleFromPathname(pathname: string): AppLocale | null {
    const maybeLocale = pathname.split("/")[1];
    return maybeLocale && isSupportedLocale(maybeLocale) ? maybeLocale : null;
}

export function localizePath(pathname: string, locale: AppLocale): string {
    const cleanPath = pathname.startsWith("/") ? pathname : `/${pathname}`;
    const stripped = stripLocaleFromPathname(cleanPath);
    return stripped === "/" ? `/${locale}` : `/${locale}${stripped}`;
}

export function replacePathLocale(pathname: string, locale: AppLocale): string {
    return localizePath(pathname, locale);
}

export function getLocaleLabel(locale: AppLocale): string {
    switch (locale) {
        case "pt-BR":
            return "Português";
        case "es":
            return "Español";
        default:
            return "English";
    }
}
