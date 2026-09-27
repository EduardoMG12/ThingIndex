import type { MetadataRoute } from "next";

import { SUPPORTED_LOCALES } from "@/lib/i18n/config";
import { PROTECTED_PATHS } from "@/lib/route-access";

export default function robots(): MetadataRoute.Robots {
  const baseUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://thingindex.app";
  const disallow = [
    "/api/",
    ...PROTECTED_PATHS,
    ...SUPPORTED_LOCALES.flatMap((locale) => PROTECTED_PATHS.map((path) => `/${locale}${path}`)),
  ];

  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow,
    },
    sitemap: `${baseUrl}/sitemap.xml`,
  };
}
