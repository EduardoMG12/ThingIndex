import { notFound } from "next/navigation";

import { SUPPORTED_LOCALES, type AppLocale } from "@/lib/i18n/config";
import { getMessages } from "@/lib/i18n/messages";
import { I18nProvider } from "@/lib/i18n/provider";

export function generateStaticParams() {
    return SUPPORTED_LOCALES.map((locale) => ({ locale }));
}

export default async function LocaleLayout({
    children,
    params,
}: {
    children: React.ReactNode;
    params: Promise<{ locale: AppLocale }>;
}) {
    const { locale } = await params;

    if (!SUPPORTED_LOCALES.includes(locale)) {
        notFound();
    }

    return <I18nProvider locale={locale} messages={getMessages(locale)}>{children}</I18nProvider>;
}
