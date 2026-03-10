import type { Metadata, Viewport } from "next";
import { Toaster } from "sonner";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "LegisGraph — Grafo de Conhecimento Jurídico com IA",
    template: "%s | LegisGraph",
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
    "LegisGraph",
  ],
  authors: [{ name: "LegisGraph Team" }],
  creator: "LegisGraph",
  robots: {
    index: true,
    follow: true,
  },
  openGraph: {
    type: "website",
    locale: "pt_BR",
    siteName: "LegisGraph",
    title: "LegisGraph — Grafo de Conhecimento Jurídico com IA",
    description:
      "Construa grafos de conhecimento a partir de legislação brasileira e extraia insights com inteligência artificial.",
  },
  twitter: {
    card: "summary_large_image",
    title: "LegisGraph — Grafo de Conhecimento Jurídico com IA",
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
      <body className="antialiased">
        <Toaster
          position="top-right"
          toastOptions={{
            classNames: {
              toast: 'backdrop-blur-2xl backdrop-saturate-150 border-white/10 bg-black/40 shadow-2xl rounded-xl p-4',
              error: 'border-red-500/30 text-red-200 !bg-red-500/10',
              success: 'border-emerald-500/30 text-emerald-200 !bg-emerald-500/10',
              warning: 'border-amber-500/30 text-amber-200 !bg-amber-500/10',
              info: 'border-blue-500/30 text-blue-200 !bg-blue-500/10',
              default: 'border-white/10 text-zinc-200 !bg-white/5',
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
