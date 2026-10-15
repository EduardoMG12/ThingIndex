"use client";

import { useState } from "react";
import { FileText, RefreshCw, Trash2, Upload } from "lucide-react";

import { DangerConfirmModal } from "@/components/ui/danger-confirm-modal";
import type { DocumentSummary } from "@/lib/api/contracts/document-context";
import { formatDocumentStatus, tomDoStatus } from "@/lib/documents/status";
import { useI18n } from "@/lib/i18n/provider";

const TOM = {
    ok: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
    erro: "bg-red-500/15 text-red-700 dark:text-red-300",
    andamento: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
} as const;

export type DocumentosDaColecaoProps = {
    documentos: DocumentSummary[];
    onFocar: (documentId: number) => void;
    onReindexar: (documentId: number) => Promise<void>;
    onApagar: (documentId: number) => Promise<void>;
    onEnviar?: () => void;
};

export function DocumentosDaColecao({ documentos, onFocar, onReindexar, onApagar, onEnviar }: DocumentosDaColecaoProps) {
    const { t } = useI18n();
    const [apagando, setApagando] = useState<DocumentSummary | null>(null);
    const [confirmacao, setConfirmacao] = useState("");
    const [ocupado, setOcupado] = useState<number | null>(null);

    const executar = async (id: number, acao: (id: number) => Promise<void>) => {
        setOcupado(id);
        try {
            await acao(id);
        } finally {
            setOcupado(null);
        }
    };

    return (
        <div className="flex h-full flex-col">
            <div className="flex items-center justify-between gap-2 border-b border-border/70 px-3 py-2">
                <p className="text-xs text-muted-foreground">{t("workspace.documents.count", { count: documentos.length })}</p>
                {onEnviar ? (
                    <button
                        type="button"
                        onClick={onEnviar}
                        data-testid="enviar-documento"
                        className="flex items-center gap-1 rounded-md border border-primary/40 bg-primary/10 px-2 py-1 text-xs font-medium text-primary hover:bg-primary/15"
                    >
                        <Upload className="h-3.5 w-3.5" aria-hidden />
                        {t("workspace.documents.upload")}
                    </button>
                ) : null}
            </div>
            <ul className="min-h-0 flex-1 space-y-1 overflow-y-auto p-2" data-testid="lista-de-documentos">
                {documentos.length === 0 ? (
                    <li className="p-2 text-sm text-muted-foreground">{t("workspace.documents.empty")}</li>
                ) : null}
                {documentos.map((documento) => {
                    const tom = tomDoStatus(documento.status);
                    const emAcao = ocupado === documento.id;
                    return (
                        <li key={documento.id} data-testid={`documento-item-${documento.id}`} className="rounded-md border border-transparent px-2 py-1.5 hover:border-border hover:bg-muted/40">
                            <div className="flex items-start gap-2">
                                <button type="button" onClick={() => onFocar(documento.id)} className="min-w-0 flex-1 text-left" title={t("workspace.documents.focus")}>
                                    <span className="flex items-center gap-1.5 text-sm font-medium">
                                        <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden />
                                        <span className="truncate">{documento.title}</span>
                                    </span>
                                    <span className={`mt-1 inline-block rounded px-1.5 py-0.5 text-[10px] font-medium ${TOM[tom]}`} data-testid="status-do-documento">
                                        {formatDocumentStatus(documento.status, t)}
                                    </span>
                                </button>
                                <span className="flex shrink-0 items-center gap-0.5">
                                    <button
                                        type="button"
                                        onClick={() => void executar(documento.id, onReindexar)}
                                        disabled={emAcao}
                                        aria-label={t("workspace.documents.reindex")}
                                        title={t("workspace.documents.reindex")}
                                        className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-40"
                                    >
                                        <RefreshCw className={`h-3.5 w-3.5 ${emAcao ? "animate-spin" : ""}`} aria-hidden />
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => {
                                            setConfirmacao("");
                                            setApagando(documento);
                                        }}
                                        disabled={emAcao}
                                        aria-label={t("workspace.documents.delete")}
                                        title={t("workspace.documents.delete")}
                                        className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive disabled:opacity-40"
                                    >
                                        <Trash2 className="h-3.5 w-3.5" aria-hidden />
                                    </button>
                                </span>
                            </div>
                        </li>
                    );
                })}
            </ul>
            <DangerConfirmModal
                isOpen={apagando !== null}
                eyebrow={t("documentsPage.deleteModal.eyebrow")}
                title={t("documentsPage.deleteModal.documentTitle")}
                description={t("documentsPage.deleteModal.documentDescription")}
                confirmLabel={t("documentsPage.deleteModal.confirmLabel")}
                cancelLabel={t("common.dangerConfirm.cancelLabel")}
                closeLabel={t("common.dangerConfirm.closeLabel")}
                expectedValue={apagando?.title}
                confirmationValue={confirmacao}
                confirmationPrompt={apagando ? t("common.dangerConfirm.typeToConfirm", { value: apagando.title }) : null}
                inputLabel={t("common.dangerConfirm.inputLabel")}
                isPending={apagando !== null && ocupado === apagando.id}
                onConfirmationChange={setConfirmacao}
                onCancel={() => setApagando(null)}
                onConfirm={() => {
                    if (!apagando) {
                        return;
                    }
                    const id = apagando.id;
                    void executar(id, onApagar).finally(() => setApagando(null));
                }}
            />
        </div>
    );
}
