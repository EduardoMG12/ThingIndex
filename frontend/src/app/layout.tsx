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
              toast: 'backdrop-blur-xl backdrop-filter border',
              error: 'bg-red-950/40 border-red-500/20 text-red-200',
              success: 'bg-emerald-950/40 border-emerald-500/20 text-emerald-200',
              warning: 'bg-amber-950/40 border-amber-500/20 text-amber-200',
              info: 'bg-blue-950/40 border-blue-500/20 text-blue-200',
              default: 'bg-zinc-950/40 border-white/10 text-zinc-200',
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
