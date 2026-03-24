import type { AppLocale } from "@/lib/i18n/config";
import { buildLocalizedMetadata } from "@/lib/seo";
import CostVisibilityPage from "@/app/cost-visibility/page";

export async function generateMetadata({ params }: { params: Promise<{ locale: AppLocale }> }) {
    const { locale } = await params;
    return buildLocalizedMetadata(locale, "costVisibility", "/cost-visibility");
}

export default CostVisibilityPage;
