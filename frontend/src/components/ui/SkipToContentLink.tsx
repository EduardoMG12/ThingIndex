"use client";

import { useI18n } from "@/lib/i18n/provider";

// Link "pular para o conteudo". E client para ler o locale ativo pelo pathname
// (o layout raiz e estatico e nao conhece o idioma da rota).
export function SkipToContentLink() {
    const { t } = useI18n();

    return (
        <a
            href="#main-content"
            className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-primary-foreground focus:shadow-lg focus:outline-none"
        >
            {t("common.skipToContent")}
        </a>
    );
}
