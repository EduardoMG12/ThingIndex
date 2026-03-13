import type { Metadata } from "next";

import {
    DEFAULT_LOCALE,
    localizePath,
    SUPPORTED_LOCALES,
    type AppLocale,
} from "@/lib/i18n/config";
import { getMessages } from "@/lib/i18n/messages";

type PageKey = "home" | "login" | "register" | "dashboard" | "documents" | "documentation" | "settings";

export function buildLocalizedMetadata(locale: AppLocale, page: PageKey, pathname: string): Metadata {
    const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://thingindex.app";
    const seo = getMessages(locale).seo as {
        siteTitle: string;
        siteDescription: string;
        pages: Record<PageKey, { title: string; description: string }>;
    };
    const pageMetadata = seo.pages[page];
    const canonical = `${siteUrl}${localizePath(pathname, locale)}`;

    return {
        title: pageMetadata.title,
        description: pageMetadata.description,
        alternates: {
            canonical,
            languages: Object.fromEntries(
                SUPPORTED_LOCALES.map((supportedLocale) => [
                    supportedLocale,
                    `${siteUrl}${localizePath(pathname, supportedLocale)}`,
                ])
            ),
        },
        openGraph: {
            type: "website",
            locale,
            siteName: seo.siteTitle,
            title: pageMetadata.title,
            description: pageMetadata.description,
            url: canonical,
        },
        twitter: {
            card: "summary_large_image",
            title: pageMetadata.title,
            description: pageMetadata.description,
        },
        robots: {
            index: true,
            follow: true,
        },
        metadataBase: new URL(siteUrl),
    };
}

export function buildAlternateLinks(pathname: string) {
    const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://thingindex.app";

    return Object.fromEntries(
        SUPPORTED_LOCALES.map((locale) => [locale, `${siteUrl}${localizePath(pathname, locale)}`]).concat([
            ["x-default", `${siteUrl}${localizePath(pathname, DEFAULT_LOCALE)}`],
        ])
    );
}
