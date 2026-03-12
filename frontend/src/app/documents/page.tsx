"use client";

import { motion } from "framer-motion";
import { Database, FileText, FolderOpen, Hash, Loader2, Tag } from "lucide-react";

import AppShell from "@/components/layout/AppShell";
import { useDocuments } from "@/hooks/use-documents";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

function formatSize(bytes: number) {
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
        return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(value?: string) {
    if (!value) {
        return "-";
    }

    return new Intl.DateTimeFormat("pt-BR", {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}

export default function DocumentsPage() {
    const { documents, isLoading, error } = useDocuments();

    return (
        <AppShell>
            <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="mb-8">
                <div className="mb-3 flex items-center gap-2">
                    <span className="h-px w-8 bg-primary" />
                    <span className="font-bold uppercase tracking-[0.3em] text-[10px] text-primary">Object Storage Index</span>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    Document<span className="text-primary italic">os</span>
                </h1>
                <p className="mt-4 max-w-3xl text-base leading-relaxed text-muted-foreground">
                    Visualize os arquivos enviados, o status de processamento e a vinculacao de cada item com collection e categoria.
                </p>
            </motion.div>

            <motion.div
                {...fadeUp}
                transition={{ delay: 0.1, duration: 0.4 }}
                className="surface-elevated rounded-xl border border-border"
            >
                <div className="flex items-center justify-between border-b border-border px-6 py-4">
                    <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                            <Database className="h-5 w-5" />
                        </div>
                        <div>
                            <h2 className="text-sm font-bold tracking-tight">Biblioteca de documentos</h2>
                            <p className="text-xs text-muted-foreground">{documents.length} documento(s) indexado(s)</p>
                        </div>
                    </div>
                </div>

                <div className="p-6">
                    {isLoading && (
                        <div className="flex min-h-64 items-center justify-center gap-3 text-sm text-muted-foreground">
                            <Loader2 className="h-4 w-4 animate-spin" />
                            Carregando documentos...
                        </div>
                    )}

                    {!isLoading && error && (
                        <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                            {error}
                        </div>
                    )}

                    {!isLoading && !error && documents.length === 0 && (
                        <div className="flex min-h-64 flex-col items-center justify-center text-center">
                            <FileText className="mb-4 h-10 w-10 text-muted-foreground" />
                            <h3 className="text-sm font-semibold">Nenhum documento enviado ainda</h3>
                            <p className="mt-2 max-w-md text-xs leading-relaxed text-muted-foreground">
                                Envie arquivos pelo dashboard para que eles aparecam aqui e possam ser auditados por collection, categoria e status.
                            </p>
                        </div>
                    )}

                    {!isLoading && !error && documents.length > 0 && (
                        <div className="grid gap-4">
                            {documents.map((document) => (
                                <div
                                    key={document.id}
                                    className="rounded-xl border border-border bg-background/70 p-4 shadow-card"
                                >
                                    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                                        <div className="min-w-0">
                                            <div className="flex items-center gap-2">
                                                <FileText className="h-4 w-4 text-primary" />
                                                <h3 className="truncate text-sm font-semibold">{document.title}</h3>
                                            </div>
                                            <p className="mt-2 break-all font-mono text-[11px] text-muted-foreground">
                                                {document.file_path || document.file_url}
                                            </p>
                                        </div>

                                        <div className="rounded-full border border-primary/20 bg-primary/10 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-primary">
                                            {document.status}
                                        </div>
                                    </div>

                                    <div className="mt-4 flex flex-wrap gap-2">
                                        <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                            <Hash className="h-3 w-3" />
                                            {formatSize(document.size)}
                                        </span>
                                        <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                            <FileText className="h-3 w-3" />
                                            {document.mime_type}
                                        </span>
                                        <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                            <FolderOpen className="h-3 w-3" />
                                            {document.collection || "Sem collection"}
                                        </span>
                                        <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                            <Tag className="h-3 w-3" />
                                            {document.category?.name || "Sem categoria"}
                                        </span>
                                    </div>

                                    <div className="mt-4 grid gap-2 border-t border-border pt-4 text-[11px] text-muted-foreground md:grid-cols-2">
                                        <p>Criado em: <span className="font-mono text-foreground">{formatDate(document.created_at)}</span></p>
                                        <p>Atualizado em: <span className="font-mono text-foreground">{formatDate(document.updated_at)}</span></p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </motion.div>
        </AppShell>
    );
}
