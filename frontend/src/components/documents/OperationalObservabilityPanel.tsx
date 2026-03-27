"use client";

import { Search } from "lucide-react";

import OperationalAnomalyBoard from "@/components/documents/OperationalAnomalyBoard";
import OperationalExecutiveOverview from "@/components/documents/OperationalExecutiveOverview";
import OperationalWatchlist from "@/components/documents/OperationalWatchlist";
import type { Document, DocumentMonitoringSummary } from "@/lib/api/documents";
import {
    createDocumentOperationalOverview,
    type DocumentOperationalFilter,
} from "@/lib/document-observability";
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
    const { t } = useI18n();
    const now = new Date();
    const overview = createDocumentOperationalOverview(documents, now);
    const processingCount = summary?.processing_documents ?? documents.filter((document) => document.status === "PROCESSING").length;
    const queuedCount =
        summary?.queued_documents ??
        documents.filter((document) => document.status === "QUEUED" && document.processing_stage !== "RETRY_SCHEDULED").length;
    const isSliceActive = activeFilter !== "all" || searchTerm.trim() !== "";
    const primaryDocument =
        documents.find((document) => document.id === overview.primaryAction.documentIds[0]) ??
        overview.snapshot.highlights[0]?.document ??
        null;

    const filters: Array<{ key: DocumentOperationalFilter; count: number; label: string }> = [
        { key: "all", count: overview.snapshot.counts.total, label: t("documentsPage.observability.filters.all") },
        { key: "attention", count: overview.snapshot.counts.attention, label: t("documentsPage.observability.filters.attention") },
        { key: "manual", count: overview.snapshot.counts.manual, label: t("documentsPage.observability.filters.manual") },
        { key: "stalled", count: overview.snapshot.counts.stalled, label: t("documentsPage.observability.filters.stalled") },
        { key: "retrying", count: overview.snapshot.counts.retrying, label: t("documentsPage.observability.filters.retrying") },
        {
            key: "recentFailures",
            count: overview.snapshot.counts.recentFailures,
            label: t("documentsPage.observability.filters.recentFailures"),
        },
        { key: "nominal", count: overview.snapshot.counts.nominal, label: t("documentsPage.observability.filters.nominal") },
    ];

    const handleClearSlice = () => {
        onFilterChange("all");
        onSearchTermChange("");
    };

    const handleFocusFilter = (filter: DocumentOperationalFilter) => {
        onFilterChange(filter);
        onSearchTermChange("");
    };

    return (
        <section className="mb-6 rounded-2xl border border-border bg-card/60 p-5">
            <div className="max-w-2xl">
                <p className="text-[10px] uppercase tracking-[0.28em] text-primary">
                    {t("documentsPage.observability.eyebrow")}
                </p>
                <h2 className="mt-2 text-lg font-semibold">{t("documentsPage.observability.title")}</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                    {t("documentsPage.observability.description")}
                </p>
            </div>

            <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.attention")}
                    value={overview.snapshot.counts.attention}
                    detail={t("documentsPage.observability.metrics.attentionDetail")}
                    tone="border-destructive/20 bg-destructive/10"
                />
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.manual")}
                    value={overview.snapshot.counts.manual}
                    detail={t("documentsPage.observability.metrics.manualDetail")}
                    tone="border-amber-500/20 bg-amber-500/10"
                />
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.stalled")}
                    value={overview.snapshot.counts.stalled}
                    detail={t("documentsPage.observability.metrics.stalledDetail")}
                    tone="border-sky-500/20 bg-sky-500/10"
                />
                <SummaryMetricCard
                    title={t("documentsPage.observability.metrics.recentFailures")}
                    value={overview.snapshot.counts.recentFailures}
                    detail={t("documentsPage.observability.metrics.recentFailuresDetail")}
                    tone="border-border bg-background/60"
                />
            </div>

            <div className="mt-5">
                <OperationalExecutiveOverview
                    overview={overview}
                    processingCount={processingCount}
                    queuedCount={queuedCount}
                    lastCompletedAt={summary?.last_completed_at}
                    isSliceActive={isSliceActive}
                    onFocusFilter={handleFocusFilter}
                    onClearSlice={handleClearSlice}
                    onOpenPreview={onOpenPreview}
                    primaryDocument={primaryDocument}
                />
            </div>

            <div className="mt-5 rounded-2xl border border-border bg-background/60 p-5">
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

                    {isSliceActive ? (
                        <button
                            type="button"
                            onClick={handleClearSlice}
                            className="rounded-full border border-border bg-background px-3 py-2 text-[11px] font-semibold text-muted-foreground transition hover:bg-muted"
                        >
                            {t("documentsPage.observability.sliceClear")}
                        </button>
                    ) : null}
                </div>
            </div>

            <div className="mt-5 grid gap-4 xl:grid-cols-[0.92fr_1.08fr]">
                <OperationalAnomalyBoard
                    anomalies={overview.anomalies}
                    onFocusFilter={handleFocusFilter}
                />
                <OperationalWatchlist
                    highlights={overview.snapshot.highlights}
                    onOpenPreview={onOpenPreview}
                    onRetry={onRetry}
                    retryingDocumentId={retryingDocumentId}
                />
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
