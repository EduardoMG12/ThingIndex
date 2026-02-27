"use client";

import * as React from "react";
import { motion } from "framer-motion";
import Header from "@/components/Header";
import UrlPanel, { type UrlEntry } from "@/components/UrlPanel";
import PromptPanel from "@/components/PromptPanel";
import GraphViewer from "@/components/GraphViewer";
import ResponsePanel, { type Message } from "@/components/ResponsePanel";

// Demo graph data
const DEMO_NODES = [
  { id: "1", label: "CDC", group: 0 },
  { id: "2", label: "Cap. I", group: 1 },
  { id: "3", label: "Cap. II", group: 1 },
  { id: "4", label: "Art. 6°", group: 2 },
  { id: "5", label: "Art. 12", group: 2 },
  { id: "6", label: "Art. 14", group: 2 },
  { id: "7", label: "Resp. Civil", group: 3 },
  { id: "8", label: "Direito Info", group: 3 },
  { id: "9", label: "Cap. III", group: 1 },
  { id: "10", label: "Art. 18", group: 2 },
];

const DEMO_LINKS = [
  { source: "1", target: "2" },
  { source: "1", target: "3" },
  { source: "1", target: "9" },
  { source: "2", target: "4" },
  { source: "3", target: "5" },
  { source: "3", target: "6" },
  { source: "5", target: "7" },
  { source: "6", target: "7" },
  { source: "4", target: "8" },
  { source: "9", target: "10" },
  { source: "10", target: "7" },
];

const fadeUp = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.5 }
};

export default function Home() {
  const [urls, setUrls] = React.useState<UrlEntry[]>([]);
  const [messages, setMessages] = React.useState<Message[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);

  const handleAddUrl = React.useCallback((url: string, depth: number) => {
    setUrls((prev) => [...prev, { id: Math.random().toString(36).substr(2, 9), url, depth }]);
  }, []);

  const handleRemoveUrl = React.useCallback((id: string) => {
    setUrls((prev) => prev.filter((u) => u.id !== id));
  }, []);

  const handlePromptSubmit = React.useCallback((prompt: string) => {
    setMessages((prev) => [...prev, { role: "user", content: prompt }]);
    setIsLoading(true);

    // Simulate response
    setTimeout(() => {
      setMessages((prev) => [
        ...prev,
        {
          role: "assistant",
          content:
            "Análise realizada com base nos documentos indexados. Foram encontrados 3 artigos relevantes sobre o tema solicitado no Código de Defesa do Consumidor. Os artigos 12, 14 e 18 tratam diretamente da responsabilidade civil do fornecedor.",
        },
      ]);
      setIsLoading(false);
    }, 1500);
  }, []);

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <Header />

      <main className="container py-8 flex-1 flex flex-col">
        <motion.div {...fadeUp}>
          <h1 className="text-3xl font-bold tracking-tight">
            <span className="text-gradient-gold">Workspace</span>
          </h1>
          <p className="mt-2 text-sm text-muted-foreground max-w-2xl">
            Bem-vindo ao LegisGraph. Adicione URLs de referência para construir seu grafo de conhecimento legal
            e interaja com a inteligência artificial para extrair insights precisos.
          </p>
        </motion.div>

        <div className="mt-8 grid gap-6 lg:grid-cols-2 flex-1 items-start">
          <motion.div
            {...fadeUp}
            transition={{ delay: 0.1, duration: 0.5 }}
            className="flex flex-col gap-6"
          >
            <UrlPanel urls={urls} onAdd={handleAddUrl} onRemove={handleRemoveUrl} />
            <PromptPanel onSubmit={handlePromptSubmit} isLoading={isLoading} />
            <ResponsePanel messages={messages} />
          </motion.div>

          <motion.div
            {...fadeUp}
            transition={{ delay: 0.2, duration: 0.5 }}
            className="h-full sticky top-24"
          >
            <GraphViewer nodes={DEMO_NODES} links={DEMO_LINKS} />
          </motion.div>
        </div>
      </main>
    </div>
  );
}
