"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Link2, Layers, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { cn } from "@/lib/utils";

export type UrlEntry = {
    id: string;
    url: string;
    depth: number;
};

type Props = {
    urls: UrlEntry[];
    onAdd: (url: string, depth: number) => void;
    onRemove: (id: string) => void;
};

export default function UrlPanel({ urls, onAdd, onRemove }: Props) {
    const [url, setUrl] = useState("");
    const [depth, setDepth] = useState(2);

    const handleAdd = () => {
        if (!url.trim()) return;
        onAdd(url.trim(), depth);
        setUrl("");
        setDepth(2);
    };

    return (
        <div className="surface-card rounded-lg p-5">
            <div className="mb-4 flex items-center gap-2">
                <Link2 className="h-4 w-4 text-primary" />
                <h3 className="text-sm font-semibold">URLs de Referência</h3>
            </div>

            <p className="mb-4 text-xs text-muted-foreground">
                Adicione URLs como raiz (ex: Código de Trânsito) e defina a profundidade de crawling.
            </p>

            <div className="flex gap-2">
                <Input
                    placeholder="https://www.planalto.gov.br/..."
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    className="flex-1 bg-background text-sm"
                    onKeyDown={(e) => e.key === "Enter" && handleAdd()}
                />
                <div className="flex items-center gap-1.5 rounded-md border border-border bg-background px-3">
                    <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                    <input
                        type="number"
                        min={1}
                        max={10}
                        value={depth}
                        onChange={(e) => setDepth(Number(e.target.value))}
                        className="w-10 bg-transparent text-center text-sm text-foreground outline-none"
                    />
                </div>
                <Button size="sm" onClick={handleAdd} className="gap-1.5">
                    <Plus className="h-3.5 w-3.5" />
                    Adicionar
                </Button>
            </div>

            {urls.length > 0 && (
                <div className="mt-4 space-y-2">
                    {urls.map((entry, i) => (
                        <motion.div
                            key={entry.id}
                            initial={{ opacity: 0, y: -8 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: i * 0.05 }}
                            className="flex items-center justify-between rounded-md border border-border bg-background px-3 py-2"
                        >
                            <div className="flex items-center gap-3 overflow-hidden">
                                <span className="shrink-0 rounded bg-secondary px-2 py-0.5 font-mono text-[10px] text-secondary-foreground">
                                    depth: {entry.depth}
                                </span>
                                <span className="truncate font-mono text-xs text-muted-foreground">
                                    {entry.url}
                                </span>
                            </div>
                            <button
                                onClick={() => onRemove(entry.id)}
                                className="ml-2 shrink-0 text-muted-foreground transition-colors hover:text-destructive"
                            >
                                <Trash2 className="h-3.5 w-3.5" />
                            </button>
                        </motion.div>
                    ))}
                </div>
            )}
        </div>
    );
}
