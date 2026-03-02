"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
    Link2,
    Layers,
    Plus,
    Trash2,
    FolderOpen,
    Square,
    Activity,
    Loader2,
    CheckCircle2,
    XCircle,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { cn } from "@/lib/utils";
import Link from "next/link";

export type UrlEntry = {
    id: string;
    url: string;
    dirName: string;
    depth: number;
    status: "pending" | "crawling" | "success" | "error";
    message?: string;
};

type Props = {
    urls: UrlEntry[];
    onAdd: (url: string, dirName: string, depth: number) => void;
    onRemove: (id: string) => void;
    onPurge: () => void;
    isCrawling: boolean;
    hasCrawled: boolean;
};

export default function UrlPanel({
    urls,
    onAdd,
    onRemove,
    onPurge,
    isCrawling,
    hasCrawled,
}: Props) {
    const [url, setUrl] = useState("");
    const [dirName, setDirName] = useState("");
    const [depth, setDepth] = useState(2);

    const handleAdd = () => {
        if (!url.trim()) return;
        onAdd(url.trim(), dirName.trim(), depth);
        setUrl("");
        setDirName("");
        setDepth(2);
    };

    return (
        <div className="surface-card rounded-lg p-5">
            <div className="mb-4 flex items-center gap-2">
                <Link2 className="h-4 w-4 text-primary" aria-hidden="true" />
                <h3 className="text-sm font-semibold" id="url-panel-heading">
                    URLs de Referência
                </h3>
            </div>

            <p className="mb-4 text-xs text-muted-foreground">
                Adicione URLs como raiz (ex: Código de Trânsito) e defina a
                profundidade de crawling.
            </p>

            {/* Row 1: URL input */}
            <div className="flex gap-2">
                <Input
                    placeholder="https://www.planalto.gov.br/..."
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    className="flex-1 bg-background text-sm"
                    onKeyDown={(e) => e.key === "Enter" && handleAdd()}
                    aria-label="URL de referência para crawling"
                />
                <div className="flex items-center gap-1.5 rounded-md border border-border bg-background px-3">
                    <Layers
                        className="h-3.5 w-3.5 text-muted-foreground"
                        aria-hidden="true"
                    />
                    <input
                        type="number"
                        min={0}
                        max={10}
                        value={depth}
                        onChange={(e) => setDepth(Number(e.target.value))}
                        className="w-10 bg-transparent text-center text-sm text-foreground outline-none"
                        aria-label="Profundidade de crawling"
                    />
                </div>
            </div>

            {/* Row 2: Dir name + Add button */}
            <div className="mt-2 flex gap-2">
                <div className="flex flex-1 items-center gap-1.5 rounded-md border border-border bg-background px-3">
                    <FolderOpen
                        className="h-3.5 w-3.5 shrink-0 text-muted-foreground"
                        aria-hidden="true"
                    />
                    <input
                        type="text"
                        placeholder="Nome do tópico (ex: Codigo de Transito Brasileiro)"
                        value={dirName}
                        onChange={(e) => setDirName(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && handleAdd()}
                        className="h-10 w-full bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none"
                        aria-label="Nome do diretório ou tópico"
                    />
                </div>
                <Button
                    size="sm"
                    onClick={handleAdd}
                    className="gap-1.5 shrink-0"
                    disabled={!url.trim() || isCrawling}
                >
                    {isCrawling ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                    ) : (
                        <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                    )}
                    {isCrawling ? "Enviando..." : "Adicionar"}
                </Button>
            </div>

            {/* URL list */}
            <AnimatePresence>
                {urls.length > 0 && (
                    <div className="mt-4 space-y-2">
                        {urls.map((entry, i) => (
                            <motion.div
                                key={entry.id}
                                initial={{ opacity: 0, y: -8 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{ opacity: 0, y: -8 }}
                                transition={{ delay: i * 0.05 }}
                                className="flex items-center justify-between rounded-md border border-border bg-background px-3 py-2"
                            >
                                <div className="flex items-center gap-3 overflow-hidden">
                                    {/* Status icon */}
                                    {entry.status === "crawling" && (
                                        <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-primary" />
                                    )}
                                    {entry.status === "success" && (
                                        <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-500" />
                                    )}
                                    {entry.status === "error" && (
                                        <XCircle className="h-3.5 w-3.5 shrink-0 text-red-500" />
                                    )}
                                    {entry.status === "pending" && (
                                        <div className="h-2 w-2 shrink-0 rounded-full bg-muted-foreground" />
                                    )}

                                    <span className="shrink-0 rounded bg-secondary px-2 py-0.5 font-mono text-[10px] text-secondary-foreground">
                                        depth: {entry.depth}
                                    </span>
                                    {entry.dirName && (
                                        <span className="shrink-0 rounded bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
                                            {entry.dirName}
                                        </span>
                                    )}
                                    <span className="truncate font-mono text-xs text-muted-foreground">
                                        {entry.url}
                                    </span>
                                </div>
                                <button
                                    onClick={() => onRemove(entry.id)}
                                    className="ml-2 shrink-0 text-muted-foreground transition-colors hover:text-destructive"
                                    aria-label={`Remover URL ${entry.url}`}
                                >
                                    <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                                </button>
                            </motion.div>
                        ))}
                    </div>
                )}
            </AnimatePresence>

            {/* Action buttons: Stop + Monitoring */}
            {hasCrawled && (
                <motion.div
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="mt-4 flex gap-2"
                >
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={onPurge}
                        className="gap-1.5 border-red-500/30 text-red-400 hover:bg-red-500/10 hover:border-red-500/50"
                    >
                        <Square className="h-3 w-3" aria-hidden="true" />
                        Parar Crawler
                    </Button>
                    <Link href="/monitoring" className="flex-1">
                        <Button
                            variant="outline"
                            size="sm"
                            className="w-full gap-1.5 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10 hover:border-emerald-500/50"
                        >
                            <Activity className="h-3.5 w-3.5" aria-hidden="true" />
                            Monitoramento
                        </Button>
                    </Link>
                </motion.div>
            )}
        </div>
    );
}
