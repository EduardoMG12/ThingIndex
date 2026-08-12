import type { MetadataRoute } from "next";

export default function sitemap(): MetadataRoute.Sitemap {
  const baseUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://thingindex.app";
  const locales = ["pt-BR", "en", "es"] as const;
  const localizedPaths = ["/", "/login", "/register", "/dashboard", "/chat", "/documents", "/cost-visibility", "/documentation", "/settings"];

  return locales.flatMap((locale) =>
    localizedPaths.map((path) => ({
      url: path === "/" ? `${baseUrl}/${locale}` : `${baseUrl}/${locale}${path}`,
      lastModified: new Date(),
      changeFrequency: path === "/" ? "weekly" : "monthly",
      priority: path === "/" ? 1 : 0.7,
    }))
  );
}
