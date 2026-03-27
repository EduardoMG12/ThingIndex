import type { AppLocale } from "@/lib/i18n/config";
import { buildLocalizedMetadata } from "@/lib/seo";
import LandingPage from "@/app/page";

export async function generateMetadata({ params }: { params: Promise<{ locale: AppLocale }> }) {
    const { locale } = await params;
    return buildLocalizedMetadata(locale, "home", "/");
}

export default LandingPage;
