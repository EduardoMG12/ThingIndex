import type { AppLocale } from "@/lib/i18n/config";
import { buildLocalizedMetadata } from "@/lib/seo";
import LoginPage from "@/app/(auth)/login/page";

export async function generateMetadata({ params }: { params: Promise<{ locale: AppLocale }> }) {
    const { locale } = await params;
    return buildLocalizedMetadata(locale, "login", "/login");
}

export default LoginPage;
