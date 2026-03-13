import type { AppLocale } from "@/lib/i18n/config";
import { buildLocalizedMetadata } from "@/lib/seo";
import DocumentsPage from "@/app/documents/page";

export async function generateMetadata({ params }: { params: Promise<{ locale: AppLocale }> }) {
    const { locale } = await params;
    return buildLocalizedMetadata(locale, "documents", "/documents");
}

export default DocumentsPage;
