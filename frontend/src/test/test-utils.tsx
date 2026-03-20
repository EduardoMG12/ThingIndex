import type { PropsWithChildren } from "react";

import { I18nProvider } from "@/lib/i18n/provider";
import type { AppLocale } from "@/lib/i18n/config";

export function createTestWrapper(locale: AppLocale = "en") {
    return function TestWrapper({ children }: PropsWithChildren) {
        return <I18nProvider locale={locale}>{children}</I18nProvider>;
    };
}
