import type { AppLocale } from "@/lib/i18n/config";
import { buildLocalizedMetadata } from "@/lib/seo";
import DocumentationPage from "@/app/documentation/page";

export async function generateMetadata({ params }: { params: Promise<{ locale: AppLocale }> }) {
    const { locale } = await params;
    return buildLocalizedMetadata(locale, "documentation", "/documentation");
}

export default DocumentationPage;
