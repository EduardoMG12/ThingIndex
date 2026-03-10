"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { FileStack, MessageSquare, Info, FolderOpen, Tag } from "lucide-react";
import AppShell from "@/components/layout/AppShell";
import AccountStatus from "@/components/dashboard/AccountStatus";
import CostCalculator from "@/components/dashboard/CostCalculator";
import UploadTab from "@/components/dashboard/UploadTab";
import ChatTab from "@/components/dashboard/ChatTab";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

export default function DashboardPage() {
    const [activeTab, setActiveTab] = useState("upload");

    return (
        <AppShell>
            <motion.div
                {...fadeUp}
                transition={{ duration: 0.5 }}
                className="mb-8"
            >
                <div className="flex items-center gap-2 mb-3">
                    <span className="h-px w-8 bg-primary" />
                    <span className="text-[10px] uppercase tracking-[0.3em] text-primary font-bold">Intelligent Workspace</span>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    Legis<span className="text-primary italic">Graph</span> Dashboard
                </h1>
                <p className="mt-4 max-w-2xl text-base text-muted-foreground leading-relaxed">
                    Centralize sua legislação e documentos jurídicos. Extraia conhecimento estruturado
                    através do nosso pipeline de <span className="text-foreground font-semibold">Graph-RAG</span> de alta performance.
                </p>
            </motion.div>

            <motion.div {...fadeUp} transition={{ delay: 0.1, duration: 0.4 }}>
                <AccountStatus />
            </motion.div>

            <div className="mt-8 grid gap-8 lg:grid-cols-[1fr_380px]">
                <motion.div
                    {...fadeUp}
                    transition={{ delay: 0.2, duration: 0.4 }}
                    className="min-w-0"
                >
                    <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
                        <TabsList className="mb-6 w-full justify-start gap-4 bg-transparent p-0 h-auto">
                            <TabsTrigger
                                value="upload"
                                className="data-[state=active]:bg-primary/10 data-[state=active]:text-primary border border-transparent data-[state=active]:border-primary/20 gap-2 px-6 py-2.5 text-xs font-bold uppercase tracking-widest transition-all"
                            >
                                <FileStack className="h-4 w-4" /> Pipeline
                            </TabsTrigger>
                            <TabsTrigger
                                value="chat"
                                className="data-[state=active]:bg-primary/10 data-[state=active]:text-primary border border-transparent data-[state=active]:border-primary/20 gap-2 px-6 py-2.5 text-xs font-bold uppercase tracking-widest transition-all"
                            >
                                <MessageSquare className="h-4 w-4" /> AI Assistant
                            </TabsTrigger>
                        </TabsList>

                        <TabsContent value="upload" className="mt-0 focus-visible:ring-0">
                            <UploadTab />
                        </TabsContent>

                        <TabsContent value="chat" className="mt-0 focus-visible:ring-0">
                            <ChatTab />
                        </TabsContent>
                    </Tabs>
                </motion.div>

                <motion.div
                    {...fadeUp}
                    transition={{ delay: 0.3, duration: 0.4 }}
                    className="space-y-6"
                >
                    <CostCalculator
                        config={{
                            aiProvider: "Vertex AI / Gemini 1.5",
                            vectorStore: "Neo4j Graph + Qdrant",
                            fileCount: 3,
                            estimatedTokens: 24500,
                        }}
                    />

                    <div className="surface-elevated rounded-xl border border-border p-6 space-y-5">
                        <div className="flex items-center gap-2 text-primary">
                            <Info className="h-4 w-4" />
                            <h3 className="text-sm font-bold tracking-tight">Guia Rápido</h3>
                        </div>

                        <div className="space-y-4">
                            <div className="flex gap-4">
                                <div className="h-6 w-6 shrink-0 rounded bg-muted flex items-center justify-center">
                                    <FolderOpen className="h-3 w-3 text-muted-foreground" />
                                </div>
                                <div>
                                    <h4 className="text-[11px] font-bold uppercase tracking-wider mb-1">Categorias</h4>
                                    <p className="text-[11px] text-muted-foreground leading-relaxed">Organize por tipo de documento (Leis, Processos, etc).</p>
                                </div>
                            </div>

                            <div className="flex gap-4">
                                <div className="h-6 w-6 shrink-0 rounded bg-muted flex items-center justify-center">
                                    <Tag className="h-3 w-3 text-muted-foreground" />
                                </div>
                                <div>
                                    <h4 className="text-[11px] font-bold uppercase tracking-wider mb-1">Collections</h4>
                                    <p className="text-[11px] text-muted-foreground leading-relaxed">Agrupamentos lógicos para restringir o contexto da IA.</p>
                                </div>
                            </div>

                            <div className="flex gap-4">
                                <div className="h-6 w-6 shrink-0 rounded bg-muted flex items-center justify-center">
                                    <MessageSquare className="h-3 w-3 text-muted-foreground" />
                                </div>
                                <div>
                                    <h4 className="text-[11px] font-bold uppercase tracking-wider mb-1">Rag-Chat</h4>
                                    <p className="text-[11px] text-muted-foreground leading-relaxed">Use "@collection" no chat para focar a busca.</p>
                                </div>
                            </div>
                        </div>

                        <div className="pt-2">
                            <div className="rounded-lg bg-primary/5 p-3 border border-primary/10 flex items-center justify-between">
                                <span className="text-[10px] font-mono text-primary font-bold">PRECISÃO ESTIMADA</span>
                                <span className="text-xs font-bold text-primary italic">98.4%</span>
                            </div>
                        </div>
                    </div>
                </motion.div>
            </div>
        </AppShell>
    );
}
