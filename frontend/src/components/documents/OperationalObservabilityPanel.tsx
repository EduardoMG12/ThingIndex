"use client";

import { AlertTriangle, Clock3, Loader2, RotateCw, Search, ShieldCheck, Siren } from "lucide-react";

import type { Document, DocumentMonitoringSummary } from "@/lib/api/documents";
import {
    createDocumentOperationalSnapshot,
    type DocumentOperationalFilter,
} from "@/lib/document-observability";
import { getDocumentOperationalIssue, type DocumentOperationalKind } from "@/lib/document-state";
import { useI18n } from "@/lib/i18n/provider";

type OperationalObservabilityPanelProps = {
    documents: Document[];
    summary: DocumentMonitoringSummary | null;
    activeFilter: DocumentOperationalFilter;
    searchTerm: string;
    onFilterChange: (filter: DocumentOperationalFilter) => void;
    onSearchTermChange: (value: string) => void;
    onOpenPreview: (document: Document) => void;
    onRetry: (document: Document) => void;
    retryingDocumentId: number | null;
};

export default function OperationalObservabilityPanel({
    documents,
    summary,
    activeFilter,
    searchTerm,
    onFilterChange,
    onSearchTermChange,
    onOpenPreview,
    onRetry,
    retryingDocumentId,
}: OperationalObservabilityPanelProps) {
    const { locale, t } = useI18n();
    const now = new Date();
    const snapshot = createDocumentOperationalSnapshot(documents, now);
    const processingCount = summary?.processing_documents ?? documents.filter((document) => document.status === "PROCESSING").length;
    const queuedCount =
        summary?.queued_documents ??
        documents.filter((document) => document.status === "QUEUED" && document.processing_stage !== "RETRY_SCHEDULED").length;

    const filters: Array<{ key: DocumentOperationalFilter; count: number; label: string }> = [
        { key: "all", count: snapshot.counts.total, label: t("documentsPage.observability.filters.all") },
        { key: "attention", count: snapshot.counts.attention, label: t("documentsPage.observability.filters.attention") },
        { key: "manual", count: snapshot.counts.manual, label: t("documentsPage.observability.filters.manual") },
        { key: "stalled", count: snapshot.counts.stalled, label: t("documentsPage.observability.filters.stalled") },
        { key: "retrying", count: snapshot.counts.retrying, label: t("documentsPage.observability.filters.retrying") },
        {
            key: "recentFailures",
            count: snapshot.counts.recentFailures,
            label: t("documentsPage.observability.filters.recentFailures"),
        },
        { key: "nominal", count: snapshot.counts.nominal, label: t("documentsPage.observability.filters.nominal") },
    ];

    return (
        <section className="mb-6 rounded-2xl border border-border bg-card/60 p-5">
            <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                <div className="max-w-2xl">
                    <p className="text-[10px] uppercase tracking-[0.28em] text-primary">
                        {t("documentsPage.observability.eyebrow")}
                    </p>
                    <h2 className="mt-2 text-lg font-semibold">{t("documentsPage.observability.title")}</h2>
                    <p className="mt-1 text-sm text-muted-foreground">
                        {t("documentsPage.observability.description")}
                    </p>
                </div>
                <div className={buildHealthTone(snapshot.health)}>
                    {renderHealthIcon(snapshot.health)}
                    <span>{t(`documentsPage.observability.health.${snapshot.health}`)}</span>
                </div>
            </div>

            <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.attention")}
                    value={snapshot.counts.attention}
                    detail={t("documentsPage.observability.metrics.attentionDetail")}
                    tone="border-destructive/20 bg-destructive/10"
                />
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.manual")}
                    value={snapshot.counts.manual}
                    detail={t("documentsPage.observability.metrics.manualDetail")}
                    tone="border-amber-500/20 bg-amber-500/10"
                />
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.stalled")}
                    value={snapshot.counts.stalled}
                    detail={t("documentsPage.observability.metrics.stalledDetail")}
                    tone="border-sky-500/20 bg-sky-500/10"
                />
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.recentFailures")}
                    value={snapshot.counts.recentFailures}
                    detail={t("documentsPage.observability.metrics.recentFailuresDetail")}
                    tone="border-border bg-background/60"
                />
            </div>

            <div className="mt-4 flex flex-wrap gap-2 text-[11px] text-muted-foreground">
                <span className="rounded-full border border-border bg-background/70 px-3 py-1">
                    {t("documentsPage.observability.snapshot.processing", { count: processingCount })}
                </span>
                <span className="rounded-full border border-border bg-background/70 px-3 py-1">
                    {t("documentsPage.observability.snapshot.queued", { count: queuedCount })}
                </span>
                <span className="rounded-full border border-border bg-background/70 px-3 py-1">
                    {t("documentsPage.observability.snapshot.retrying", { count: snapshot.counts.retrying })}
                </span>
                <span className="rounded-full border border-border bg-background/70 px-3 py-1">
                    {summary?.last_completed_at
                        ? t("documentsPage.observability.snapshot.lastCompleted", {
                            value: formatDateTime(summary.last_completed_at, locale),
                        })
                        : t("documentsPage.observability.snapshot.lastCompletedMissing")}
                </span>
            </div>

            <div className="mt-5 grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Search className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.observability.sliceTitle")}
                    </div>
                    <label htmlFor="documents-operational-search" className="mt-4 block text-xs font-medium text-foreground">
                        {t("documentsPage.observability.searchLabel")}
                    </label>
                    <input
                        id="documents-operational-search"
                        type="search"
                        value={searchTerm}
                        onChange={(event) => onSearchTermChange(event.target.value)}
                        placeholder={t("documentsPage.observability.searchPlaceholder")}
                        className="mt-2 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none transition focus:border-primary focus:ring-2 focus:ring-primary/20"
                    />

                    <div className="mt-4 flex flex-wrap gap-2">
                        {filters.map((filter) => (
                            <button
                                key={filter.key}
                                type="button"
                                onClick={() => onFilterChange(filter.key)}
                                aria-pressed={activeFilter === filter.key}
                                className={`rounded-full border px-3 py-2 text-[11px] font-semibold transition ${
                                    activeFilter === filter.key
                                        ? "border-primary/30 bg-primary/10 text-primary"
                                        : "border-border bg-background text-foreground hover:bg-muted"
                                }`}
                            >
                                {filter.label} · {filter.count}
                            </button>
                        ))}
                    </div>
                </div>

                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Siren className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.observability.watchlistTitle")}
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                        {t("documentsPage.observability.watchlistDescription")}
                    </p>

                    {snapshot.highlights.length > 0 ? (
                        <div className="mt-4 space-y-3">
                            {snapshot.highlights.map((highlight) => {
                                const issue = getDocumentOperationalIssue(highlight.document, locale, now);
                                if (!issue) {
                                    return null;
                                }

                                return (
                                    <div
                                        key={highlight.document.id}
                                        className={`rounded-lg border p-3 ${buildIssueTone(highlight.kind)}`}
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
                                        <p className="mt-2 text-xs text-foreground/90">
                                            {issue.detail || issue.description}
                                        </p>
                                        <p className="mt-2 text-[11px] text-muted-foreground">
                                            {highlight.document.collection || t("documentsPage.collectionFallback")} · {formatDateTime(highlight.document.updated_at, locale)}
                                        </p>
                                        <div className="mt-3 flex flex-wrap gap-2">
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
            </div>
        </section>
    );
}

function SummaryMetricCard({
    title,
    value,
    detail,
    tone,
}: {
    title: string;
    value: number;
    detail: string;
    tone: string;
}) {
    return (
        <div className={`rounded-xl border p-4 ${tone}`}>
            <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{title}</p>
            <p className="mt-2 text-lg font-semibold text-foreground">{value}</p>
            <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
        </div>
    );
}

function buildHealthTone(health: "nominal" | "attention" | "critical") {
    switch (health) {
        case "critical":
            return "inline-flex items-center gap-2 rounded-full border border-destructive/30 bg-destructive/10 px-4 py-2 text-sm font-semibold text-destructive";
        case "attention":
            return "inline-flex items-center gap-2 rounded-full border border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm font-semibold text-amber-700 dark:text-amber-300";
        default:
            return "inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-4 py-2 text-sm font-semibold text-emerald-700 dark:text-emerald-300";
    }
}

function renderHealthIcon(health: "nominal" | "attention" | "critical") {
    switch (health) {
        case "critical":
            return <AlertTriangle className="h-4 w-4" />;
        case "attention":
            return <Clock3 className="h-4 w-4" />;
        default:
            return <ShieldCheck className="h-4 w-4" />;
    }
}

function buildIssueTone(kind: DocumentOperationalKind) {
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
