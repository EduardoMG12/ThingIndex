import type { MetadataRoute } from "next";

import { SUPPORTED_LOCALES } from "@/lib/i18n/config";
import { PUBLIC_INDEXABLE_PATHS } from "@/lib/route-access";

// So paginas publicas: as autenticadas redirecionam para /login e nao devem
// ser indexadas (o robots.ts tambem as bloqueia).
export default function sitemap(): MetadataRoute.Sitemap {
  const baseUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://thingindex.app";

  return SUPPORTED_LOCALES.flatMap((locale) =>
    PUBLIC_INDEXABLE_PATHS.map((path) => ({
      url: path === "/" ? `${baseUrl}/${locale}` : `${baseUrl}/${locale}${path}`,
      lastModified: new Date(),
      changeFrequency: path === "/" ? "weekly" : "monthly",
      priority: path === "/" ? 1 : 0.7,
    }))
  );
}
