import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    optimizePackageImports: ["lucide-react", "framer-motion"],
  },
  // Dashboard e Chat viraram a Area de trabalho; a query (?collection,
  // ?conversation) e preservada por padrao.
  async redirects() {
    return [
      { source: "/:locale(pt-BR|en|es)/dashboard", destination: "/:locale/workspace", permanent: false },
      { source: "/:locale(pt-BR|en|es)/chat", destination: "/:locale/workspace", permanent: false },
    ];
  },
};

export default nextConfig;
