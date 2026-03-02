"use client";

import * as React from "react";
import { motion } from "framer-motion";
import Header from "@/components/Header";
import UrlPanel, { type UrlEntry } from "@/components/UrlPanel";
import PromptPanel from "@/components/PromptPanel";
import GraphViewer from "@/components/GraphViewer";
import ResponsePanel, { type Message } from "@/components/ResponsePanel";
import { fetchGraphTopics, fetchGraphData, startCrawl, purgeCrawler } from "./actions";

const fadeUp = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.4 }
};

export default function Home() {
  const [urls, setUrls] = React.useState<UrlEntry[]>([]);
  const [messages, setMessages] = React.useState<Message[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [isCrawling, setIsCrawling] = React.useState(false);
  const [hasCrawled, setHasCrawled] = React.useState(false);
  const [graphData, setGraphData] = React.useState<{ nodes: any[]; links: any[]; totalNodes: number; }>({ nodes: [], links: [], totalNodes: 0 });
  const [isLoadingGraph, setIsLoadingGraph] = React.useState(true);
  const [graphContext, setGraphContext] = React.useState<string>("all");
  const [availableTopics, setAvailableTopics] = React.useState<string[]>([]);

  // Fetch topics once on mount
  React.useEffect(() => {
    async function loadTopics() {
      const topics = await fetchGraphTopics();
      setAvailableTopics(topics);
    }
    loadTopics();
  }, []);

  React.useEffect(() => {
    async function loadGraphData() {
      setIsLoadingGraph(true);
      const data = await fetchGraphData(graphContext, 3000);
      setGraphData(data);
      setIsLoadingGraph(false);
    }
    loadGraphData();
  }, [graphContext]);

  const handleAddUrl = React.useCallback(async (url: string, dirName: string, depth: number) => {
    const id = Math.random().toString(36).substr(2, 9);

    // Add to UI immediately with "crawling" status
    setUrls((prev) => [...prev, { id, url, dirName, depth, status: "crawling" }]);
    setIsCrawling(true);
    setHasCrawled(true);

    // Call the server action
    const result = await startCrawl(url, dirName, depth);

    // Update status based on result
    setUrls((prev) =>
      prev.map((u) =>
        u.id === id
          ? {
            ...u,
            status: result.status === "success" ? "success" : "error",
            message: result.message,
          }
          : u
      )
    );
    setIsCrawling(false);
  }, []);

  const handleRemoveUrl = React.useCallback((id: string) => {
    setUrls((prev) => prev.filter((u) => u.id !== id));
  }, []);

  const handlePurge = React.useCallback(async () => {
    await purgeCrawler();
    setUrls((prev) =>
      prev.map((u) =>
        u.status === "crawling" ? { ...u, status: "error", message: "Crawler parado pelo usuário" } : u
      )
    );
    setIsCrawling(false);
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

      <main id="main-content" className="container py-8 flex-1 flex flex-col lg:flex-row gap-6">
        <div className="flex flex-col gap-6 w-full lg:w-[450px] shrink-0">
          <motion.div {...fadeUp}>
            <h1 className="text-2xl font-bold tracking-tight">
              Workspace <span className="text-gradient-gold">GraphRAG</span>
            </h1>
            <p className="mt-1.5 text-xs text-muted-foreground">
              Construa seu grafo de conhecimento legal e extraia insights com IA.
            </p>
          </motion.div>

          <motion.div
            {...fadeUp}
            transition={{ delay: 0.1 }}
            className="flex flex-col gap-6"
          >
            <UrlPanel
              urls={urls}
              onAdd={handleAddUrl}
              onRemove={handleRemoveUrl}
              onPurge={handlePurge}
              isCrawling={isCrawling}
              hasCrawled={hasCrawled}
            />
            <PromptPanel onSubmit={handlePromptSubmit} isLoading={isLoading} />
            <ResponsePanel messages={messages} />
          </motion.div>
        </div>

        <motion.div
          {...fadeUp}
          transition={{ delay: 0.2 }}
          className="flex-1 min-h-[500px] lg:min-h-0 lg:sticky lg:top-24 lg:h-[calc(100vh-8rem)]"
          role="region"
          aria-label="Visualização do grafo de conhecimento"
        >
          {/* Right Column: Visualization */}
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
            className="surface-card flex min-h-[500px] flex-col overflow-hidden rounded-xl lg:col-span-8 lg:min-h-0"
            role="region"
            aria-label="Visualização do grafo de conhecimento"
          >
            <div className="flex-1 relative">
              {isLoadingGraph ? (
                <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
                  <div className="flex flex-col items-center gap-3">
                    <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                    <p className="text-sm font-medium">Carregando grafo de conhecimento...</p>
                  </div>
                </div>
              ) : graphData.nodes.length === 0 ? (
                <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
                  <p className="text-sm">Nenhum dado encontrado no Neo4j.</p>
                </div>
              ) : (
                <GraphViewer
                  nodes={graphData.nodes}
                  links={graphData.links}
                  totalNodes={graphData.totalNodes}
                  currentContext={graphContext}
                  onContextChange={setGraphContext}
                  availableTopics={availableTopics}
                />
              )}
            </div>
          </motion.div>
        </motion.div>
      </main>
    </div>
  );
}
