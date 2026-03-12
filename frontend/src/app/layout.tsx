import type { Metadata, Viewport } from "next";
import { Toaster } from "sonner";
import { cn } from '@/lib/utils';
import { SplashScreen } from '@/components/ui/SplashScreen';
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "ThingIndex — Grafo de Conhecimento Jurídico com IA",
    template: "%s | ThingIndex",
  },
  description:
    "Plataforma de GraphRAG que constrói automaticamente um grafo de conhecimento jurídico a partir de legislação brasileira, permitindo consultas semânticas com IA.",
  keywords: [
    "grafo de conhecimento",
    "legislação brasileira",
    "GraphRAG",
    "inteligência artificial",
    "NLP jurídico",
    "Neo4j",
    "direito digital",
    "ThingIndex",
  ],
  authors: [{ name: "ThingIndex Team" }],
  creator: "ThingIndex",
  robots: {
    index: true,
    follow: true,
  },
  openGraph: {
    type: "website",
    locale: "pt_BR",
    siteName: "ThingIndex",
    title: "ThingIndex — Grafo de Conhecimento Jurídico com IA",
    description:
      "Construa grafos de conhecimento a partir de legislação brasileira e extraia insights com inteligência artificial.",
  },
  twitter: {
    card: "summary_large_image",
    title: "ThingIndex — Grafo de Conhecimento Jurídico com IA",
    description:
      "Construa grafos de conhecimento a partir de legislação brasileira e extraia insights com inteligência artificial.",
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
    <html lang="pt-BR">
      <body className={cn("antialiased min-h-screen bg-background")}>
        <SplashScreen />
        <Toaster
          position="top-right"
          toastOptions={{
            classNames: {
              toast: 'group backdrop-blur-3xl backdrop-saturate-200 border-white/20 bg-white/5 shadow-[0_8px_32px_rgba(0,0,0,0.6)] rounded-2xl p-6 min-w-[440px] transition-all duration-300',
              error: '!border-red-500/60 !text-white !bg-red-500/20 shadow-[0_0_40px_-10px_rgba(239,68,68,0.6)]',
              success: '!border-emerald-500/40 !text-white !bg-emerald-500/20 shadow-[0_0_40px_-10px_rgba(16,185,129,0.4)]',
              warning: '!border-amber-500/40 !text-white !bg-amber-500/20 shadow-[0_0_40px_-10px_rgba(245,158,11,0.4)]',
              info: '!border-blue-500/40 !text-white !bg-blue-500/20 shadow-[0_0_40px_-10px_rgba(59,130,246,0.4)]',
              default: 'border-white/10 !text-white !bg-white/5',
              title: 'text-xl font-bold',
              description: 'text-base font-normal !text-zinc-200 mt-1',
            },
          }}
        />
        {/* Skip to main content — accessibility */}
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-primary-foreground focus:shadow-lg focus:outline-none"
        >
          Pular para o conteúdo principal
        </a>
        {children}
      </body>
    </html>
  );
}
