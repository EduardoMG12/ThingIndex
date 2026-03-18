import type { Metadata, Viewport } from "next";
import { Toaster } from "sonner";
import { cn } from "@/lib/utils";
import { SplashScreen } from "@/components/ui/SplashScreen";
import { I18nProvider } from "@/lib/i18n/provider";
import "./globals.css";

const themeInitScript = `
(() => {
  const preferredTheme = window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";

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
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className={cn("antialiased min-h-screen bg-background")}>
        <I18nProvider locale="pt-BR">
          <SplashScreen />
          <Toaster
            position="top-right"
            toastOptions={{
              classNames: {
                toast: "group rounded-2xl border-white/20 bg-white/5 p-6 min-w-[440px] shadow-[0_8px_32px_rgba(0,0,0,0.6)] backdrop-blur-3xl backdrop-saturate-200 transition-all duration-300",
                error: "!border-red-500/60 !text-white !bg-red-500/20 shadow-[0_0_40px_-10px_rgba(239,68,68,0.6)]",
                success: "!border-emerald-500/40 !text-white !bg-emerald-500/20 shadow-[0_0_40px_-10px_rgba(16,185,129,0.4)]",
                warning: "!border-amber-500/40 !text-white !bg-amber-500/20 shadow-[0_0_40px_-10px_rgba(245,158,11,0.4)]",
                info: "!border-blue-500/40 !text-white !bg-blue-500/20 shadow-[0_0_40px_-10px_rgba(59,130,246,0.4)]",
                default: "border-white/10 !text-white !bg-white/5",
                title: "text-xl font-bold",
                description: "mt-1 text-base font-normal !text-zinc-200",
              },
            }}
          />
          <a
            href="#main-content"
            className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-primary-foreground focus:shadow-lg focus:outline-none"
          >
            Skip to main content
          </a>
          {children}
        </I18nProvider>
      </body>
    </html>
  );
}
