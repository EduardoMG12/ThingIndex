"use client";

import { Loader2, RotateCw, Siren } from "lucide-react";

import type { Document } from "@/lib/api/documents";
import type { DocumentOperationalHighlight } from "@/lib/document-observability";
import { getDocumentOperationalIssue } from "@/lib/document-state";
import { useI18n } from "@/lib/i18n/provider";

type OperationalWatchlistProps = {
    highlights: DocumentOperationalHighlight[];
    onOpenPreview: (document: Document) => void;
    onRetry: (document: Document) => void;
    retryingDocumentId: number | null;
};

export default function OperationalWatchlist({
    highlights,
    onOpenPreview,
    onRetry,
    retryingDocumentId,
}: OperationalWatchlistProps) {
    const { locale, t } = useI18n();
    const now = new Date();

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-5">
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                <Siren className="h-3.5 w-3.5 text-primary" />
                {t("documentsPage.observability.watchlistTitle")}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
                {t("documentsPage.observability.watchlistDescription")}
            </p>

            {highlights.length > 0 ? (
                <div className="mt-4 space-y-3">
                    {highlights.map((highlight) => {
                        const issue = getDocumentOperationalIssue(highlight.document, locale, now);
                        if (!issue) {
                            return null;
                        }

                        return (
                            <div
                                key={highlight.document.id}
                                className={`rounded-xl border p-4 ${buildIssueTone(highlight.kind)}`}
                            >
                                <div className="flex items-start justify-between gap-3">
                                    <div className="min-w-0">
                                        <p className="truncate text-sm font-semibold text-foreground">
                                            {highlight.document.title}
                                        </p>
                                        <p className="mt-1 text-xs text-muted-foreground">
                                            {issue.title}
                                        </p>
                                    </div>
                                    {issue.ageMinutes ? (
                                        <span className="shrink-0 rounded-full border border-border bg-background/80 px-2 py-1 text-[10px] font-mono text-muted-foreground">
                                            {t("documentsPage.observability.watchlistAge", { count: issue.ageMinutes })}
                                        </span>
                                    ) : null}
                                </div>

                                <p className="mt-3 text-xs text-foreground/90">
                                    {issue.detail || issue.description}
                                </p>
                                <p className="mt-3 text-[11px] text-muted-foreground">
                                    <span className="font-semibold text-foreground">
                                        {t("documentsPage.observability.anomalies.nextAction")}:
                                    </span>{" "}
                                    {issue.hint}
                                </p>
                                <p className="mt-3 text-[11px] text-muted-foreground">
                                    {highlight.document.collection || t("documentsPage.collectionFallback")} · {formatDateTime(highlight.document.updated_at, locale)}
                                </p>

                                <div className="mt-4 flex flex-wrap gap-2">
                                    <button
                                        type="button"
                                        onClick={() => onOpenPreview(highlight.document)}
                                        className="inline-flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/10 px-3 py-2 text-[11px] font-semibold text-primary"
                                    >
                                        {t("documentsPage.actions.openPanel")}
                                    </button>
                                    {highlight.document.status === "FAILED" ? (
                                        <button
                                            type="button"
                                            onClick={() => onRetry(highlight.document)}
                                            disabled={retryingDocumentId === highlight.document.id}
                                            className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-[11px] font-semibold text-foreground disabled:cursor-not-allowed disabled:opacity-60"
                                        >
                                            {retryingDocumentId === highlight.document.id ? (
                                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                            ) : (
                                                <RotateCw className="h-3.5 w-3.5" />
                                            )}
                                            {t("documentsPage.actions.retry")}
                                        </button>
                                    ) : null}
                                </div>
                            </div>
                        );
                    })}
                </div>
            ) : (
                <p className="mt-4 text-xs text-muted-foreground">
                    {t("documentsPage.observability.watchlistEmpty")}
                </p>
            )}
        </div>
    );
}

function buildIssueTone(kind: DocumentOperationalHighlight["kind"]) {
    switch (kind) {
        case "orphaned":
        case "dlq":
            return "border-destructive/20 bg-destructive/10";
        case "failed":
            return "border-amber-500/20 bg-amber-500/10";
        case "stalledProcessing":
        case "stalledQueue":
            return "border-sky-500/20 bg-sky-500/10";
        default:
            return "border-border bg-card/70";
    }
}

function formatDateTime(value: string | undefined, locale: string) {
    if (!value) {
        return "-";
    }

    return new Intl.DateTimeFormat(locale, {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}
