import type { Metadata, Viewport } from "next";
import { Toaster } from "sonner";
import { cn } from "@/lib/utils";
import { SplashScreen } from "@/components/ui/SplashScreen";
import { I18nProvider } from "@/lib/i18n/provider";
import { SkipToContentLink } from "@/components/ui/SkipToContentLink";
import { TOAST_CLASS_NAMES } from "@/components/ui/toast-class-names";
import "./globals.css";

const themeInitScript = `
(() => {
  // Corrige o atributo lang antes do primeiro paint. O <html> e renderizado
  // aqui, na raiz estatica, entao nao ha como saber o locale no servidor sem
  // chamar cookies() -- o que desligaria a pre-renderizacao das 38 paginas.
  // Ler o locale do proprio pathname mantem o SSG e ja entrega o valor certo
  // ao leitor de tela, que le o DOM vivo. Ver ROADMAP secao 4 para a correcao
  // definitiva (mover <html> para app/[locale]/layout.tsx).
  const SUPPORTED = ["pt-BR", "en", "es"];
  const segment = window.location.pathname.split("/")[1];
  const matched = SUPPORTED.find((locale) => locale.toLowerCase() === (segment || "").toLowerCase());
  if (matched) {
    document.documentElement.lang = matched;
  }

  let storedTheme = null;
  try {
    const raw = window.localStorage.getItem("thingindex-theme");
    if (raw === "dark" || raw === "light") {
      storedTheme = raw;
    }
  } catch (error) {
    // localStorage indisponivel: cai no prefers-color-scheme.
  }

  const preferredTheme = storedTheme
    ?? (window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark");

  document.documentElement.dataset.theme = preferredTheme;
  document.documentElement.style.colorScheme = preferredTheme;
})();
`;

export const metadata: Metadata = {
  title: {
    default: "ThingIndex",
    template: "%s | ThingIndex",
  },
  description: "Relational document intelligence for indexing, structured extraction, and context recovery.",
  keywords: ["document intelligence", "relational context", "document indexing", "structured extraction", "ThingIndex"],
  authors: [{ name: "ThingIndex Team" }],
  creator: "ThingIndex",
  robots: {
    index: true,
    follow: true,
  },
  openGraph: {
    type: "website",
    locale: "en",
    siteName: "ThingIndex",
    title: "ThingIndex",
    description: "Relational document intelligence for indexing, structured extraction, and context recovery.",
  },
  twitter: {
    card: "summary_large_image",
    title: "ThingIndex",
    description: "Relational document intelligence for indexing, structured extraction, and context recovery.",
  },
};

export const viewport: Viewport = {
  themeColor: "#1a1c22",
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt-BR" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className={cn("antialiased min-h-screen bg-background")}>
        <I18nProvider locale="pt-BR">
          <SplashScreen />
          <Toaster
            position="top-right"
            toastOptions={{
              classNames: TOAST_CLASS_NAMES,
            }}
          />
          <SkipToContentLink />
          {children}
        </I18nProvider>
      </body>
    </html>
  );
}
