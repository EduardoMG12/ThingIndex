"use client";

import { AnimatePresence, motion } from "framer-motion";
import { ComponentType } from "react";
import { File, FileSpreadsheet, FileText, FolderOpen, Globe, Hash, X } from "lucide-react";

import type { DocumentUploadFile } from "@/hooks/use-document-upload";
import { useI18n } from "@/lib/i18n/provider";

type UploadFileListProps = {
    files: DocumentUploadFile[];
    onRemove: (id: string) => void;
    isProcessing: boolean;
};

const FILE_ICONS: Record<string, ComponentType<{ className?: string }>> = {
    "application/pdf": FileText,
    "text/html": Globe,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": FileSpreadsheet,
};

const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / 1048576).toFixed(1) + " MB";
};

export default function UploadFileList({ files, onRemove, isProcessing }: UploadFileListProps) {
    const { t } = useI18n();

    if (files.length === 0) {
        return null;
    }

    return (
        <div className="space-y-2">
            <AnimatePresence>
                {files.map((file) => {
                    const Icon = FILE_ICONS[file.type] || File;
                    return (
                        <motion.div
                            key={file.id}
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
                                    <p className="truncate text-xs font-semibold">{file.name}</p>
                                    <p className="text-[10px] text-muted-foreground uppercase font-mono">{formatSize(file.size)}</p>
                                </div>
                                {file.status === "uploading" && (
                                    <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
                                        <motion.div className="h-full bg-primary" style={{ width: `${file.progress}%` }} />
                                    </div>
                                )}
                                {file.status === "done" && <span className="text-xs font-bold text-emerald-500">{t("dashboard.upload.ready")}</span>}
                                {file.status === "error" && <span className="text-xs font-bold text-destructive">{t("dashboard.upload.error")}</span>}
                                <button
                                    type="button"
                                    disabled={isProcessing}
                                    onClick={() => onRemove(file.id)}
                                    className="text-muted-foreground transition-colors hover:text-destructive disabled:cursor-not-allowed disabled:opacity-40"
                                >
                                    <X className="h-4 w-4" />
                                </button>
                            </div>
                            <div className="flex gap-2 pl-11">
                                <span className="flex items-center gap-1 text-[9px] bg-muted px-2 py-0.5 rounded border border-border text-muted-foreground font-mono">
                                    <FolderOpen className="h-2.5 w-2.5" /> {file.category || t("dashboard.upload.uncategorized")}
                                </span>
                                <span className="flex items-center gap-1 text-[9px] bg-muted px-2 py-0.5 rounded border border-border text-muted-foreground font-mono">
                                    <Hash className="h-2.5 w-2.5" /> {file.collection || t("dashboard.upload.collectionRequired")}
                                </span>
                            </div>
                            {file.error && (
                                <p className="pl-11 text-[10px] text-destructive">{file.error}</p>
                            )}
                        </motion.div>
                    );
                })}
            </AnimatePresence>
        </div>
    );
}
