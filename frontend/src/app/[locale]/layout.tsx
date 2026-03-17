import { notFound } from "next/navigation";

import { SUPPORTED_LOCALES, type AppLocale } from "@/lib/i18n/config";
import { I18nProvider } from "@/lib/i18n/provider";

export function generateStaticParams() {
    return SUPPORTED_LOCALES.map((locale) => ({ locale }));
}

export default async function LocaleLayout({
    children,
    params,
}: {
    children: React.ReactNode;
    params: Promise<{ locale: string }>;
}) {
    const { locale } = await params;

    if (!SUPPORTED_LOCALES.includes(locale as AppLocale)) {
        notFound();
    }

    return <I18nProvider locale={locale as AppLocale}>{children}</I18nProvider>;
}
