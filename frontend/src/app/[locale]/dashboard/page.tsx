import type { AppLocale } from "@/lib/i18n/config";
import { buildLocalizedMetadata } from "@/lib/seo";
import DashboardPage from "@/app/dashboard/page";

export async function generateMetadata({ params }: { params: Promise<{ locale: AppLocale }> }) {
    const { locale } = await params;
    return buildLocalizedMetadata(locale, "dashboard", "/dashboard");
}

export default DashboardPage;
