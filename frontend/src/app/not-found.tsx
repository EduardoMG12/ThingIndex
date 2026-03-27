import { NotFoundContent } from "@/components/ui/NotFoundContent";

export const metadata = {
    title: "404 - Página Não Encontrada | ThingIndex",
    description: "Oops! A página que você está procurando não existe no ThingIndex.",
};

import { I18nProvider } from "@/lib/i18n/provider";

export default function NotFound() {
    return (
        <I18nProvider locale="pt-BR">
            <NotFoundContent />
        </I18nProvider>
    );
}
