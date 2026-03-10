"use client";

import { useState, useCallback } from "react";
import { useDropzone } from "react-dropzone";
import { Upload, X, File, FileText, Globe, FileSpreadsheet, FolderOpen, Tag, Hash, ArrowRight } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type UploadedFile = {
    id: string;
    name: string;
    size: number;
    type: string;
    status: "pending" | "uploading" | "done" | "error";
    progress: number;
    category: string;
    collection: string;
};

const FILE_ICONS: Record<string, any> = {
    "application/pdf": FileText,
    "text/html": Globe,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": FileSpreadsheet,
};

const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / 1048576).toFixed(1) + " MB";
};

const UploadTab = () => {
    const [files, setFiles] = useState<UploadedFile[]>([]);
    const [isProcessing, setIsProcessing] = useState(false);
    const [defaultCategory, setDefaultCategory] = useState("Processos Jurídicos");
    const [defaultCollection, setDefaultCollection] = useState("Processo 0012345-XX");

    const onDrop = useCallback(
        (accepted: File[]) => {
            const mapped: UploadedFile[] = accepted.map((f) => ({
                id: crypto.randomUUID(),
                name: f.name,
                size: f.size,
                type: f.type,
                status: "pending",
                progress: 0,
                category: defaultCategory,
                collection: defaultCollection,
            }));
            setFiles((prev) => [...prev, ...mapped]);
        },
        [defaultCategory, defaultCollection]
    );

    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop,
        accept: {
            "application/pdf": [".pdf"],
            "text/html": [".html", ".htm"],
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": [".xlsx"],
        },
    });

    const handleProcessAll = () => {
        setIsProcessing(true);
        const interval = setInterval(() => {
            setFiles((prev) => {
                if (prev.every((f) => f.status === "done")) {
                    clearInterval(interval);
                    setIsProcessing(false);
                    return prev;
                }
                return prev.map((f) => {
                    if (f.status === "done") return f;
                    const newProgress = Math.min(f.progress + Math.random() * 30, 100);
                    return { ...f, status: newProgress >= 100 ? "done" : "uploading", progress: newProgress };
                });
            });
        }, 400);
    };

    return (
        <div className="surface-elevated rounded-lg overflow-hidden border border-border">
            <div className="border-b border-border p-4 bg-muted/20">
                <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1">
                        <Label className="flex items-center gap-1 text-[9px] uppercase tracking-wider text-muted-foreground">
                            <FolderOpen className="h-3 w-3" /> Categoria
                        </Label>
                        <Input
                            value={defaultCategory}
                            onChange={(e) => setDefaultCategory(e.target.value)}
                            className="h-7 bg-background font-mono text-xs"
                        />
                    </div>
                    <div className="space-y-1">
                        <Label className="flex items-center gap-1 text-[9px] uppercase tracking-wider text-muted-foreground">
                            <Tag className="h-3 w-3" /> Collection
                        </Label>
                        <Input
                            value={defaultCollection}
                            onChange={(e) => setDefaultCollection(e.target.value)}
                            className="h-7 bg-background font-mono text-xs"
                        />
                    </div>
                </div>
            </div>

            <div className="p-4 space-y-4">
                <div
                    {...getRootProps()}
                    className={`cursor-pointer rounded-lg border-2 border-dashed p-8 text-center transition-all ${isDragActive ? "border-primary bg-primary/5 shadow-inner" : "border-border hover:border-primary/40 hover:bg-muted/30"
                        }`}
                >
                    <input {...getInputProps()} />
                    <Upload className="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
                    <p className="text-sm font-medium">Arraste seus documentos jurídicos</p>
                    <p className="mt-1 text-xs text-muted-foreground">PDF, HTML ou Excel (Máx 20MB)</p>
                </div>

                <div className="space-y-2">
                    <AnimatePresence>
                        {files.map((f) => {
                            const Icon = FILE_ICONS[f.type] || File;
                            return (
                                <motion.div
                                    key={f.id}
                                    initial={{ opacity: 0, scale: 0.95 }}
                                    animate={{ opacity: 1, scale: 1 }}
                                    exit={{ opacity: 0, scale: 0.95 }}
                                    className="rounded-lg border border-border bg-background p-3 flex flex-col gap-2"
                                >
                                    <div className="flex items-center gap-3">
                                        <div className="h-8 w-8 rounded bg-primary/10 flex items-center justify-center shrink-0">
                                            <Icon className="h-4 w-4 text-primary" />
                                        </div>
                                        <div className="min-w-0 flex-1">
                                            <p className="truncate text-xs font-semibold">{f.name}</p>
                                            <p className="text-[10px] text-muted-foreground uppercase font-mono">{formatSize(f.size)}</p>
                                        </div>
                                        {f.status === "uploading" && (
                                            <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
                                                <motion.div className="h-full bg-primary" animate={{ width: `${f.progress}%` }} />
                                            </div>
                                        )}
                                        {f.status === "done" && <span className="text-xs font-bold text-emerald-500">PRONTO</span>}
                                        <button onClick={() => setFiles(fs => fs.filter(x => x.id !== f.id))} className="text-muted-foreground hover:text-destructive">
                                            <X className="h-4 w-4" />
                                        </button>
                                    </div>
                                    <div className="flex gap-2 pl-11">
                                        <span className="flex items-center gap-1 text-[9px] bg-muted px-2 py-0.5 rounded border border-border text-muted-foreground font-mono">
                                            <FolderOpen className="h-2.5 w-2.5" /> {f.category}
                                        </span>
                                        <span className="flex items-center gap-1 text-[9px] bg-muted px-2 py-0.5 rounded border border-border text-muted-foreground font-mono">
                                            <Hash className="h-2.5 w-2.5" /> {f.collection}
                                        </span>
                                    </div>
                                </motion.div>
                            );
                        })}
                    </AnimatePresence>
                </div>

                {files.length > 0 && (
                    <Button onClick={handleProcessAll} disabled={isProcessing} className="w-full gap-2">
                        {isProcessing ? <ArrowRight className="h-4 w-4 animate-pulse" /> : <ArrowRight className="h-4 w-4" />}
                        {isProcessing ? "Indexando no Grafo..." : `Processar ${files.length} Documentos`}
                    </Button>
                )}
            </div>
        </div>
    );
};

export default UploadTab;
